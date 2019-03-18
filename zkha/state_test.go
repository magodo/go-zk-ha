package zkha

import (
	"log"
	"strings"
	"testing"
	"time"

	"github.com/magodo/go-zk-ha/zkha"
)

func TestFoo(t *testing.T) {
	zkha.RegisterStartFunc(func() error {
		log.Println("start work")
		return nil
	})

	zkha.RegisterStopFunc(func() error {
		log.Println("start to stop work...")
		time.Sleep(time.Second * 5)
		log.Println("finish to stop work...")
		return nil
	})

	err := zkha.Run(strings.Split("172.18.0.4:2181,172.18.0.2:2181,172.18.0.3:2181", ","), 3*time.Second, "/bar")
	if err != nil {
		t.Fatal(err)
	}
}
