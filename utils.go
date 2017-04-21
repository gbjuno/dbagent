package main

import (
	"github.com/golang/glog"
	"time"
)

func Duration(invocation time.Time, name string) {
	elapsed := time.Since(invocation)
	glog.Infof("func %s consume %s", name, elapsed)
}
