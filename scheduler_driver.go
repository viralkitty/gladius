package main

import (
	"net"
	"os"

	proto "github.com/gogo/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	sched "github.com/mesos/mesos-go/scheduler"
)

func NewSchedulerDriver() (*sched.MesosSchedulerDriver, error) {
	mesosTCPAddr := net.TCPAddr{
		IP:   net.ParseIP(os.Getenv("MASTER_PORT_5050_TCP_ADDR")),
		Port: 5050,
	}

	frameworkInfo := &mesos.FrameworkInfo{
		User: proto.String(""),
		Name: proto.String(frameworkName),
	}

	driverConfig := sched.DriverConfig{
		Scheduler: NewScheduler(),
		Framework: frameworkInfo,
		Master:    mesosTCPAddr.String(),
	}

	driver, err := sched.NewMesosSchedulerDriver(driverConfig)

	if err != nil {
		return nil, err
	}

	return driver, nil

}
