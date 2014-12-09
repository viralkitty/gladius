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
	b.cloneRepo()
	b.commitImage()
}

func (b *Build) cloneRepo() {
	opts := ContainerOpts(b.App, "clone")

	opts.Config.Entrypoint = []string{"sh"}

	opts.Config.Cmd = []string{"-c", fmt.Sprintf("git clone --depth 1 --branch %s git@git.corp.adobe.com:typekit/%s.git . && bundle install --jobs 4 --deployment", b.Branch, b.App)}
	//opts.Config.Cmd = []string{"-c", fmt.Sprintf("git clone --depth 1 --branch %s git@git.corp.adobe.com:typekit/%s.git .", b.Branch, b.App)}
	b.Container = NewContainer(opts)

	WaitForContainer(b.Container)
}

func (b *Build) commitImage() {
	image := CommitContainer(CommitContainerOpts(b.Container.ID, b.Branch))

	log.Printf("Created image %v", image)
}
