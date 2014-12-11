package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

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

	log.Printf("Container exited with status code %+v", statusCode)

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
		Repository: "typekit/typekitapp",
	}

	return opts
}

func CommitContainer(opts docker.CommitContainerOptions) *docker.Image {
	img, err := client.CommitContainer(opts)

	if err != nil {
		log.Fatal("Could not commit container:", err)
	}

	log.Printf("Committed container %+v", img)

	return img
}

func TagImage(tag string) {
	opts := docker.TagImageOptions{
		Repo:  "docker.corp.adobe.com/typekit/typekitapp",
		Force: true,
	}

	err := client.TagImage("typekit/typekitapp", opts)

	if err != nil {
		log.Fatal("Could not tag the image:", err)
	}

	log.Printf("Tagged image %+v", opts)
}

func PushImage(tag string) {
	var buf bytes.Buffer

	opts := docker.PushImageOptions{
		Name:         "docker.corp.adobe.com/typekit/typekitapp",
		OutputStream: &buf,
	}

	err := client.PushImage(opts, docker.AuthConfiguration{})

	if err != nil {
		log.Fatal("Could not push the image: ", err)
	}

	if strings.Contains(buf.String(), "Image successfully pushed") == true {
		log.Printf("Image successfully pushed...")
		return
	}

	log.Fatal("Error occurred while pushing the image")
}
