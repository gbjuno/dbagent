package main

import (
	"time"
)

type Mongo struct {
	Name         string
	BasePath     string
	Role         string
	Port         int
	MasterServer string
	CacheSizeMB  int
	Version      string
	Type         uint
	NextOp       string

	Running     bool
	Created     bool
	Deleted     bool
	ValidOp     bool
	PrevOp      string
	CurrOp      string
	DataPath    string
	ContainerID string
	LastUpdate  time.Time
}

type MongoInstance struct {
	Name         string
	BasePath     string
	Role         string
	Port         int
	MasterServer string
	CacheSizeMB  int
	Version      string
	Type         uint
	NextOp       string
}

type MongoInstanceStatus struct {
	Name       string
	Running    bool
	Created    bool
	Deleted    bool
	PrevOp     string
	CurrOp     string
	DataPath   string
	LastUpdate time.Time
}

const (
	CREATE = "CREATE"
	START  = "START"
	STOP   = "STOP"
	DELETE = "DELETE"
	NOP    = ""
)
