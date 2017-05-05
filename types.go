package main

import (
	"time"
)

type MongoInstance struct {
	Name         string
	BasePath     string
	Role         string
	Port         int
	MasterServer string
	CacheSizeMB  int
	Version      string
	Type         uint
	WantedStatus string
	NextOp       string
}

type MongoInstanceStatus struct {
	Name         string
	LastStatus   string
	Status       string
	Running      bool
	PrevOp       string
	CurrOp       string
	DataPath     string
	LastUpdate   time.Time
	UnderMonitor bool
}
