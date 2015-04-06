package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	redis "github.com/garyburd/redigo/redis"
	sched "github.com/mesos/mesos-go/scheduler"
)

var (
	quit             chan bool
	tasks            chan *Task
	routes           *Routes
	dockerCli        *docker.Client
	executorId       string
	executorCommand  string
	cpusPerTask      float64
	memoryPerTask    float64
	frameworkName    string
	gladiusPort      string
	mesosMaster      string
	redisPool        *redis.Pool
	redisIdleTimeout time.Duration
	redisMaxIdle     int
	schedulerDriver  *sched.MesosSchedulerDriver
)

func init() {
	var (
		dockerCliErr       error
		schedulerDriverErr error
		cpusParseErr       error
		memoryParseErr     error
	)

	if os.Getenv("CPUS_PER_TASK") == "" {
		log.Fatal("CPUS_PER_TASK must be set")
	}

	if os.Getenv("DOCKER_SOCKET") == "" {
		log.Fatal("DOCKER_SOCKET must be set")
	}

	if os.Getenv("EXECUTOR_COMMAND") == "" {
		log.Fatal("EXECUTOR_COMMAND must be set")
	}

	if os.Getenv("EXECUTOR_ID") == "" {
		log.Fatal("EXECUTOR_ID must be set")
	}

	if os.Getenv("FRAMEWORK_NAME") == "" {
		log.Fatal("FRAMEWORK_NAME must be set")
	}

	if os.Getenv("GLADIUS_PORT") == "" {
		log.Fatal("GLADIUS_PORT must be set")
	}

	if os.Getenv("MEMORY_PER_TASK") == "" {
		log.Fatal("MEMORY_PER_TASK must be set")
	}

	if os.Getenv("REDIS_IDLE_TIMEOUT") == "" {
		log.Fatal("REDIS_IDLE_TIMEOUT must be set")
	}

	if os.Getenv("REDIS_MAX_IDLE") == "" {
		log.Fatal("REDIS_MAX_IDLE must be set")
	}

	if os.Getenv("REDIS_PORT") == "" {
		log.Fatal("REDIS_PORT must be set")
	}

	if os.Getenv("REDIS_PROTOCOL") == "" {
		log.Fatal("REDIS_PROTOCOL must be set")
	}

	quit = make(chan bool)
	tasks = make(chan *Task)
	cpusPerTask, cpusParseErr = strconv.ParseFloat(os.Getenv("CPUS_PER_TASK"), 64)
	dockerCli, dockerCliErr = docker.NewTLSClient(os.Getenv("DOCKER_API"), os.Getenv("DOCKER_CERT"), os.Getenv("DOCKER_KEY"), os.Getenv("DOCKER_CA"))
	executorCommand = os.Getenv("EXECUTOR_COMMAND")
	executorId = os.Getenv("EXECUTOR_ID")
	frameworkName = os.Getenv("FRAMEWORK_NAME")
	mesosMaster = os.Getenv("MESOS_MASTER")
	gladiusPort = os.Getenv("GLADIUS_PORT")
	memoryPerTask, memoryParseErr = strconv.ParseFloat(os.Getenv("MEMORY_PER_TASK"), 64)
	redisPool = NewRedisPool()
	routes = NewRoutes()
	schedulerDriver, schedulerDriverErr = NewSchedulerDriver()

	if dockerCliErr != nil {
		log.Fatal("Failed to initialize Docker: ", dockerCliErr)
	}

	if cpusParseErr != nil {
		log.Fatal("Failed to parse CPUS_PER_TASK: %v", cpusParseErr)
	}

	if memoryParseErr != nil {
		log.Fatal("Failed to parse MEMORY_PER_TASK: %v", memoryParseErr)
	}

	if schedulerDriverErr != nil {
		log.Fatal("Failed to initialized scheduler driver: %s", schedulerDriverErr)
	}

	rand.Seed(time.Now().UTC().UnixNano())
	http.HandleFunc("/builds", routes.Builds)
	http.HandleFunc("/builds/", routes.Builds)

	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%s", gladiusPort), nil)

		if err != nil {
			log.Printf("Failed to serve the API: %s", err.Error())

			quit <- true
		}
	}()

	go func() {
		stat, err := schedulerDriver.Run()

		if err != nil {
			log.Printf("Framework stopped with status %s and error: %s\n", stat.String(), err.Error())

			quit <- true
		}
	}()
}

func main() {
	<-quit
}
