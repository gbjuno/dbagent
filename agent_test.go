package main

import (
	"os"
	"os/exec"
	"testing"
)

func TestNewMongoAgent(t *testing.T) {
	var ins *MongoAgent
	ins = NewMongoAgent()
	if ins == nil {
		t.Fatal("can not get autoagent")
	}
}
