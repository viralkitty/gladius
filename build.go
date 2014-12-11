package main

import (
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"log"
)

type Build struct {
	App       string
	Branch    string
	Container *docker.Container
}

func (b *Build) Create() {
	statusCode := b.cloneRepo()

	if statusCode != 0 {
		log.Fatal("Error occurred while cloning the repo")
		return
	}

	b.commitContainer()
	b.tagImage()
	b.pushImage()
}

func (b *Build) cloneRepo() int {
	opts := ContainerOpts(b.App, "clone")

	opts.Config.Entrypoint = []string{"sh"}

	log.Printf("Starting to clone the repo")

	//opts.Config.Cmd = []string{"-c", fmt.Sprintf("git clone --depth 1 --branch %s git@git.corp.adobe.com:typekit/%s.git . && bundle install --jobs 4 --deployment", b.Branch, b.App)}
	opts.Config.Cmd = []string{"-c", fmt.Sprintf("git clone --depth 1 --branch %s git@git.corp.adobe.com:typekit/%s.git .", b.Branch, b.App)}
	b.Container = NewContainer(opts)

	log.Printf("Waiting...")

	return WaitForContainer(b.Container)
}

func (b *Build) commitContainer() {
	CommitContainer(CommitContainerOpts(b.Container.ID, b.Branch))
}

func (b *Build) tagImage() {
	TagImage(b.Branch)
}

func (b *Build) pushImage() {
	PushImage(b.Branch)
}

func (b *Build) runTests() {
}
