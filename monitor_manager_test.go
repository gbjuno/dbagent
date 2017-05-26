package main

import (
	//"os"
	//"os/exec"
	//"golang.org/x/net/context"
	"testing"
	"time"
)

func Test_checkInsDetail(t *testing.T) {
	mongoAgent := initial()
	ins := mongoAgent.mongoMap["test1"]
	ins.Status.Status = CREATING
	if err := mongoAgent.mongoMgr.GO_Handle(ins); err != nil {
		t.Fatal("create a mongo instance failed")
	}
	t.Logf("create mongo instance success")

	time.Sleep(time.Duration(3) * time.Second)

	//insNameCh := make(chan string)
	//mongoAgent.monitorMgr.checkInsDetail(context.Background(), insNameCh)

	mongoAgent.monitorMgr.getMongoStatus(ins)
	t.Fatal("monitor success")
}
