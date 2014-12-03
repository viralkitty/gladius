package appimage

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"

	"github.com/fsouza/go-dockerclient"
)

func newUUID() string {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return ""
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

func NewBuild(buildId string) {
	client, err := docker.NewClient("unix:///var/run/docker.sock")

	if err != nil {
		log.Fatal(err)
	}

	buildImage := &BuildImage{client: client, buildId: buildId}
	//buildImage.PullImageDeps()
	buildImage.CreateDataVolume()
	buildImage.PullSha()
}

type BuildImage struct {
	buildId string
	client  *docker.Client
	uuid    string
}

func (b BuildImage) UUID() string {
	if b.uuid == "" {
		b.uuid = fmt.Sprintf("%s-%s", b.buildId[0:6], newUUID())
	}

	return b.uuid
}

func (b BuildImage) PullImageDeps() {
	log.Printf("Pulling ubuntu image...")

	err := b.client.PullImage(docker.PullImageOptions{
		Repository: "ubuntu",
	}, docker.AuthConfiguration{})

	if err != nil {
		log.Fatal("Can't find an image %v", err)
	}

	log.Printf("Pulling bundler-typekit image...")

	err = b.client.PullImage(docker.PullImageOptions{
		Repository: "docker.corp.adobe.com/typekit/bundler-typekit",
	}, docker.AuthConfiguration{})

	if err != nil {
		log.Fatal("Can't find an image %v", err)
	}
}

func (b BuildImage) CreateDataVolume() {

	container, err := b.client.CreateContainer(docker.CreateContainerOptions{
		Name: fmt.Sprintf("typekit-%s", b.UUID()),
		Config: &docker.Config{
			Image:       "ubuntu",
			Cmd:         []string{"cat"},
			Volumes:     map[string]struct{}{"/typekit": {}},
			WorkingDir:  "/typekit",
			AttachStdin: true,
			Tty:         true,
		},
	})

	if err != nil {
		log.Fatal("Could not create data volume %v", err)
	}

	err = b.client.StartContainer(container.ID, &docker.HostConfig{})

	if err != nil {
		log.Fatal("Could start data volume %v", err)
	}
}

func (b BuildImage) PullSha() {
	container, err := b.client.CreateContainer(docker.CreateContainerOptions{
		Name: fmt.Sprintf("typekit-%s", b.UUID()),
		Config: &docker.Config{
			Image:      "docker.corp.adobe.com/typekit/bundler-typekit-ssh",
			WorkingDir: "/typekit",
			Cmd:        []string{"git", "clone", "git@git.corp.adobe.com:typekit/typekit.git", "--depth", "1", "--branch", b.buildId, "typekit"},
			// Cmd: 		 []string{"bash"},
			Tty: true,
		},
	})

	if err != nil {
		log.Fatal("Could not create a container to pull the SHA%v", err)
	}

	err = b.client.StartContainer(container.ID, &docker.HostConfig{
		VolumesFrom: []string{"typekit"},
	})

	if err != nil {
		log.Fatal("Could not start a git pull container %v", err)
	}
}
