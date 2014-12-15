package main

import (
	"code.google.com/p/gogoprotobuf/proto"
	"flag"
	"fmt"
	log "github.com/golang/glog"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
	"net/http"
	"os"
)

const (
	FRAMEWORK_NAME = "Gladius"
)

var master = flag.String("master", "127.0.0.1:5050", "Master address <ip:port>")
var execUri = flag.String("executor", "/gladius/test-executor", "Path to test executor")
var builds = make(chan *Build)

func init() {
	flag.Parse()
	log.Infoln("Initializing the Example Scheduler...")
}

func main() {
	// routes
	routes := &Routes{
		builds: builds,
	}

	// build command executor
	exec := &mesos.ExecutorInfo{
		ExecutorId: util.NewExecutorID("default"),
		Command:    util.NewCommandInfo(*execUri),
	}

	// the framework
	fwinfo := &mesos.FrameworkInfo{
		User: proto.String(""), // Mesos-go will fill in user.
		Name: proto.String(FRAMEWORK_NAME),
	}

	driver, err := sched.NewMesosSchedulerDriver(
		NewScheduler(exec, builds),
		fwinfo,
		*master,
		nil,
	)

	if err != nil {
		log.Errorln("Unable to create a SchedulerDriver ", err.Error())
	}

	listenAt := fmt.Sprintf(":%s", os.Getenv("GLADIUS_HTTP_PORT"))

	log.Infoln("Listening at %s", listenAt)

	http.HandleFunc("/builds", routes.Builds)

	go http.ListenAndServe(listenAt, nil)

	if stat, err := driver.Run(); err != nil {
		log.Infof("Framework stopped with status %s and error: %s\n", stat.String(), err.Error())
	}
}
