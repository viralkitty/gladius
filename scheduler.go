package main

import (
	"code.google.com/p/gogoprotobuf/proto"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
	"log"
	"strconv"
)

const (
	CPUS_PER_TASK = 1
	MEM_PER_TASK  = 128
)

type Scheduler struct {
	executor      *mesos.ExecutorInfo
	tasksLaunched int
	tasksFinished int
	chanchan      chan Task
	taskChans     map[string]chan mesos.TaskStatus
}

func NewScheduler(exec *mesos.ExecutorInfo) *Scheduler {
	return &Scheduler{
		executor:      exec,
		tasksLaunched: 0,
		tasksFinished: 0,
		taskChans:     make(map[string](chan mesos.TaskStatus)),
		chanchan:      make(chan Task),
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
	go func() {
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

			//cpusLeft := cpus
			//memsLeft := mems

			//for CPUS_PER_TASK <= cpusLeft && MEM_PER_TASK <= memsLeft {
			obj := <-sched.chanchan

			log.Printf("Got a obj %+v", obj)

			sched.tasksLaunched++

			taskId := &mesos.TaskID{
				Value: proto.String(strconv.Itoa(sched.tasksLaunched)),
			}

			sched.taskChans[taskId.GetValue()] = obj.Output

			task := &mesos.TaskInfo{
				Name:     proto.String("go-task-" + taskId.GetValue()),
				TaskId:   taskId,
				SlaveId:  offer.SlaveId,
				Data:     []byte(obj.Cmd),
				Executor: sched.executor,
				Resources: []*mesos.Resource{
					util.NewScalarResource("cpus", CPUS_PER_TASK),
					util.NewScalarResource("mem", MEM_PER_TASK),
				},
			}
			log.Printf("Launching task: %s with offer %s\n", task.GetName(), offer.Id.GetValue())

			driver.LaunchTasks([]*mesos.OfferID{offer.Id}, []*mesos.TaskInfo{task}, &mesos.Filters{RefuseSeconds: proto.Float64(1)})

			//	cpusLeft -= CPUS_PER_TASK
			//	memsLeft -= MEM_PER_TASK
			//}
		}
	}()
}

func (sched *Scheduler) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	go func() {
		log.Printf("Status update: task %v is in state %s", status.TaskId.GetValue(), status.State.Enum().String())

		if status.GetState() == mesos.TaskState_TASK_FINISHED {
			go func() {
				sched.taskChans[status.TaskId.GetValue()] <- *status
			}()

			sched.tasksFinished++
		}

		if status.GetState() == mesos.TaskState_TASK_LOST ||
			status.GetState() == mesos.TaskState_TASK_KILLED ||
			status.GetState() == mesos.TaskState_TASK_FAILED {

			log.Printf("Aborting because task %v is in unexpected state %s with message %s", status.TaskId.GetValue(), status.State.String(), status.GetMessage())

			go func() {
				sched.taskChans[status.TaskId.GetValue()] <- *status
			}()

			driver.Abort()
		}
	}()
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
