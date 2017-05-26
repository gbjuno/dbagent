package main

import (
	//"fmt"
	"sync"
	"time"
	//	"os"
	//	"os/exec"
	"teego/pkg/api"
	"testing"
)

func Test_createMongo(t *testing.T) {
	t.Logf("Test_template")
	ins := &api.MongoInstance{
		TypeMeta: api.TypeMeta{
			Kind:       "MongoInstance",
			APIVersion: "v1",
		},
		ObjectMeta: api.ObjectMeta{
			Name:              "test01",
			Namespace:         "aaa",
			Labels:            map[string]string{"foo": "bar"},
			ResourceVersion:   "7215",
			CreationTimestamp: time.Now(),
		},
		Spec: api.MongoInstanceSpec{
			Role:         "master",
			Node:         "127.0.0.1",
			Port:         27001,
			Replication:  "",
			MasterServer: "",
			CacheSizeMB:  1024,
			Version:      "3.4.4",
		},
		Status: api.MongoInstanceStatus{
			Status:            CREATING,
			Running:           "initial",
			Message:           "",
			Pid:               "",
			BasePath:          "/opt",
			DataPath:          "",
			LastHeartbeatTime: time.Now(),
		},
	}

	n := NativeDeployment{}
	if err := n.createMongo(ins); err != nil {
		t.Fatalf("create mongo using native deploy failed, err", err)
	}
}

func Test_getMongoBinary(t *testing.T) {
	n := &NativeDeployment{}
	if err := n.getMongoBinary("3.4.4", "ubuntu"); err != nil {
		t.Fatalf("getMongoBinary failed, err: %v", err)
	}
	t.Fatalf("getMongoBinary")
}

func Test_NativeDeploy(t *testing.T) {
	t.Logf("Test_NativeDeploy")
	mongoAgent := NewMongoAgent()
	mongoAgent.mongoMap["test1"] = &api.MongoInstance{
		TypeMeta: api.TypeMeta{
			Kind:       "MongoInstance",
			APIVersion: "v1",
		},
		ObjectMeta: api.ObjectMeta{
			Name:              "test1",
			Namespace:         "aaa",
			Labels:            map[string]string{"foo": "bar"},
			ResourceVersion:   "7215",
			CreationTimestamp: time.Now(),
		},
		Spec: api.MongoInstanceSpec{
			Role:         "master",
			Node:         "127.0.0.1",
			Port:         27001,
			Replication:  "",
			MasterServer: "",
			CacheSizeMB:  1024,
			Version:      "3.4.4",
		},
		Status: api.MongoInstanceStatus{
			Status:            CREATING,
			Running:           "initial",
			Message:           "",
			Pid:               "",
			BasePath:          "/opt/data",
			DataPath:          "",
			LastHeartbeatTime: time.Now(),
		},
	}
	mongoAgent.mapLock["test1"] = &sync.Mutex{}
	t.Logf("get mongoAgent success, mongoAgent: %v", mongoAgent)
	ins := mongoAgent.mongoMap["test1"]
	if err := mongoAgent.mongoMgr.GO_Handle(ins); err != nil {
		t.Fatal("create a mongo instance failed")
	}
	t.Logf("create mongo instance success")
	time.Sleep(time.Duration(30) * time.Second)

	ins.Status.Status = STOPPING
	if err := mongoAgent.mongoMgr.GO_Handle(ins); err != nil {
		t.Fatal("stop a mongo instance failed")
	}

	t.Logf("stop mongo instance success")

	time.Sleep(time.Duration(30) * time.Second)

	ins.Status.Status = STARTING
	if err := mongoAgent.mongoMgr.GO_Handle(ins); err != nil {
		t.Fatal("start a mongo instance failed")
	}
	t.Logf("start mongo instance success")

	ins.Status.Status = DELETING
	if err := mongoAgent.mongoMgr.GO_Handle(ins); err != nil {
		t.Fatal("delete a mongo instance failed")
	}
	t.Fatalf("delete mongo instance success")
}
