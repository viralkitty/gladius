package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/fsouza/go-dockerclient"
)

var client, _ = docker.NewClient(os.Getenv("DOCKER_SOCK_PATH"))

func NewContainer(opts docker.CreateContainerOptions) *docker.Container {
	c, err := client.CreateContainer(opts)

	if err != nil {
		log.Fatal("Could not create container: ", err)
	}

	err = client.StartContainer(c.ID, opts.HostConfig)

	if err != nil {
		log.Fatal("Could start container: ", err)
	}

	return c
}

func WaitForContainer(ctr *docker.Container) int {
	statusCode, err := client.WaitContainer(ctr.ID)

	if err != nil {
		log.Fatal("Could not wait for the container: ", err)
	}

	return statusCode
}

func ContainerOpts(app string, name string) docker.CreateContainerOptions {
	config, err := os.Open(fmt.Sprintf("%s/src/git.corp.adobe.com/typekit/gladius/containers/%s-%s.json", os.Getenv("GOPATH"), app, name))

	if err != nil {
		log.Fatal("Could not open file: ", err)
	}

	opts := docker.CreateContainerOptions{}
	decoder := json.NewDecoder(config)

	err = decoder.Decode(&opts)

	if err != nil {
		log.Fatal("Could not read config file:", err)
	}

	return opts
}

func CommitContainerOpts(container string, tag string) docker.CommitContainerOptions {
	opts := docker.CommitContainerOptions{
		Container:  container,
		Repository: "docker.corp.adobe.com/typekit/typekit",
		Tag:        tag,
		Message:    "typekit image",
	}

	return opts
}

func CommitContainer(opts docker.CommitContainerOptions) *docker.Image {
	img, err := client.CommitContainer(opts)

	if err != nil {
		log.Fatal("Could not commit container:", err)
	}

	return img
}
