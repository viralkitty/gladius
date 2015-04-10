package main

import (
	"net"
	"os"
	"strconv"

	proto "github.com/gogo/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	sched "github.com/mesos/mesos-go/scheduler"
)

func NewSchedulerDriver() (*sched.MesosSchedulerDriver, error) {
	var (
		masterPort    int
		schedulerPort int
		err           error
	)

	masterPort = 5050

	if os.Getenv("MASTER_PORT") != "" {
		masterPort, err = strconv.Atoi(os.Getenv("MASTER_PORT"))

		if err != nil {
			masterPort = 5050
		}
	}

	schedulerPort = 5051

	if os.Getenv("SCHEDULER_PORT") != "" {
		schedulerPort, err = strconv.Atoi(os.Getenv("SCHEDULER_PORT"))

		if err != nil {
			schedulerPort = 5051
		}
	}

	schedulerTCPAddr := net.TCPAddr{
		IP:   net.ParseIP(os.Getenv("SCHEDULER_IP")),
		Port: schedulerPort,
	}
	mesosTCPAddr := net.TCPAddr{
		IP:   net.ParseIP(os.Getenv("MASTER_PORT_5050_TCP_ADDR")),
		Port: masterPort,
	}
	frameworkInfo := &mesos.FrameworkInfo{
		User: proto.String(""),
		Name: proto.String(frameworkName),
	}
	driverConfig := sched.DriverConfig{
		Scheduler:      NewScheduler(),
		Framework:      frameworkInfo,
		Master:         mesosTCPAddr.String(),
		BindingAddress: schedulerTCPAddr.IP,
		BindingPort:    uint16(schedulerTCPAddr.Port),
	}
	driver, err := sched.NewMesosSchedulerDriver(driverConfig)

	if err != nil {
		return nil, err
	}

	return driver, nil

}
