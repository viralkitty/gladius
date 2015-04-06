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
		port        int
		bindingPort int
		err         error
	)

	port = 5050

	if os.Getenv("MASTER_PORT") != "" {
		port, err = strconv.Atoi(os.Getenv("MASTER_PORT"))

		if err != nil {
			port = 5050
		}
	}

	bindingPort = 8081

	if os.Getenv("SCHEDULER_PORT") != "" {
		port, err = strconv.Atoi(os.Getenv("SCHEDULER_PORT"))

		if err != nil {
			port = 5050
		}
	}

	schedulerTCPAddr := net.TCPAddr{
		IP:   net.ParseIP(os.Getenv("SCHEDULER_IP")),
		Port: bindingPort,
	}
	mesosTCPAddr := net.TCPAddr{
		IP:   net.ParseIP(os.Getenv("MASTER_PORT_5050_TCP_ADDR")),
		Port: port,
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
