package main

import (
	"fmt"
)

type Build struct {
	App    string
	Branch string
}

func (b Build) Create() {
	b.clone()
	//b.bundle()
}

func (b Build) clone() {
	opts := ContainerOpts(b.App, "clone")

	opts.Config.Entrypoint = []string{"sh"}
	opts.Config.Cmd = []string{"-c", fmt.Sprintf("git clone --branch %s git@git.corp.adobe.com:typekit/%s.git . && bundle && bash", b.Branch, b.App)}

	NewContainer(opts)
}
