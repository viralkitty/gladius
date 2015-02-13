package main

import (
	"encoding/json"
	"log"

	"github.com/gogo/protobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
)

type Scheduler struct {
	executor          *mesos.ExecutorInfo
	tasksLaunched     int
	tasksFinished     int
	taskStatusesChans map[string]chan *mesos.TaskStatus
}

func NewScheduler(exec *mesos.ExecutorInfo) *Scheduler {
	return &Scheduler{
		executor:          exec,
		tasksLaunched:     0,
		tasksFinished:     0,
		taskStatusesChans: make(map[string]chan *mesos.TaskStatus),
	}
}

func (sched *Scheduler) Registered(driver sched.SchedulerDriver, frameworkId *mesos.FrameworkID, masterInfo *mesos.MasterInfo) {
	log.Printf("Framework Registered with Master %v", masterInfo)
}

func (sched *Scheduler) Reregistered(driver sched.SchedulerDriver, masterInfo *mesos.MasterInfo) {
	log.Printf("Framework Re-Registered with Master %v", masterInfo)
}

func (sched *Scheduler) Disconnected(sched.SchedulerDriver) {
	log.Printf("Disconnected")
}

func (sched *Scheduler) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	for _, offer := range offers {
		cpuResources := util.FilterResources(offer.Resources, func(res *mesos.Resource) bool {
			return res.GetName() == "cpus"
		})
		cpus := 0.0
		for _, res := range cpuResources {
			cpus += res.GetScalar().GetValue()
		}

		memResources := util.FilterResources(offer.Resources, func(res *mesos.Resource) bool {
			return res.GetName() == "mem"
		})
		mems := 0.0
		for _, res := range memResources {
			mems += res.GetScalar().GetValue()
		}

		log.Printf("Received Offer <%v> with cpus=%d mem=%d", offer.Id.GetValue(), cpus, mems)

		cpusLeft := cpus
		memsLeft := mems
		task := <-tasks

		log.Printf("got task: %+v", task)

		if cpusPerTask <= cpusLeft && memPerTask <= memsLeft {
			log.Printf("about to marshal task: %+v", task)

			taskJsonBytes, err := json.Marshal(task)

			if err != nil {
				log.Printf("Could not marshal task")
				return
			}

			sched.tasksLaunched++

			taskId := &mesos.TaskID{
				Value: proto.String(task.Id),
			}

			sched.taskStatusesChans[task.Id] = task.Build.TaskStatusesChan

			taskInfo := &mesos.TaskInfo{
				Name:     proto.String(task.Id),
				TaskId:   taskId,
				SlaveId:  offer.SlaveId,
				Data:     taskJsonBytes,
				Executor: sched.executor,
				Resources: []*mesos.Resource{
					util.NewScalarResource("cpus", cpusPerTask),
					util.NewScalarResource("mem", memPerTask),
				},
			}

			log.Printf("Launching task: %s with offer %s\n", taskInfo.GetName(), offer.Id.GetValue())

			driver.LaunchTasks([]*mesos.OfferID{offer.Id}, []*mesos.TaskInfo{taskInfo}, &mesos.Filters{RefuseSeconds: proto.Float64(1)})
		}
	}
}

func (sched *Scheduler) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	statusChan := sched.taskStatusesChans[status.TaskId.GetValue()]

	log.Printf("Status update: task %v is in state %s", status.TaskId.GetValue(), status.State.Enum().String())

	go func() { statusChan <- status }()

	if status.GetState() == mesos.TaskState_TASK_LOST ||
		status.GetState() == mesos.TaskState_TASK_KILLED ||
		status.GetState() == mesos.TaskState_TASK_FAILED {
		log.Printf("Task %v is in unexpected state %s with message %s", status.TaskId.GetValue(), status.State.String(), status.GetMessage())
	}
}

func (sched *Scheduler) OfferRescinded(sched.SchedulerDriver, *mesos.OfferID) {
	log.Printf("Offer rescinded")
}

func (sched *Scheduler) FrameworkMessage(sched.SchedulerDriver, *mesos.ExecutorID, *mesos.SlaveID, string) {
	log.Printf("Framework received message")
}

func (sched *Scheduler) SlaveLost(sched.SchedulerDriver, *mesos.SlaveID) {
	log.Printf("Slave lost")
}

func (sched *Scheduler) ExecutorLost(sched.SchedulerDriver, *mesos.ExecutorID, *mesos.SlaveID, int) {
	log.Printf("Executor lost")
}

func (sched *Scheduler) Error(driver sched.SchedulerDriver, err string) {
	log.Printf("Scheduler received error: %v", err)
}
