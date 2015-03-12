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

	"github.com/fsouza/go-dockerclient"
	"github.com/garyburd/redigo/redis"
	"github.com/gogo/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
)

const (
	FRAMEWORK_NAME = "Gladius"
)

var pool *redis.Pool
var master = os.Getenv("MESOS_MASTER")
var execUri = os.Getenv("EXEC_URI")
var schedIP = net.ParseIP(os.Getenv("MESOS_SCHEDULER_IP"))
var schedPort, schedPortParseErr = strconv.Atoi(os.Getenv("MESOS_SCHEDULER_PORT"))
var cpusPerTask, cpusParseErr = strconv.ParseFloat(os.Getenv("CPUS_PER_TASK"), 64)
var memPerTask, memParseErr = strconv.ParseFloat(os.Getenv("MEM_PER_TASK"), 64)
var dockerCli, dockerCliErr = docker.NewClient(os.Getenv("DOCKER_SOCK_PATH"))
var tasks = make(chan *Task)

func init() {
	pool = newPool(os.Getenv("REDIS_TCP_ADDR"))

	log.Printf("Initializing the Example Scheduler...")

	if dockerCliErr != nil {
		log.Fatal("Failed to connect with Redis: %v", dockerCliErr)
	}

	if cpusParseErr != nil {
		log.Fatal("Failed to parse CPUS per task: %v", cpusParseErr)
	}

	if memParseErr != nil {
		log.Fatal("Failed to parse mem per task: %v", memParseErr)
	}

	if schedPortParseErr != nil {
		log.Fatal("Failed to parse scheduler ip: %v", schedPortParseErr)
	}

	rand.Seed(time.Now().UTC().UnixNano())
}

func main() {
	// build command executor
	exec := &mesos.ExecutorInfo{
		ExecutorId: util.NewExecutorID("default"),
		Command:    util.NewCommandInfo(execUri),
	}

	scheduler := NewScheduler(exec)

	fwinfo := &mesos.FrameworkInfo{
		User: proto.String(""),
		Name: proto.String(FRAMEWORK_NAME),
	}

	schedConfig := sched.DriverConfig{
		Scheduler:      scheduler,
		Framework:      fwinfo,
		Master:         master,
		BindingAddress: schedIP,
		BindingPort:    uint16(schedPort),
	}
	driver, err := sched.NewMesosSchedulerDriver(schedConfig)

	if err != nil {
		log.Printf("Unable to create a SchedulerDriver ", err.Error())
	}

	routes := &Routes{scheduler}
	listenAt := fmt.Sprintf("%s:%s", os.Getenv("GLADIUS_TCP_ADDR"), os.Getenv("GLADIUS_HTTP_PORT"))

	http.HandleFunc("/", routes.Home)
	http.HandleFunc("/builds", routes.Builds)

	go func() { log.Print(http.ListenAndServe(listenAt, nil)) }()

	log.Printf("Listening at %s", listenAt)

	if stat, err := driver.Run(); err != nil {
		log.Printf("Framework stopped with status %s and error: %s\n", stat.String(), err.Error())
	}
}

func newPool(server string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}
