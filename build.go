package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	mesos "github.com/mesos/mesos-go/mesosproto"
)

type Build struct {
	Id               string                 `json:"id,omitempty"`
	App              string                 `json:"app,omitempty"`
	Branch           string                 `json:"branch,omitempty"`
	Container        *docker.Container      `json:"-"`
	Image            *docker.Image          `json:"-"`
	TaskStatusesChan chan *mesos.TaskStatus `json:"-"`
	Tasks            []*Task                `json:"tasks,omitempty"`
	BaseImage        string                 `json:"baseImage,omitempty"`
	Log              string                 `json:"log,omitempty"`
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
			NewTask("rspec spec/models --no-color"),
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
		BaseImage:        "docker.corp.adobe.com/typekit/bundler-typekit",
	}
}

func (b *Build) Build() {
	retryInterval := 10 * time.Second
	buildIsTakingTooLong := time.After(30 * time.Minute)

pullImageLoop:
	for {
		pullIsTakingTooLong := time.After(10 * time.Second)
		pulledSuccessfully, errorWhilePulling := b.pullBaseImage()

		select {
		case <-pulledSuccessfully:
			break pullImageLoop
		case <-errorWhilePulling:
			time.Sleep(retryInterval)

			continue pullImageLoop
		case <-pullIsTakingTooLong:
			b.log("Timed out pulling base image")

			return
		case <-buildIsTakingTooLong:
			b.log("Build timed out")

			return
		}
	}

createContainerLoop:
	for {
		createIsTakingTooLong := time.After(10 * time.Second)
		createdSuccessfully, errorWhileCreating := b.createContainer()

		select {
		case <-createdSuccessfully:
			break createContainerLoop
		case err := <-errorWhileCreating:
			b.log(err.Error())
			time.Sleep(retryInterval)

			continue createContainerLoop
		case <-createIsTakingTooLong:
			b.log("Timed out creating container")

			return
		case <-buildIsTakingTooLong:
			b.log("Build timed out")

			return
		}
	}

startContainerLoop:
	for {
		startIsTakingTooLong := time.After(10 * time.Second)
		startedSuccessfully, errorWhileStarting := b.startContainer()

		select {
		case <-startedSuccessfully:
			break startContainerLoop
		case err := <-errorWhileStarting:
			b.log(err.Error())
			time.Sleep(retryInterval)

			continue startContainerLoop
		case <-startIsTakingTooLong:
			b.log("Timed out starting container")

			return
		case <-buildIsTakingTooLong:
			b.log("Build timed out")

			return
		}
	}

waitContainerLoop:
	for {
		waitIsTakingTooLong := time.After(10 * time.Minute)
		waitedSuccessfully, errorWhileWaiting := b.waitContainer()

		select {
		case <-waitedSuccessfully:
			break waitContainerLoop
		case <-errorWhileWaiting:
			time.Sleep(retryInterval)

			continue waitContainerLoop
		case <-waitIsTakingTooLong:
			b.log("Timed out waiting container")

			return
		case <-buildIsTakingTooLong:
			b.log("Build timed out")

			return
		}
	}

commitContainerLoop:
	for {
		commitIsTakingTooLong := time.After(1 * time.Minute)
		commitedSuccessfully, errorWhileCommitting := b.commitContainer()

		select {
		case <-commitedSuccessfully:
			break commitContainerLoop
		case <-errorWhileCommitting:
			time.Sleep(retryInterval)

			continue commitContainerLoop
		case <-commitIsTakingTooLong:
			b.log("Timed out commiting container")

			return
		case <-buildIsTakingTooLong:
			b.log("Build timed out")

			return
		}
	}

removeContainerLoop:
	for {
		removeIsTakingTooLong := time.After(1 * time.Minute)
		removedSuccessfully, errorWhileRemoving := b.removeContainer()

		select {
		case <-removedSuccessfully:
			break removeContainerLoop
		case <-errorWhileRemoving:
			time.Sleep(retryInterval)

			continue removeContainerLoop
		case <-removeIsTakingTooLong:
			b.log("Timed out removing container")

			break removeContainerLoop
		case <-buildIsTakingTooLong:
			b.log("Build timed out")

			return
		}
	}

pushImageLoop:
	for {
		pushIsTakingTooLong := time.After(10 * time.Minute)
		pushedSuccessfully, errorWhilePushing := b.pushImage()

		select {
		case <-pushedSuccessfully:
			break pushImageLoop
		case <-errorWhilePushing:
			time.Sleep(retryInterval)

			continue pushImageLoop
		case <-pushIsTakingTooLong:
			b.log("Timed out pushing image")

			return
		case <-buildIsTakingTooLong:
			b.log("Build timed out")

			return
		}
	}

	b.launchTasks()

removeImageLoop:
	for {
		removeIsTakingTooLong := time.After(1 * time.Minute)
		removedSuccessfully, errorWhileRemoving := b.removeImage()

		select {
		case <-removedSuccessfully:
			break removeImageLoop
		case <-errorWhileRemoving:
			time.Sleep(retryInterval)

			continue removeImageLoop
		case <-removeIsTakingTooLong:
			b.log("Timed out removing image")

			return
		case <-buildIsTakingTooLong:
			b.log("Build timed out")

			return
		}
	}
}

func (b *Build) createContainer() (<-chan bool, <-chan error) {
	doneChan := make(chan bool)
	errorChan := make(chan error)
	opts := docker.CreateContainerOptions{
		Config: &docker.Config{
			Tty:          true,
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			WorkingDir:   "/",
			Image:        b.BaseImage,
			Entrypoint:   []string{"sh"},
			Cmd:          []string{"-c", b.CloneCmd()},
		},
	}

	b.log("Creating container %v", opts)

	go func() {
		defer close(doneChan)
		defer close(errorChan)

		var err error

		b.Container, err = dockerCli.CreateContainer(opts)

		if err != nil {
			b.log("Could not create container %v: %v", opts, err)

			errorChan <- err

			return
		}

		b.log("Created container %v", b.Container.ID[:7])

		doneChan <- true
	}()

	return doneChan, errorChan
}

func (b *Build) startContainer() (<-chan bool, <-chan error) {
	doneChan := make(chan bool)
	errorChan := make(chan error)

	b.log("Starting container %s", b.Container.ID[:7])

	go func() {
		sshKey := "/root/.ssh/id_rsa"

		if os.Getenv("SSH_KEY") != "" {
			sshKey = os.Getenv("SSH_KEY")
		}

		hostConfig := &docker.HostConfig{
			Binds: []string{
				fmt.Sprintf("%s:/root/.ssh/id_rsa", sshKey),
			},
		}
		err := dockerCli.StartContainer(b.Container.ID, hostConfig)

		if err != nil {
			b.log("Error starting container %s: %v", b.Container.ID[:7], err)

			errorChan <- err

			return
		}

		b.log("Started container %s", b.Container.ID[:7])

		doneChan <- true
	}()

	return doneChan, errorChan
}

func (b *Build) waitContainer() (<-chan bool, <-chan error) {
	doneChan := make(chan bool)
	errorChan := make(chan error)

	b.log("Waiting for container %s", b.Container.ID[:7])

	go func() {
		defer close(doneChan)
		defer close(errorChan)

		status, err := dockerCli.WaitContainer(b.Container.ID)

		if err != nil {
			b.log("Error waiting for container: %v", err)

			errorChan <- err

			return
		}

		if status != 0 {
			msg := fmt.Sprintf("Container %s exited with status 1: %v", b.Container.ID[:7], err)
			err := errors.New(msg)

			b.log(msg)

			errorChan <- err

			return
		}

		b.log("Waited for container %s", b.Container.ID[:7])

		doneChan <- true
	}()

	return doneChan, errorChan
}

func (b *Build) pullImage(opts docker.PullImageOptions) (<-chan bool, <-chan error) {
	doneChan := make(chan bool)
	errorChan := make(chan error)

	b.log("Pulling image %v", opts)

	go func() {
		defer close(doneChan)
		defer close(errorChan)

		err := dockerCli.PullImage(opts, docker.AuthConfiguration{})

		if err != nil {
			b.log("Error pulling image %v: %v", opts, err)

			errorChan <- err

			return
		}

		b.log("Pulled image %s", opts)

		doneChan <- true
	}()

	return doneChan, errorChan
}

func (b *Build) pullBaseImage() (<-chan bool, <-chan error) {
	return b.pullImage(docker.PullImageOptions{
		Repository: b.BaseImage,
	})
}

func (b *Build) commitContainer() (<-chan bool, <-chan error) {
	doneChan := make(chan bool)
	errorChan := make(chan error)
	opts := docker.CommitContainerOptions{
		Container:  b.Container.ID,
		Repository: b.ImageName(),
		Tag:        b.Id,
	}

	b.log("Commiting container %v", opts)

	go func() {
		defer close(doneChan)
		defer close(errorChan)

		img, err := dockerCli.CommitContainer(opts)

		if err != nil {
			b.log("Could not commit container %v: %v", opts, err)

			errorChan <- err

			return
		}

		b.Image = img

		b.log("Commited container: %v", opts)

		doneChan <- true
	}()

	return doneChan, errorChan
}

func (b *Build) removeContainer() (<-chan bool, <-chan error) {
	doneChan := make(chan bool)
	errorChan := make(chan error)
	opts := docker.RemoveContainerOptions{
		ID:    b.Container.ID,
		Force: true,
	}

	b.log("Removing container %v", opts)

	go func() {
		defer close(doneChan)
		defer close(errorChan)

		err := dockerCli.RemoveContainer(opts)

		if err != nil {
			b.log("Error removing container %v: %v", opts, err)

			errorChan <- err

			return
		}

		b.log("Removed container %v", opts)

		doneChan <- true
	}()

	return doneChan, errorChan
}

func (b *Build) pushImage() (<-chan bool, <-chan error) {
	doneChan := make(chan bool)
	errorChan := make(chan error)

	b.log("Pushing image %s", b.Image.ID)

	go func() {
		defer close(doneChan)
		defer close(errorChan)

		var buf bytes.Buffer

		successMsg := "Image successfully pushed"
		auth := docker.AuthConfiguration{}
		opts := docker.PushImageOptions{
			Name:         b.ImageName(),
			OutputStream: &buf,
			Tag:          b.Id,
		}
		err := dockerCli.PushImage(opts, auth)

		if err != nil {
			b.log("Error pushing image %s: %v", b.Image.ID, err)

			errorChan <- err

			return
		}

		if strings.Contains(buf.String(), successMsg) != true {
			b.log("Error pushing image %s: %s", b.Image.ID, buf.String())

			errorChan <- err

			return
		}

		b.log("Pushed image %s", b.Image.ID)

		doneChan <- true
	}()

	return doneChan, errorChan
}

func (b *Build) launchTasks() {
	go b.taskStatusLoop()

	for _, task := range b.Tasks {
		go func(t *Task) {
			fmt.Sprintf("throwing task into chan: %+v", t)
			t.Build = b
			t.BuildId = b.Id
			tasks <- t
		}(task)
	}
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

func (b *Build) RedisLogKey() string {
	return fmt.Sprintf("pugio:builds:%s:log", b.Id)
}

func (b *Build) GitRepo() string {
	return fmt.Sprintf("git@git.corp.adobe.com:typekit/%s.git", b.App)
}

func (b *Build) ImageName() string {
	return fmt.Sprintf("docker.corp.adobe.com/typekit/%s", b.App)
}

func (b *Build) CloneCmd() string {
	return fmt.Sprintf("(ssh -o StrictHostKeyChecking=no git@git.corp.adobe.com || true) && rm -rf %s && git clone --depth 1 --branch %s %s && cd %s && bundle install --jobs 4 --deployment", b.App, b.Branch, b.GitRepo(), b.App)
}

func (b *Build) removeImage() (<-chan bool, <-chan error) {
	doneChan := make(chan bool)
	errorChan := make(chan error)

	b.log("Removing image %s", b.Image.ID[:7])

	go func() {
		defer close(doneChan)
		defer close(errorChan)

		err := dockerCli.RemoveImage(b.Image.ID)

		if err != nil {
			b.log("Couldn't remove image '%s': %s", b.Image.ID[:7], err)

			errorChan <- err

			return
		}

		b.log("Removed image %s", b.Image.ID[:7])

		doneChan <- true
	}()

	return doneChan, errorChan
}

func (b *Build) log(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)

	conn := redisPool.Get()

	defer conn.Close()

	_, err := conn.Do("RPUSH", b.RedisLogKey(), fmt.Sprintf("[%s] %s", time.Now().String(), msg))

	if err != nil {
		log.Printf(err.Error())
	}

	log.Printf("[Build %s] %s", b.Id, msg)
}

func (b *Build) containerLogs() (<-chan bool, <-chan error) {
	doneChan := make(chan bool)
	errorChan := make(chan error)

	go func() {
		reader, writer := io.Pipe()
		opts := docker.AttachToContainerOptions{
			Container:    b.Container.ID,
			OutputStream: writer,
			ErrorStream:  writer,
			Stdout:       true,
			Stderr:       true,
			Stream:       true,
			Logs:         true,
		}

		go func() {
			err := dockerCli.AttachToContainer(opts)

			if err != nil {
				errorChan <- err
			}
		}()

		go func(r io.Reader) {
			scanner := bufio.NewScanner(r)

			for scanner.Scan() {
				log.Printf("%s\n", scanner.Text())
			}

			if err := scanner.Err(); err != nil {
				log.Printf("There was an error with the scanner in attached container: %v", err)
			}
		}(reader)
	}()

	return doneChan, errorChan
}
