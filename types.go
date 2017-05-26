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

	Running     bool
	Status      string
	DataPath    string
	ContainerID string
	LastUpdate  time.Time
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
