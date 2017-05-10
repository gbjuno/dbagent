package main

import (
	//"fmt"
	//"time"
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
