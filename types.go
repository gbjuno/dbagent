package main

import (
	"sync"
	"time"
)

type Mongo struct {
	locker       sync.Mutex
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
	Status      string
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
	CREATING = "creating"
	STARTING = "starting"
	STOPPING = "stopping"
	DELETING = "deleting"
	RUNNING  = "running"
	STOPPED  = "stopped"
	DELETED  = "deleted"
	ERROR    = "error"
	NOP      = ""
)
