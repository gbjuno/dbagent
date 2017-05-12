package main

import (
	"github.com/golang/glog"
	//	"reflect"
	"time"
)

func Duration(invocation time.Time, name string) {
	elapsed := time.Since(invocation)
	glog.Infof("func %s consume %s", name, elapsed)
}

func checkType(i interface{}) {
}
