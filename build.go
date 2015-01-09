package main

import (
	"bytes"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"log"
	"os"
	"strings"
)

var client, _ = docker.NewClient(os.Getenv("DOCKER_SOCK_PATH"))

type Build struct {
	App    string
	Branch string
}

func (b *Build) Create(scheduler *Scheduler) {
	var status int
	var buf bytes.Buffer

	log.Printf("Starting new build for %s:%s", b.App, b.Branch)

	imgName := fmt.Sprintf("typekit/%s", b.App)
	fullImgName := fmt.Sprintf("docker.corp.adobe.com/%s", imgName)
	gitRepo := fmt.Sprintf("git@git.corp.adobe.com:%s.git", imgName)
	createOpts := docker.CreateContainerOptions{
		Config: &docker.Config{
			Tty:          true,
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			WorkingDir:   "/",
			Image:        "razic/bundler",
			Entrypoint:   []string{"sh"},
			Cmd:          []string{"-c", fmt.Sprintf("git clone --depth 1 --branch %s %s && cd %s", b.Branch, gitRepo, b.App)},
		},
		HostConfig: &docker.HostConfig{
			VolumesFrom: []string{"ssh"},
		},
	}
	tagOpts := docker.TagImageOptions{
		Force: true,
		Tag:   b.Branch,
		Repo:  fullImgName,
	}
	pushOpts := docker.PushImageOptions{
		Name:         fullImgName,
		OutputStream: &buf,
	}
	authConfig := docker.AuthConfiguration{}
	c, err := client.CreateContainer(createOpts)
	commitOpts := docker.CommitContainerOptions{
		Container:  c.ID,
		Repository: imgName,
	}
	removeOpts := docker.RemoveContainerOptions{
		ID:    c.ID,
		Force: true,
	}

	if err != nil {
		log.Printf("Could not create container: %v", err)
		return
	}

	log.Printf("Starting container: %v", c)

	err = client.StartContainer(c.ID, createOpts.HostConfig)

	if err != nil {
		log.Printf("Could not start container: %v", err)
		return
	}

	log.Printf("Waiting for container: %v", c)

	status, err = client.WaitContainer(c.ID)

	if err != nil {
		log.Printf("Could not wait for the container: %v", err)
		return
	}

	if status != 0 {
		log.Printf("Clone or bundle failed in container %d", c.ID)
		return
	}

	log.Printf("Commiting container into image")

	_, err = client.CommitContainer(commitOpts)

	if err != nil {
		log.Printf("Could not commit container: %v", err)
		return
	}

	log.Printf("Tagging image with: %s", b.Branch)

	err = client.TagImage(imgName, tagOpts)

	if err != nil {
		log.Printf("Could not tag the image:", err)
		return
	}

	log.Printf("Removing container")

	err = client.RemoveContainer(removeOpts)

	if err != nil {
		log.Printf("Could not remove the container: %v", err)
		return
	}

	log.Printf("Pushing image")

	err = client.PushImage(pushOpts, authConfig)

	if err != nil {
		log.Printf("Could not push the image: %v", err)
		return
	}

	if strings.Contains(buf.String(), "Image successfully pushed") != true {
		log.Printf("Push was unsuccessful: %v", err)
		return
	}

	taskStatusChan := make(chan mesos.TaskStatus)

	log.Printf("Launching Tasks")

	go func() {
		scheduler.chanchan <- Task{
			Cmd:    "rspec spec/models/kit_spec.rb",
			Output: taskStatusChan,
		}
	}()

	go func() {
		scheduler.chanchan <- Task{
			Cmd:    "rspec spec/controllers/errors_controller_spec.rb",
			Output: taskStatusChan,
		}
	}()

	log.Printf("Waiting for results", status)

	for i := 0; i < 2; i++ {
		status := <-taskStatusChan

		log.Printf("Got status %+v", status)

		if status.GetState() != mesos.TaskState_TASK_FINISHED {
			log.Print("Failure")
			return
		}
	}

	log.Print("Success!!!!!!!!!!!!!!")
}
