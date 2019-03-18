package zkha

import (
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

type callback func() error

var (
	startCb callback
	stopCb  callback
	logger  Logger
)

// ZKConn is the zk handle, it is available after `Run()` successfully return,
// and is closed once exit signal is accepted.
var ZKConn *zk.Conn

// RegisterStartFunc register the function to be run once this node is upgraded to be master.
func RegisterStartFunc(f callback) {
	startCb = f
}

// RegisterStopFunc register the function to be run once this node is downgraded from master.
// Note that you have to ensure your stop function returns before session timeout, otherwise,
// it just exit, so that to avoid to conflict caused by multiple master.
func RegisterStopFunc(f callback) {
	stopCb = f
}

func RegisterLogger(l Logger) {
	logger = l
}

func Run(servers []string, timeout time.Duration, znode string) error {
	sm, err := initSm(servers, timeout, znode)
	if err != nil {
		return err
	}

	for {
		sm.transit()
	}
}
