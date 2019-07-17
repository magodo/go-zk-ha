package main

import (
	"log"
	"strings"
	"time"

	"github.com/magodo/go-zk-ha/zkha"
)

func main() {
	service, err := zkha.NewService(
		zkha.ZkServers(strings.Split("172.21.0.1:2181,172.21.0.1:2182,172.21.0.1:2183", ",")),
		zkha.NodeExpiry(3*time.Second),
		zkha.LockZNode("/foo"),
		zkha.StartFunc(func() error {
			log.Println("start work")
			return nil
		}),
		zkha.StopFunc(func() error {
			log.Println("start to stop work...")
			time.Sleep(time.Second)
			log.Println("finish to stop work...")
			return nil
		}),
		zkha.GenNodeDataFunc(func() []byte {
			return []byte("Hello")
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
