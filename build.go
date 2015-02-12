package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/garyburd/redigo/redis"
	mesos "github.com/mesos/mesos-go/mesosproto"
)

const (
	dockerRegistry = "docker.corp.adobe.com"
	baseImage      = "docker.corp.adobe.com/typekit/bundler-typekit"
)

type Build struct {
	Id               string                 `json:"id,omitempty"`
	App              string                 `json:"app,omitempty"`
	Branch           string                 `json:"branch,omitempty"`
	Log              string                 `json:"log,omitempty"`
	State            string                 `json:"state,omitempty"`
	Container        *docker.Container      `json:"-"`
	Scheduler        *Scheduler             `json:"-"`
	TaskStatusesChan chan *mesos.TaskStatus `json:"-"`
	Tasks            []*Task                `json:"tasks,omitempty"`
}

func NewBuild(scheduler *Scheduler) *Build {
	// TODO: Dynamically generate list of tasks

	return &Build{
		Scheduler: scheduler,
		Id:        strconv.Itoa(rand.Int()),
		Tasks: []*Task{
			NewTask("rspec spec/api --no-color"),
			NewTask("rspec spec/config --no-color"),
			NewTask("rspec spec/controllers --no-color"),
			NewTask("rspec spec/helpers --no-color"),
			NewTask("rspec spec/integration --no-color"),
			NewTask("rspec spec/lib --no-color"),
			NewTask("rspec spec/mails --no-color"),
			NewTask("rspec spec/racks --no-color"),
			NewTask("rspec spec/requests --no-color"),
			NewTask("rspec spec/routing --no-color"),
			NewTask("cucumber --profile=default --no-color --format=progress features/accounts"),
			NewTask("cucumber --profile=default --no-color --format=progress features/auth"),
			NewTask("cucumber --profile=default --no-color --format=progress features/api"),
			NewTask("cucumber --profile=default --no-color --format=progress features/web"),
			NewTask("cucumber --profile=default --no-color --format=progress features/ccm"),
			NewTask("cucumber --profile=licensing --no-color --format=progress features/licensing"),
			NewTask("cucumber --profile=mails --no-color --format=progress features/mails"),
		},
		TaskStatusesChan: make(chan *mesos.TaskStatus),
	}
}

func AllBuilds() []Build {
	var builds []Build

	conn := pool.Get()
	prefix := "pugio:builds:"
	keys, err := redis.Strings(conn.Do("KEYS", fmt.Sprintf("%s*", prefix)))

	defer conn.Close()

	if err != nil {
		log.Printf("Could not get keys: %s", err)
		return nil
	}

	for _, key := range keys {
		log.Printf("found key: %s", key)

		var b *Build
		var buildJsonBytes []byte

		buildJsonBytes, err = redis.Bytes(conn.Do("GET", key))

		if err != nil {
			log.Printf("Could not get key: %s", key)
			continue
		}

		err = json.Unmarshal(buildJsonBytes, &b)

		if err != nil {
			log.Printf("Could not unmarshal object: %v", err)
			return nil
		}

		builds = append(builds, *b)
	}

	return builds
}

func (b *Build) Build() {
	var err error

	b.Save()

	err = b.pullImage()

	if err != nil {
		b.SaveAndHandleError(err)
		return
	}

	err = b.createContainer()

	if err != nil {
		b.SaveAndHandleError(err)
		return
	}

	for {
		err = b.startContainer()

		if err != nil {
			b.SaveAndHandleError(err)
			continue
		}

		err = b.waitContainer()

		if err != nil {
			b.SaveAndHandleError(err)
			continue
		}

		break
	}

	err = b.commitContainer()

	if err != nil {
		b.SaveAndHandleError(err)
		return
	}

	err = b.removeContainer()

	if err != nil {
		b.SaveAndHandleError(err)
		return
	}

	err = b.pushImage()

	if err != nil {
		b.SaveAndHandleError(err)
		return
	}

	err = b.launchTasks()

	if err != nil {
		b.SaveAndHandleError(err)
		return
	}
}

func (b *Build) createContainer() error {
	log.Printf("Creating container")

	createOpts := docker.CreateContainerOptions{
		Config: &docker.Config{
			Tty:          true,
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			WorkingDir:   "/",
			Image:        baseImage,
			Entrypoint:   []string{"sh"},
			Cmd:          []string{"-c", b.CloneCmd()},
		},
	}
	var err error
	b.Container, err = dockerCli.CreateContainer(createOpts)

	if err != nil {
		log.Printf("Could not create container")
		log.Printf(err.Error())
		return err
	}

	log.Printf("%+v", b.Container)

	return nil
}

func (b *Build) startContainer() error {
	log.Printf("Starting container")

	sshKey := "/root/.ssh/id_rsa"

	if os.Getenv("SSH_KEY") != "" {
		sshKey = os.Getenv("SSH_KEY")
	}

	err := dockerCli.StartContainer(b.Container.ID, &docker.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/root/.ssh/id_rsa", sshKey),
		},
	})

	if err != nil {
		log.Printf("Could not start container")
		log.Printf(err.Error())
		return err
	}

	return nil
}

func (b *Build) waitContainer() error {
	log.Printf("Waiting for container")

	status, err := dockerCli.WaitContainer(b.Container.ID)

	if err != nil {
		log.Printf("Could not wait for container")
		log.Printf(err.Error())
		return err
	}

	if status != 0 {
		msg := "Clone or bundle failed"

		log.Printf(msg)
		return errors.New(msg)
	}

	return nil
}

func (b *Build) pullImage() error {
	done := make(chan bool)
	timeout := time.After(5 * time.Minute)
	auth := docker.AuthConfiguration{}
	opts := docker.PullImageOptions{
		Repository: baseImage,
		Registry:   dockerRegistry,
	}

	go func() {
		for {
			if dockerCli.PullImage(opts, auth) != nil {
				continue
			}

			done <- true
			break
		}
	}()

	select {
	case <-done:
		return nil
	case <-timeout:
		return errors.New("Could not pull bundler typekit image")
	}
}

func (b *Build) commitContainer() error {
	log.Printf("Commiting container")

	commitOpts := docker.CommitContainerOptions{
		Container:  b.Container.ID,
		Repository: b.FullImgName(),
		Tag:        b.Id,
	}

	_, err := dockerCli.CommitContainer(commitOpts)

	if err != nil {
		log.Printf("Could not commit container")
		log.Printf(err.Error())
		return err
	}

	return nil
}

func (b *Build) removeContainer() error {
	log.Printf("Removing container")

	removeOpts := docker.RemoveContainerOptions{
		ID:    b.Container.ID,
		Force: true,
	}

	err := dockerCli.RemoveContainer(removeOpts)

	if err != nil {
		log.Printf("Could not remove the container")
		log.Printf(err.Error())
		return err
	}

	return nil
}

func (b *Build) pushImage() error {
	var buf bytes.Buffer

	successMsg := "Image successfully pushed"
	errorMsg := "Timed out pushing image"
	done := make(chan bool)
	timeout := time.After(5 * time.Minute)
	auth := docker.AuthConfiguration{}
	opts := docker.PushImageOptions{
		Name:         b.FullImgName(),
		OutputStream: &buf,
		Tag:          b.Id,
	}

	go func() {
		for {
			if dockerCli.PushImage(opts, auth) != nil {
				continue
			}

			if strings.Contains(buf.String(), successMsg) != true {
				continue
			}

			done <- true
			break
		}
	}()

	select {
	case <-done:
		return nil
	case <-timeout:
		return errors.New(errorMsg)
	}
}

func (b *Build) launchTasks() error {
	go b.taskStatusLoop()

	for _, task := range b.Tasks {
		go func(t *Task) {
			log.Printf("throwing task into chan: %+v", t)
			t.Build = b
			t.BuildId = b.Id
			tasks <- t
		}(task)
	}

	return nil
}

func (b *Build) Save() error {
	log.Printf("Persisting in Redis")

	buildJson, err := json.Marshal(b)

	if err != nil {
		log.Printf("json error: %+v", err)
		return err
	}

	conn := pool.Get()
	defer conn.Close()

	_, err = conn.Do("SET", b.RedisKey(), buildJson)

	if err != nil {
		log.Printf("problem setting key: %v", err)
		return err
	}

	return nil
}

func (b *Build) taskStatusLoop() {
	for finishedTasks := 0; finishedTasks < len(b.Tasks); {
		select {
		case taskStatus := <-b.TaskStatusesChan:
			state := taskStatus.GetState()
			taskId := taskStatus.TaskId.GetValue()

			log.Printf("Task %s is in the %s state", taskId, state)

			// Loops through the build tasks to find the one
			// matching the received task status, so it can update
			// it's status.
			for _, task := range b.Tasks {
				if task.Id != taskId {
					continue
				}

				task.Status = taskStatus
				b.Save()
				break
			}

			switch state {
			case mesos.TaskState_TASK_RUNNING:
			case mesos.TaskState_TASK_FINISHED:
				finishedTasks++
			case mesos.TaskState_TASK_KILLED:
			case mesos.TaskState_TASK_LOST:
			case mesos.TaskState_TASK_FAILED:
			}
		}
	}

}

func (b *Build) SaveAndHandleError(err error) {
	log.Print(err)
	b.Save()
}

func (b *Build) RedisKey() string {
	return fmt.Sprintf("pugio:builds:%s", b.Id)
}

func (b *Build) GitRepo() string {
	return fmt.Sprintf("git@git.corp.adobe.com:%s.git", b.ImgName())
}

func (b *Build) ImgName() string {
	return fmt.Sprintf("typekit/%s", b.App)
}

func (b *Build) FullImgName() string {
	return fmt.Sprintf("%s/%s", dockerRegistry, b.ImgName())
}

func (b *Build) CloneCmd() string {
	return fmt.Sprintf("(ssh -o StrictHostKeyChecking=no git@git.corp.adobe.com || true) && rm -rf %s && git clone --depth 1 --branch %s %s && cd %s && bundle install --jobs 4 --deployment", b.App, b.Branch, b.GitRepo(), b.App)
}
