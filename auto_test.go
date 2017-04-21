package main

import (
	"os"
	"os/exec"
	"testing"
)

func TestNewAutoAgent(t *testing.T) {
	var ins *AutoAgent
	ins = NewAutoAgent()
	if ins == nil {
		t.Fatal("can not get autoagent")
	}
}

/*
func TestAutoAgent_deployMongoIns_noBaseP(t *testing.T) {
	var ins *AutoAgent
	ins = NewAutoAgent()
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

func TestAutoAgent_deployMongoIns_BaseP(t *testing.T) {
	var ins *AutoAgent
	var cmd *exec.Cmd
	var conParam []string

	ins = NewAutoAgent()
	if ins == nil {
		t.Fatal("can not get autoagent")
	}

	var m MongoInstance = MongoInstance{"replTest", 27017, "/opt/data", "", "3.2.11"}
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
