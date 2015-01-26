package main

import (
	mesos "github.com/mesos/mesos-go/mesosproto"
)

type Task struct {
	Cmd    string
	Output chan mesos.TaskStatus
}
