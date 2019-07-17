package zkha

import (
	"errors"
	"time"
)

type Options struct {
	ZkServers       []string
	NodeExpiry      time.Duration
	LockZNode       string
	StartFunc       func() error
	StopFunc        func() error
	GenNodeDataFunc func() []byte
	Logger          ILogger
}

type Option func(*Options)

func defaultOptions() Options {
	return Options{
		ZkServers:       nil,
		NodeExpiry:      5 * time.Second,
		LockZNode:       "",
		StartFunc:       nil,
		StopFunc:        nil,
		GenNodeDataFunc: nil,
		Logger:          &defaultLogger{},
	}
}

// ZkServers register the zk server addresses
func ZkServers(servers []string) Option {
	return func(o *Options) {
		o.ZkServers = servers
	}
}

// NodeExpiry set the ephemeral node expiry
func NodeExpiry(d time.Duration) Option {
	return func(o *Options) {
		o.NodeExpiry = d
	}
}

// LockZNode set the path of ephemeral znode
func LockZNode(path string) Option {
	return func(o *Options) {
		o.LockZNode = path
	}
}

// StartFunc register the function to be run once this node is upgraded to be master.
func StartFunc(f func() error) Option {
	return func(o *Options) {
		o.StartFunc = f
	}
}

// StopFunc register the function to be run once this node is downgraded from master.
// Note that you have to ensure your stop function returns before session timeout, otherwise,
// it just exit, so that to avoid to conflict caused by multiple master.
func StopFunc(f func() error) Option {
	return func(o *Options) {
		o.StopFunc = f
	}
}

// GenNodeDataFunc register the function which will generate and set znode content once it
// win master race.
func GenNodeDataFunc(f func() []byte) Option {
	return func(o *Options) {
		o.GenNodeDataFunc = f
	}
}

// Logger register logger used internally
func Logger(l ILogger) Option {
	return func(o *Options) {
		o.Logger = l
	}
}

func (o *Options) Err() error {
	switch {
	case o.LockZNode == "":
		return errors.New(`"LockZNode" not specified`)
	case o.ZkServers == nil:
		return errors.New(`"ZkServers" not specified`)
	}
	return nil
}
