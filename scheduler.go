package main

import (
	"encoding/json"
	"log"
	"time"

	proto "github.com/gogo/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
)

const (
	offerTimeout = 1 * time.Minute
)

var (
	filters = &mesos.Filters{RefuseSeconds: proto.Float64(5)}
)

type Scheduler struct {
	executor          *mesos.ExecutorInfo
	tasksLaunched     int
	tasksFinished     int
	taskStatusesChans map[string]chan *mesos.TaskStatus
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		executor: &mesos.ExecutorInfo{
			ExecutorId: util.NewExecutorID(executorId),
			Command:    util.NewCommandInfo(executorCommand),
		},
		tasksLaunched:     0,
		tasksFinished:     0,
		taskStatusesChans: make(map[string]chan *mesos.TaskStatus),
	}
}

func (s *Scheduler) Registered(driver sched.SchedulerDriver, frameworkId *mesos.FrameworkID, masterInfo *mesos.MasterInfo) {
	log.Printf("Framework Registered with Master %v", masterInfo)
}

func (s *Scheduler) Reregistered(driver sched.SchedulerDriver, masterInfo *mesos.MasterInfo) {
	log.Printf("Framework Re-Registered with Master %v", masterInfo)
}

func (s *Scheduler) Disconnected(sched.SchedulerDriver) {
	log.Printf("Disconnected")
}

func (s *Scheduler) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	for _, offer := range offers {
		log.Printf("Received offer %s", offer.Id.GetValue())
		go s.handleOffer(offer)
	}
}

func (s *Scheduler) handleOffer(offer *mesos.Offer) {
	select {
	case task := <-tasks:
		log.Printf("Launching task %s for offer %s", task.Id, offer.Id.GetValue())
		s.launchTaskWithOffer(task, offer)
	default:
		log.Printf("No tasks available; Declining offer %s", offer.Id.GetValue())
		schedulerDriver.DeclineOffer(offer.Id, filters)
	}
}

func (s *Scheduler) launchTaskWithOffer(task *Task, offer *mesos.Offer) {
	mems := 0.0
	cpus := 0.0
	taskJsonBytes, err := json.Marshal(task)
	cpuResources := util.FilterResources(offer.Resources, func(res *mesos.Resource) bool {
		return res.GetName() == "cpus"
	})
	memResources := util.FilterResources(offer.Resources, func(res *mesos.Resource) bool {
		return res.GetName() == "mem"
	})

	for _, res := range cpuResources {
		cpus += res.GetScalar().GetValue()
	}

	for _, res := range memResources {
		mems += res.GetScalar().GetValue()
	}

	if err != nil {
		log.Printf("Declining offer %s for task %s due to marshal error: %v", offer.Id.GetValue(), task.Id, err)
		schedulerDriver.DeclineOffer(offer.Id, filters)

		return
	}

	if cpus < cpusPerTask || mems < memoryPerTask {
		log.Printf("Declining offer %s for task %s due to insufficient offer resources", offer.Id.GetValue(), task.Id)
		schedulerDriver.DeclineOffer(offer.Id, filters)

		return
	}

	taskId := &mesos.TaskID{
		Value: proto.String(task.Id),
	}

	taskInfo := &mesos.TaskInfo{
		Name:     proto.String(task.Cmd),
		TaskId:   taskId,
		SlaveId:  offer.SlaveId,
		Data:     taskJsonBytes,
		Executor: s.executor,
		Resources: []*mesos.Resource{
			util.NewScalarResource("cpus", cpusPerTask),
			util.NewScalarResource("mem", memoryPerTask),
		},
	}

	s.taskStatusesChans[task.Id] = task.Build.TaskStatusesChan

	schedulerDriver.LaunchTasks([]*mesos.OfferID{offer.Id}, []*mesos.TaskInfo{taskInfo}, filters)
}

func (s *Scheduler) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	statusChan := s.taskStatusesChans[status.TaskId.GetValue()]

	log.Printf("Status update: task %v is in state %s", status.TaskId.GetValue(), status.State.Enum().String())

	go func() { statusChan <- status }()

	if status.GetState() == mesos.TaskState_TASK_LOST ||
		status.GetState() == mesos.TaskState_TASK_KILLED ||
		status.GetState() == mesos.TaskState_TASK_FAILED {
		log.Printf("Task %v is in unexpected state %s with message %s", status.TaskId.GetValue(), status.State.String(), status.GetMessage())
	}
}

func (s *Scheduler) OfferRescinded(sched.SchedulerDriver, *mesos.OfferID) {
	log.Printf("Offer rescinded")
}

func (s *Scheduler) FrameworkMessage(sched.SchedulerDriver, *mesos.ExecutorID, *mesos.SlaveID, string) {
	log.Printf("Framework received message")
}

func (s *Scheduler) SlaveLost(sched.SchedulerDriver, *mesos.SlaveID) {
	log.Printf("Slave lost")
}

func (s *Scheduler) ExecutorLost(sched.SchedulerDriver, *mesos.ExecutorID, *mesos.SlaveID, int) {
	log.Printf("Executor lost")
}

func (s *Scheduler) Error(driver sched.SchedulerDriver, err string) {
	log.Printf("Scheduler received error: %v", err)
}
