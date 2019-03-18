package zkha

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/looplab/fsm"
	"github.com/pkg/errors"
	"github.com/samuel/go-zookeeper/zk"
)

const (
	EVT_INIT_CONNECT       string = "init_connect"
	EVT_RACE_WIN           string = "race_win"
	EVT_RACE_LOSE          string = "race_lose"
	EVT_ZNODE_DELETE       string = "znode_delete"
	EVT_MASTER_DISCONNECT  string = "master_disconnect"
	EVT_MASTER_CONNECT     string = "master_connect"
	EVT_MASTER_EXPIRE      string = "master_expire"
	EVT_MASTER_TASK_STOP   string = "master_task_stop"
	EVT_WATCHER_DISCONNECT string = "watcher_disconnect"
	EVT_WATCHER_CONNECT    string = "watcher_connect"
	EVT_WATCHER_EXPIRE     string = "watcher_expire"
	EVT_EXIT               string = "exit"
)

type State interface {
	String() string
	Transit(sm *StateMachine)
}

type StateMachine struct {
	State
	*fsm.FSM
	logger         Logger
	exitCh         <-chan struct{}
	sessionTimeout time.Duration

	c         *zk.Conn
	defaultCh <-chan zk.Event
	existCh   <-chan zk.Event
	znode     string
}

func initSm(servers []string, timeout time.Duration, znode string) (sm *StateMachine, err error) {

	// connect to zk cluster
	c, defaultCh, err := zk.Connect(servers, timeout)
	if err != nil {
		return
	}
	ZKConn = c

	// register exit signal
	exitCh := make(chan struct{})
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-ch
		c.Close()
		exitCh <- struct{}{}
	}()

	fsM := fsm.NewFSM(
		init_state.String(),
		fsm.Events{
			{Name: EVT_INIT_CONNECT, Src: []string{init_state.String()}, Dst: race_state.String()},
			{Name: EVT_RACE_WIN, Src: []string{race_state.String()}, Dst: master_state.String()},
			{Name: EVT_RACE_LOSE, Src: []string{race_state.String()}, Dst: watch_state.String()},
			{Name: EVT_ZNODE_DELETE, Src: []string{watch_state.String()}, Dst: race_state.String()},
			{Name: EVT_MASTER_DISCONNECT, Src: []string{master_state.String()}, Dst: master_closing_state.String()},
			{Name: EVT_MASTER_TASK_STOP, Src: []string{master_closing_state.String()}, Dst: master_closed_state.String()},
			{Name: EVT_MASTER_CONNECT, Src: []string{master_closed_state.String()}, Dst: master_state.String()},
			{Name: EVT_MASTER_EXPIRE, Src: []string{master_closed_state.String()}, Dst: init_state.String()},
			{Name: EVT_WATCHER_DISCONNECT, Src: []string{watch_state.String()}, Dst: watch_closed_state.String()},
			{Name: EVT_WATCHER_CONNECT, Src: []string{watch_closed_state.String()}, Dst: watch_state.String()},
			{Name: EVT_WATCHER_EXPIRE, Src: []string{watch_closed_state.String()}, Dst: init_state.String()},
			{Name: EVT_EXIT, Src: []string{init_state.String(), race_state.String(), watch_closed_state.String(), master_closing_state.String(), master_closed_state.String()}, Dst: exit_state.String()},
		},
		fsm.Callbacks{
			"before_event": func(e *fsm.Event) {
				var err error
				switch e.Event {
				case EVT_INIT_CONNECT:
					err = sm.InitStateToRaceState()
				case EVT_RACE_WIN:
					err = sm.RaceStateToMasterState()
				case EVT_RACE_LOSE:
					err = sm.RaceStateToWatchState()
				case EVT_ZNODE_DELETE:
					err = sm.WatchStateToRaceState()
				case EVT_MASTER_DISCONNECT:
					err = sm.MasterStateToMasterClosingState()
				case EVT_MASTER_TASK_STOP:
					err = sm.MasterClosingStateToMasterClosedState()
				case EVT_MASTER_CONNECT:
					err = sm.MasterClosedStateToMasterState()
				case EVT_MASTER_EXPIRE:
					err = sm.MasterClosedStateToInitState()
				case EVT_WATCHER_DISCONNECT:
					err = sm.WatchStateToWatchClosedState()
				case EVT_WATCHER_CONNECT:
					err = sm.WatchClosedStateToWatchState()
				case EVT_WATCHER_EXPIRE:
					err = sm.WatchClosedStateToInitState()
				case EVT_EXIT:
					err = sm.ToExit()
				}
				if err != nil {
					err = errors.Wrapf(err, "state transition failed(triggered by %s)", e.Event)
					e.Cancel(err)
				}
			},
			"enter_state": func(e *fsm.Event) {
				switch e.Event {
				case EVT_INIT_CONNECT:
					sm.State = race_state
				case EVT_RACE_WIN:
					sm.State = master_state
				case EVT_RACE_LOSE:
					sm.State = watch_state
				case EVT_ZNODE_DELETE:
					sm.State = race_state
				case EVT_MASTER_DISCONNECT:
					sm.State = master_closing_state
				case EVT_MASTER_CONNECT:
					sm.State = master_state
				case EVT_MASTER_TASK_STOP:
					sm.State = master_closed_state
				case EVT_MASTER_EXPIRE:
					sm.State = init_state
				case EVT_WATCHER_DISCONNECT:
					sm.State = watch_closed_state
				case EVT_WATCHER_CONNECT:
					sm.State = watch_state
				case EVT_WATCHER_EXPIRE:
					sm.State = init_state
				case EVT_EXIT:
					sm.State = exit_state
				}
			},
		},
	)

	return &StateMachine{
		State:  init_state,
		FSM:    fsM,
		logger: logger,
		exitCh: exitCh,
		// FIXME: the `timeout` is not the real one. see this [issue](https://github.com/samuel/go-zookeeper/issues/210)
		sessionTimeout: timeout,
		c:              c,
		defaultCh:      defaultCh,
		znode:          znode,
	}, nil
}

func (sm *StateMachine) transit() {
	sm.State.Transit(sm)
}

func (sm *StateMachine) logStateTransition(target State) {
	sm.logger.INFOF("Transit from %s to %s", sm.Current(), target.String())
}

func (sm *StateMachine) InitStateToRaceState() error {
	sm.logStateTransition(race_state)
	return nil
}
func (sm *StateMachine) RaceStateToMasterState() error {
	sm.logStateTransition(master_state)
	return startCb()
}

func (sm *StateMachine) RaceStateToWatchState() error {
	sm.logStateTransition(watch_state)
	return nil
}

func (sm *StateMachine) WatchStateToRaceState() error {
	sm.logStateTransition(race_state)
	return nil
}
func (sm *StateMachine) MasterStateToMasterClosingState() error {
	sm.logStateTransition(master_closing_state)
	return nil
}
func (sm *StateMachine) MasterClosingStateToMasterClosedState() error {
	sm.logStateTransition(master_closed_state)
	return nil
}
func (sm *StateMachine) MasterClosedStateToMasterState() error {
	sm.logStateTransition(master_state)
	return startCb()
}
func (sm *StateMachine) MasterClosedStateToInitState() error {
	sm.logStateTransition(init_state)
	return nil
}
func (sm *StateMachine) WatchStateToWatchClosedState() error {
	sm.logStateTransition(watch_closed_state)
	return nil
}
func (sm *StateMachine) WatchClosedStateToWatchState() error {
	sm.logStateTransition(watch_state)
	return nil
}
func (sm *StateMachine) WatchClosedStateToInitState() error {
	sm.logStateTransition(init_state)
	return nil
}
func (sm *StateMachine) ToExit() error {
	sm.logStateTransition(exit_state)
	return nil
}

///////////////////////////////////////// STATE //////////////////////////////////////////////////

var (
	init_state           = &initState{}
	race_state           = &raceState{}
	master_state         = &masterState{}
	master_closing_state = &masterClosingState{}
	master_closed_state  = &masterClosedState{}
	watch_state          = &watchState{}
	watch_closed_state   = &watchClosedState{}
	exit_state           = &exitState{}
)

type initState struct{}

func (s *initState) String() string { return "init_state" }
func (s *initState) Transit(sm *StateMachine) {
	for {
		select {
		case evt := <-sm.defaultCh:
			if evt.State == zk.StateHasSession {
				sm.Event(EVT_INIT_CONNECT)
				return
			}
		case <-sm.exitCh:
			sm.Event(EVT_EXIT)
			return
		}
	}
}

type raceState struct{}

func (s *raceState) String() string { return "race_state" }
func (s *raceState) Transit(sm *StateMachine) {
	for {
		err := func() (err error) {
			eflag := int32(zk.FlagEphemeral)
			acl := zk.WorldACL(zk.PermAll)

			// check wether needs to exit
			select {
			case <-sm.exitCh:
				sm.Event(EVT_EXIT)
				return
			default:
			}

			// check existance first
			isExists, _, ch, err := sm.c.ExistsW(sm.znode)
			if err != nil {
			}
			if isExists {
				// save the exist watcher
				sm.existCh = ch
				sm.Event(EVT_RACE_LOSE)
				return
			}

			// try to create eznode
			paths := strings.Split(sm.znode, "/")
			var parentPath string
			for _, v := range paths[1 : len(paths)-1] {
				parentPath += "/" + v
				exist, _, err := sm.c.Exists(parentPath)
				if err != nil {
					return err
				}
				if !exist {
					_, err = sm.c.Create(parentPath, nil, 0, acl)
					if err != nil {
						return err
					}
				}
			}
			_, err = sm.c.Create(sm.znode, nil, eflag, acl)
			if err != nil {
				return
			}
			sm.Event(EVT_RACE_WIN)
			return
		}()
		if err != nil {
			logger.WARNF("%s. Retry later...", err)
			time.Sleep(3 * time.Second)
			continue
		}
		break
	}
}

type masterState struct{}

func (s *masterState) String() string { return "master_state" }
func (s *masterState) Transit(sm *StateMachine) {
	for {
		evt := <-sm.defaultCh
		if evt.State == zk.StateDisconnected {
			sm.Event(EVT_MASTER_DISCONNECT)
			return
		}
	}
}

type masterClosingState struct{}

func (s *masterClosingState) String() string { return "master_closing_state" }
func (s *masterClosingState) Transit(sm *StateMachine) {
	c := make(chan struct{})
	go func() {
		if err := stopCb(); err == nil {
			c <- struct{}{}
		}
	}()
	select {
	case <-c:
		sm.Event(EVT_MASTER_TASK_STOP)
		return
	case <-time.After(sm.sessionTimeout):
		logger.WARN("Task not stopped before session expiration, quit process...")
		sm.Event(EVT_EXIT)
		return
	}
}

type masterClosedState struct{}

func (s *masterClosedState) String() string { return "master_closed_state" }
func (s *masterClosedState) Transit(sm *StateMachine) {
	for {
		select {
		case evt := <-sm.defaultCh:
			switch evt.State {
			case zk.StateHasSession:
				sm.Event(EVT_MASTER_CONNECT)
				return
			case zk.StateExpired:
				sm.Event(EVT_MASTER_EXPIRE)
				return
			}
		case <-sm.exitCh:
			sm.Event(EVT_EXIT)
			return
		}
	}
}

type watchState struct{}

func (s *watchState) String() string { return "watch_state" }
func (s *watchState) Transit(sm *StateMachine) {
	if sm.existCh == nil {
		panic("existsence watcher not passed in")
	}
	for {
		select {
		case evt := <-sm.defaultCh:
			if evt.State == zk.StateDisconnected {
				sm.Event(EVT_WATCHER_DISCONNECT)
				return
			}
		case evt := <-sm.existCh:
			if evt.Type == zk.EventNodeDeleted {
				sm.Event(EVT_ZNODE_DELETE)
				return
			}
		}
	}
}

type watchClosedState struct{}

func (s *watchClosedState) String() string { return "watch_closed_state" }
func (s *watchClosedState) Transit(sm *StateMachine) {
	for {
		select {
		case evt := <-sm.defaultCh:
			switch evt.State {
			case zk.StateHasSession:
				sm.Event(EVT_WATCHER_CONNECT)
				return
			case zk.StateExpired:
				sm.Event(EVT_WATCHER_EXPIRE)
				return
			}
		case <-sm.exitCh:
			sm.Event(EVT_EXIT)
			return
		}
	}
}

type exitState struct{}

func (s *exitState) String() string { return "exit_state" }
func (s *exitState) Transit(sm *StateMachine) {
	// just quit
	os.Exit(0)
}
