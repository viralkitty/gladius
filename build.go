package main

import (
	"fmt"
	"github.com/fsouza/go-dockerclient"
	log "github.com/golang/glog"
)

type Build struct {
	App       string
	Branch    string
	Container *docker.Container
}

func (b *Build) Create(builds chan *Build) {
	statusCode := b.cloneRepo()

	if statusCode != 0 {
		log.Infoln("Clone failed")
		return
	}

	b.commitContainer()
	b.tagImage()
	b.pushImage()

	go func() { builds <- b }()
}

func (b *Build) cloneRepo() int {
	repo := fmt.Sprintf("git@git.corp.adobe.com:typekit/%s.git", b.App)

	log.Infoln("Cloning :", repo)

	opts := ContainerOpts(b.App, "clone")
	opts.Config.Entrypoint = []string{"sh"}
	//opts.Config.Cmd = []string{"-c", fmt.Sprintf("git clone --depth 1 --branch %s %s", b.Branch, repo)}
	opts.Config.Cmd = []string{"-c", fmt.Sprintf("git clone --depth 1 --branch %s %s && cd typekit && bundle install --jobs 4 --deployment", b.Branch, repo)}
	b.Container = NewContainer(opts)

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
