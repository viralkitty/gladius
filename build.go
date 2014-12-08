package main

type Build struct {
	App string
	Sha string
}

func (b Build) Create() {
	NewContainer(ContainerOpts(b.App, "clone"))
}
