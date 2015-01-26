package main

import (
	mesos "github.com/mesos/mesos-go/mesosproto"
	"math/rand"
	"strconv"
)

type Task struct {
	Id      string            `json:"id,omitempty"`
	Cmd     string            `json:"cmd,omitempty"`
	Build   *Build            `json:"-"`
	BuildId string            `json:"buildId,omitempty"`
	Status  *mesos.TaskStatus `json:"status,omitempty"`
}

func NewTask(cmd string) *Task {
	return &Task{
		Id:  strconv.Itoa(rand.Int()),
		Cmd: cmd,
	}
}
