package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"github.com/garyburd/redigo/redis"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"log"
	"math/rand"
	"strconv"
	"strings"
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
		State:     "running",
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

	prefix := "pugio:builds:"
	keys, err := redis.Strings(redisCli.Do("KEYS", fmt.Sprintf("%s*", prefix)))

	if err != nil {
		log.Printf("Could not get keys: %s", err)
		return nil
	}

	for _, key := range keys {
		log.Printf("found key: %s", key)

		var b *Build
		var buildJsonBytes []byte

		buildJsonBytes, err = redis.Bytes(redisCli.Do("GET", key))

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
	b.Save()
	b.setRedisExpiry()

	if b.pullImage() != nil {
		b.State = "failed"
		b.Save()
		return
	}
	if b.createContainer() != nil {
		b.State = "failed"
		b.Save()
		return
	}
	if b.startContainer() != nil {
		b.State = "failed"
		b.Save()
		return
	}
	if b.waitContainer() != nil {
		b.State = "failed"
		b.Save()
		return
	}
	if b.commitContainer() != nil {
		b.State = "failed"
		b.Save()
		return
	}
	if b.removeContainer() != nil {
		b.State = "failed"
		b.Save()
		return
	}
	if b.pushImage() != nil {
		b.State = "failed"
		b.Save()
		return
	}
	if b.launchTasks() != nil {
		b.State = "failed"
		b.Save()
		return
	}
}

func (b *Build) log(msg string) {
	log.Print(msg)
	redisCli.Do("APPEND", b.RedisLogKey(), fmt.Sprintf("%s\n\n", msg))
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
		b.log("Could not create container")
		b.log(err.Error())
		return err
	}

	log.Printf("%+v", b.Container)

	return nil
}

func (b *Build) startContainer() error {
	b.log("Starting container")

	err := dockerCli.StartContainer(b.Container.ID, &docker.HostConfig{
		Binds: []string{"/root/.ssh:/root/.ssh"},
	})

	if err != nil {
		b.log("Could not start container")
		b.log(err.Error())
		return err
	}

	log.Printf("%+v", b.Container)

	return nil
}

func (b *Build) waitContainer() error {
	b.log("Waiting for container")

	status, err := dockerCli.WaitContainer(b.Container.ID)

	if err != nil {
		b.log("Could not wait for container")
		b.log(err.Error())
		return err
	}

	if status != 0 {
		msg := "Clone or bundle failed"

		b.log(msg)
		return errors.New(msg)
	}

	return nil
}

func (b *Build) pullImage() error {
	b.log("Pulling typekit bundler")

	opts := docker.PullImageOptions{
		Repository: baseImage,
		Registry:   dockerRegistry,
	}

	err := dockerCli.PullImage(opts, docker.AuthConfiguration{})

	if err != nil {
		b.log("Could not pull bundler typekit image")
		b.log(err.Error())
		return err
	}

	return nil
}

func (b *Build) commitContainer() error {
	b.log("Commiting container")

	commitOpts := docker.CommitContainerOptions{
		Container:  b.Container.ID,
		Repository: b.FullImgName(),
		Tag:        b.Id,
	}

	_, err := dockerCli.CommitContainer(commitOpts)

	if err != nil {
		b.log("Could not commit container")
		b.log(err.Error())
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
		b.log("Could not remove the container")
		b.log(err.Error())
		return err
	}

	return nil
}

func (b *Build) pushImage() error {
	b.log("Pushing image")

	var buf bytes.Buffer

	pushOpts := docker.PushImageOptions{
		Name:         b.FullImgName(),
		OutputStream: &buf,
	}

	authConfig := docker.AuthConfiguration{}

	err := dockerCli.PushImage(pushOpts, authConfig)
	msg := "Could not push the image"

	if err != nil {
		b.log(msg)
		b.log(err.Error())
		return err
	}

	if strings.Contains(buf.String(), "Image successfully pushed") != true {
		b.log(msg)
		return errors.New(msg)
	}

	return nil
}

func (b *Build) launchTasks() error {
	for _, task := range b.Tasks {
		log.Printf("throwing task into chan: %+v", task)
		task.Build = b
		tasks <- task
	}

	for i := 0; i < len(b.Tasks); i++ {
		status := <-b.TaskStatusesChan

		for _, tk := range b.Tasks {
			if tk.Id != status.TaskId.GetValue() {
				continue
			}

			tk.Status = status
		}

		b.Save()
	}

	for _, task := range b.Tasks {
		if task.Status.GetState() != mesos.TaskState_TASK_FINISHED {
			log.Print("Failure")
			return errors.New("Failed running")
		}
	}

	log.Print("Success!!!!!!!!!!!!!!")

	return nil
}

func (b *Build) Save() error {
	b.log("Persisting in Redis")

	buildJson, err := json.Marshal(b)

	if err != nil {
		log.Printf("json error: %+v", err)
		return err
	}

	_, err = redisCli.Do("SET", b.RedisKey(), buildJson)

	if err != nil {
		log.Printf("problem setting key: %v", err)
		return err
	}

	return nil
}

func (b *Build) setRedisExpiry() error {
	b.log("Setting redis key expirations.")

	redisCli.Do("EXPIRE", b.RedisKey(), 3600)
	redisCli.Do("EXPIRE", b.RedisLogKey(), 3600)

	return nil
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
	return fmt.Sprintf("(ssh -o StrictHostKeyChecking=no git@git.corp.adobe.com || true) && git clone --depth 1 --branch %s %s && cd %s && bundle install --jobs 4 --deployment", b.Branch, b.GitRepo(), b.App)
}

func (b *Build) RedisLogKey() string {
	return fmt.Sprintf("pugio:logs:%s", b.Id)
}
