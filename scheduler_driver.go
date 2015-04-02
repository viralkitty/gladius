package main

import (
	proto "github.com/gogo/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	sched "github.com/mesos/mesos-go/scheduler"
)

func NewSchedulerDriver() (*sched.MesosSchedulerDriver, error) {
	driver, err := sched.NewMesosSchedulerDriver(sched.DriverConfig{
		Scheduler: NewScheduler(),
		Framework: &mesos.FrameworkInfo{
			User: proto.String(""),
			Name: proto.String(frameworkName),
		},
		Master:           mesosMaster,
		HostnameOverride: "gladius",
		BindingAddress:   schedulerIP,
		BindingPort:      uint16(schedulerPort),
	})

	if err != nil {
		return nil, err
	}

	return driver, nil

}
