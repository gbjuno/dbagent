package main

import (
	//"fmt"
	"time"
	//	"os"
	//	"os/exec"
	"testing"
)

func Test_createMongo(t *testing.T) {
	t.Logf("Test_template")
	ins := &Mongo{
		Name:        "test0",
		BasePath:    "/opt/data",
		Role:        "SingleDB",
		Port:        27000,
		CacheSizeMB: 10240,
		Version:     "3.2.11",
		Type:        SingleDB,
		Status:      CREATING,
	}

	n := NativeDeployment{}
	if err := n.createMongo(ins); err != nil {
		t.Fatalf("create mongo using native deploy failed, err", err)
	}
}

func Test_getMongoBinary(t *testing.T) {
	n := &NativeDeployment{}
	if err := n.getMongoBinary("3.4.4", "centos"); err != nil {
		t.Fatalf("getMongoBinary failed, err: %v", err)
	}
	t.Fatalf("getMongoBinary")
}

func Test_NativeDeploy(t *testing.T) {
	t.Logf("Test_NativeDeploy")
	mongoAgent := NewMongoAgent()
	mongoAgent.mongoMap["test0"] = &Mongo{
		Name:        "test0",
		BasePath:    "/opt/data",
		Role:        "SingleDB",
		Port:        27000,
		CacheSizeMB: 10240,
		Version:     "3.4.4",
		Type:        SingleDB,
		Status:      CREATING,
	}
	t.Logf("get mongoAgent success")
	ins := mongoAgent.mongoMap["test0"]
	if err := mongoAgent.mongoMgr.GO_Handle(ins); err != nil {
		t.Fatal("create a mongo instance failed")
	}
	t.Logf("create mongo instance success")
	time.Sleep(time.Duration(30) * time.Second)

	ins.Status = STOPPING
	if err := mongoAgent.mongoMgr.GO_Handle(ins); err != nil {
		t.Fatal("stop a mongo instance failed")
	}

	t.Logf("stop mongo instance success")

	time.Sleep(time.Duration(30) * time.Second)

	ins.Status = STARTING
	if err := mongoAgent.mongoMgr.GO_Handle(ins); err != nil {
		t.Fatal("start a mongo instance failed")
	}
	t.Logf("start mongo instance success")

	ins.Status = DELETING
	if err := mongoAgent.mongoMgr.GO_Handle(ins); err != nil {
		t.Fatal("delete a mongo instance failed")
	}
	t.Fatalf("delete mongo instance success")
}
