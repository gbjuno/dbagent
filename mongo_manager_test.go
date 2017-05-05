package main

import (
	"os"
	"os/exec"
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
