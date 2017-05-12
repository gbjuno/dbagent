package main

import (
	"fmt"
	"time"
	//	"os"
	//	"os/exec"
	"testing"
)

/*
func TestMongoAgent_deployMongoIns_noBaseP(t *testing.T) {
	var ins *MongoAgent
	ins = NewMongoAgent()
	if ins == nil {
		t.Fatal("can not get autoagent")
	}
	var m MongoInstance = MongoInstance{"replTest", 27017, "/opt/data", "", "3.0"}

	if err := ins.deployMongoIns(&m); err != nil {
		if err.Error() != "Mongo Deploy Failed" {
			t.Fatal(err)
		}
	}
}
*/
func initial() *MongoAgent {
	mongoAgent := NewMongoAgent()
	for i := 0; i < 5; i++ {
		mongoAgent.mongoMap[fmt.Sprintf("test%d", i)] = &Mongo{
			Name:        fmt.Sprintf("test%d", i),
			BasePath:    "/opt/data",
			Role:        "SingleDB",
			Port:        27000 + i,
			CacheSizeMB: 10240,
			Version:     "3.4.4",
			Type:        SingleDB,
			NextOp:      "CREATE",
		}
	}
	return mongoAgent
}

func Test_Handler(t *testing.T) {
	t.Logf("Test_handler")
	mongoAgent := NewMongoAgent()
	mongoAgent.mongoMap["test0"] = &Mongo{
		Name:        "test0",
		BasePath:    "/opt/data",
		Role:        "SingleDB",
		Port:        27000,
		CacheSizeMB: 10240,
		Version:     "3.2.11",
		Type:        SingleDB,
		Status:      CREATING,
	}
	t.Logf("get mongoAgent success")
	ins := mongoAgent.mongoMap["test0"]
	if err := mongoAgent.mongoMgr.GO_Handle(ins); err != nil {
		t.Fatal("create a mongo instance failed")
	}
	t.Logf("create mongo instance success")

	ins.Status = STOPPING
	if err := mongoAgent.mongoMgr.GO_Handle(ins); err != nil {
		t.Fatal("stop a mongo instance failed")
	}

	t.Logf("stop mongo instance success")

	time.Sleep(time.Duration(3) * time.Second)

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

/*
func Test_deployMongoIns_BaseP(t *testing.T) {
	var ins *MongoAgent
	var cmd *exec.Cmd
	var conParam []string

	ins = NewMongoAgent()
	if ins == nil {
		t.Fatal("can not get autoagent")
	}

	var m Mongo = Mongo{MongoInstance: MongoInstance{Name: "replTest", Port: 27017, BasePath: "/opt/data", Version: "3.2.11", Type: SingleDB}}

	os.Mkdir("/opt/data", os.ModeDir|0755)

	if err := ins.deployMongoIns(&m); err != nil {
		if err.Error() != "Mongo Deploy Failed" {
			t.Fatal(err)
		}
	}

	conParam = []string{"-H", "127.0.0.1:4321", "rm", "-f", "replTest"}
	cmd = exec.Command("docker", conParam...)
	cmd.Run()
	os.RemoveAll("/opt/data")
}
*/
