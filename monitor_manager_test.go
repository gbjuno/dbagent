package main

import (
	//"os"
	//"os/exec"
	"golang.org/x/net/context"
	"testing"
	"time"
)

func Test_checkInsDetail(t *testing.T) {
	mongoAgent := initial()
	ins := mongoAgent.mongoMap["test0"]
	if err := mongoAgent.mongoMgr.GO_Handle(ins); err != nil {
		t.Fatal("create a mongo instance failed")
	}
	t.Logf("create mongo instance success")

	ins.NextOp = "START"
	if err := mongoAgent.mongoMgr.GO_Handle(ins); err != nil {
		t.Fatal("start a mongo instance failed")
	}

	t.Logf("start mongo instance success")
	time.Sleep(time.Duration(3) * time.Second)

	insNameCh := make(chan string)

	go mongoAgent.monitorMgr.checkInsDetail(context.Background(), insNameCh)
	insNameCh <- ins.Name
	t.Fatal("monitor success")
}
