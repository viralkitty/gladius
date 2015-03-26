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

	docker "github.com/fsouza/go-dockerclient"
	redis "github.com/garyburd/redigo/redis"
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
	Image            *docker.Image          `json:"-"`
	Scheduler        *Scheduler             `json:"-"`
	TaskStatusesChan chan *mesos.TaskStatus `json:"-"`
	Tasks            []*Task                `json:"tasks,omitempty"`
}

func NewBuild() *Build {
	return &Build{
		Id: strconv.Itoa(rand.Int()),
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
	conn := redisPool.Get()
	prefix := "pugio:builds:"
	keysReply, err := conn.Do("KEYS", fmt.Sprintf("%s*", prefix))
	keys, err := redis.Strings(keysReply, err)
	builds := []Build{}

	defer conn.Close()

	if err != nil {
		log.Printf("Could not get keys: %s", err)
		return nil
	}

	for _, key := range keys {
		var b *Build

		buildReply, err := conn.Do("GET", key)
		buildJsonBytes, err := redis.Bytes(buildReply, err)

		if err != nil {
			fmt.Sprintf("Could not get key: %s", key)
			continue
		}

		err = json.Unmarshal(buildJsonBytes, &b)

		if err != nil {
			fmt.Sprintf("Could not unmarshal object: %v", err)
			return nil
		}

		builds = append(builds, *b)
	}

	return builds
}

func (b *Build) Build() {
	var err error

	//	b.Save()
	//
	//	err = b.pullImage()
	//
	//	if err != nil {
	//		b.SaveAndHandleError(err)
	//		return
	//	}
	//
	//	err = b.createContainer()
	//
	//	if err != nil {
	//		b.SaveAndHandleError(err)
	//		return
	//	}
	//
	//	for {
	//		err = b.startContainer()
	//
	//		if err != nil {
	//			b.SaveAndHandleError(err)
	//			continue
	//		}
	//
	//		err = b.waitContainer()
	//
	//		if err != nil {
	//			b.SaveAndHandleError(err)
	//			continue
	//		}
	//
	//		break
	//	}
	//
	//	err = b.commitContainer()
	//
	//	if err != nil {
	//		b.SaveAndHandleError(err)
	//		return
	//	}
	//
	//	err = b.removeContainer()
	//
	//	if err != nil {
	//		b.SaveAndHandleError(err)
	//		return
	//	}
	//
	//	err = b.pushImage()
	//
	//	if err != nil {
	//		b.SaveAndHandleError(err)
	//		return
	//	}

	err = b.launchTasks()

	if err != nil {
		b.SaveAndHandleError(err)
		return
	}

	//err = b.removeImage()

	//if err != nil {
	//	b.SaveAndHandleError(err)
	//	return
	//}

}

func (b *Build) createContainer() error {
	b.log("Creating container")

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
		fmt.Sprintf("Could not create container from image %s: %s", baseImage, err.Error())
		return err
	}

	return nil
}

func (b *Build) startContainer() error {
	b.log("Starting container")

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
		b.log("Could not start container: %s", err)

		return err
	}

	return nil
}

func (b *Build) waitContainer() error {
	b.log("Waiting for container")

	status, err := dockerCli.WaitContainer(b.Container.ID)

	if err != nil {
		b.log("Could not wait for container: %s", err)

		return err
	}

	if status != 0 {
		msg := fmt.Sprintf("Clone or bundle failed: %s", err)

		b.log(msg)

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
	maxRetries := 20
	pullRetryInterval := 1 * time.Minute

	go func() {
		for retries := 0; retries < maxRetries; retries++ {
			err := dockerCli.PullImage(opts, auth)

			if err != nil {
				log.Printf("Failed to pull image %s: %s", baseImage, err.Error())
				log.Printf("Attempting to repull image %s: %s", baseImage, err.Error())
				time.Sleep(pullRetryInterval)

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
	var err error

	b.log("Commiting container")

	commitOpts := docker.CommitContainerOptions{
		Container:  b.Container.ID,
		Repository: b.FullImgName(),
		Tag:        b.Id,
	}

	b.Image, err = dockerCli.CommitContainer(commitOpts)

	if err != nil {
		b.log("Could not commit container: %s", err)

		return err
	}

	return nil
}

func (b *Build) removeContainer() error {
	b.log("Removing container")

	removeOpts := docker.RemoveContainerOptions{
		ID:    b.Container.ID,
		Force: true,
	}

	err := dockerCli.RemoveContainer(removeOpts)

	if err != nil {
		b.log("Could not remove the container: %s", err)

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
			fmt.Sprintf("throwing task into chan: %+v", t)
			t.Build = b
			t.BuildId = b.Id
			tasks <- t
		}(task)
	}

	return nil
}

func (b *Build) Save() error {
	key := b.RedisKey()
	conn := redisPool.Get()
	buildJson, err := json.Marshal(b)

	defer conn.Close()

	b.log("Saving")

	if err != nil {
		b.log("Failed to marshal as JSON: %s", err)

		return err
	}

	_, err = conn.Do("SET", key, buildJson)

	if err != nil {
		b.log("Failed to set key %s: %v", key, err.Error())

		return err
	}

	return nil
}

func (b *Build) taskStatusLoop() {
	b.log("Entering task status loop")

	for finishedTasks := 0; finishedTasks < len(b.Tasks); {
		b.log("Began iteration %d in task status loop", finishedTasks)

		select {
		case taskStatus := <-b.TaskStatusesChan:
			state := taskStatus.GetState()
			taskId := taskStatus.TaskId.GetValue()

			fmt.Sprintf("Task %s is in the %s state", taskId, state)

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
	b.log(err.Error())
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

func (b *Build) removeImage() error {
	fmt.Sprintf("Attempting to remove image: %s", b.Image.ID)

	err := dockerCli.RemoveImage(b.Image.ID)

	if err != nil {
		fmt.Sprintf("Couldn't remove image '%s': %s", b.Image.ID, err)
		return err
	}

	fmt.Sprintf("Removed image: %s", b.Image.ID)

	return nil
}

func (b *Build) log(msg string, args ...interface{}) {
	var buffer bytes.Buffer

	prefix := fmt.Sprintf("[%s] ", b.Id)

	buffer.WriteString(prefix)
	buffer.WriteString(fmt.Sprintf(msg, args...))

	log.Printf(buffer.String())
}
