package main

import (
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"github.com/garyburd/redigo/redis"
	"github.com/gogo/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

const (
	FRAMEWORK_NAME = "Gladius"
)

var master = os.Getenv("MESOS_MASTER")
var execUri = os.Getenv("EXEC_URI")
var redisCli, redisCliErr = redis.Dial("tcp", os.Getenv("REDIS_TCP_ADDR"))
var dockerCli, dockerCliErr = docker.NewClient(os.Getenv("DOCKER_SOCK_PATH"))
var tasks = make(chan *Task)

func init() {
	log.Printf("Initializing the Example Scheduler...")

	if redisCliErr != nil {
		log.Fatal("Failed to connect with Redis: %v", redisCliErr)
	}

	if dockerCliErr != nil {
		log.Fatal("Failed to connect with Redis: %v", dockerCliErr)
	}

	rand.Seed(time.Now().UTC().UnixNano())
}

func main() {
	// build command executor
	exec := &mesos.ExecutorInfo{
		ExecutorId: util.NewExecutorID("default"),
		Command:    util.NewCommandInfo(execUri),
	}

	// the framework
	fwinfo := &mesos.FrameworkInfo{
		User: proto.String(""), // Mesos-go will fill in user.
		Name: proto.String(FRAMEWORK_NAME),
	}

	scheduler := NewScheduler(exec)

	driver, err := sched.NewMesosSchedulerDriver(
		scheduler,
		fwinfo,
		master,
		nil,
	)

	if err != nil {
		log.Printf("Unable to create a SchedulerDriver ", err.Error())
	}

	routes := &Routes{scheduler}
	listenAt := fmt.Sprintf(":%s", os.Getenv("GLADIUS_HTTP_PORT"))

	http.HandleFunc("/builds", routes.Builds)
	go http.ListenAndServe(listenAt, nil)
	log.Printf("Listening at %s", listenAt)

	if stat, err := driver.Run(); err != nil {
		log.Printf("Framework stopped with status %s and error: %s\n", stat.String(), err.Error())
	}
}
