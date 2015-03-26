package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
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
	gladiusIP        net.IP
	gladiusPort      string
	mesosMasterIP    net.IP
	mesosMasterPort  string
	redisIP          net.IP
	redisPort        string
	redisProtocol    string
	redisPool        *redis.Pool
	redisIdleTimeout time.Duration
	redisMaxIdle     int
	schedulerIP      net.IP
	schedulerPort    int
	schedulerDriver  *sched.MesosSchedulerDriver
)

func init() {
	var (
		dockerCliErr       error
		schedulerParseErr  error
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

	if os.Getenv("GLADIUS_IP") == "" {
		log.Fatal("GLADIUS_IP must be set")
	}

	if os.Getenv("GLADIUS_PORT") == "" {
		log.Fatal("GLADIUS_PORT must be set")
	}

	if os.Getenv("MEMORY_PER_TASK") == "" {
		log.Fatal("MEMORY_PER_TASK must be set")
	}

	if os.Getenv("SCHEDULER_IP") == "" {
		log.Fatal("SCHEDULER_IP must be set")
	}

	if os.Getenv("MESOS_MASTER_IP") == "" {
		log.Fatal("MESOS_MASTER_IP must be set")
	}

	if os.Getenv("MESOS_MASTER_PORT") == "" {
		log.Fatal("MESOS_MASTER_PORT must be set")
	}

	if os.Getenv("REDIS_IDLE_TIMEOUT") == "" {
		log.Fatal("REDIS_IDLE_TIMEOUT must be set")
	}

	if os.Getenv("REDIS_IP") == "" {
		log.Fatal("REDIS_IP must be set")
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

	if os.Getenv("SCHEDULER_PORT") == "" {
		log.Fatal("SCHEDULER_PORT must be set")
	}

	quit = make(chan bool)
	tasks = make(chan *Task)
	cpusPerTask, cpusParseErr = strconv.ParseFloat(os.Getenv("CPUS_PER_TASK"), 64)
	dockerCli, dockerCliErr = docker.NewClient(os.Getenv("DOCKER_SOCKET"))
	executorCommand = os.Getenv("EXECUTOR_COMMAND")
	executorId = os.Getenv("EXECUTOR_ID")
	frameworkName = os.Getenv("FRAMEWORK_NAME")
	mesosMasterIP = net.ParseIP(os.Getenv("MESOS_MASTER_IP"))
	redisIP = net.ParseIP(os.Getenv("REDIS_IP"))
	redisPort = os.Getenv("REDIS_PORT")
	mesosMasterPort = os.Getenv("MESOS_MASTER_PORT")
	gladiusIP = net.ParseIP(os.Getenv("GLADIUS_IP"))
	gladiusPort = os.Getenv("GLADIUS_PORT")
	memoryPerTask, memoryParseErr = strconv.ParseFloat(os.Getenv("MEMORY_PER_TASK"), 64)
	redisProtocol = os.Getenv("REDIS_PROTOCOL")
	redisPool = NewRedisPool()
	routes = NewRoutes()
	schedulerIP = net.ParseIP(os.Getenv("SCHEDULER_IP"))
	schedulerPort, schedulerParseErr = strconv.Atoi(os.Getenv("SCHEDULER_PORT"))
	schedulerDriver, schedulerDriverErr = NewSchedulerDriver()

	if dockerCliErr != nil {
		log.Fatal("Failed to initialize Docker: ", dockerCliErr)
	}

	if schedulerIP == nil {
		log.Fatal("Failed to parse SCHEDULER_IP: ", schedulerIP)
	}

	if redisIP == nil {
		log.Fatal("Failed to parse REDIS_IP: ", redisIP)
	}

	if mesosMasterIP == nil {
		log.Fatal("Failed to parse MESOS_MASTER_IP: ", os.Getenv("MESOS_MASTER_IP"))
	}

	if schedulerParseErr != nil {
		log.Fatal("Failed to parse SCHEDULER_PORT: %v", schedulerParseErr)
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

	go func() {
		err := http.ListenAndServe(fmt.Sprintf("%s:%s", gladiusIP, gladiusPort), nil)

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
