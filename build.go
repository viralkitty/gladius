package main

import (
	"fmt"
	"log"
)

type Build struct {
	App    string
	Branch string
}

func (b Build) Create() {
	statusCode := b.clone()
	log.Printf("Finished %v", statusCode)
	//b.bundle()
}

func (b Build) clone() int {
	opts := ContainerOpts(b.App, "clone")

	opts.Config.Entrypoint = []string{"sh"}
	opts.Config.Cmd = []string{"-c", fmt.Sprintf("git clone --depth 1 --branch %s git@git.corp.adobe.com:typekit/%s.git . && bundle install --deployment", b.Branch, b.App)}

	return WaitForContainer(NewContainer(opts))
}
