package mapreduce

import (
	"fmt"
	"sync"
	"time"
)

//
// schedule() starts and waits for all tasks in the given phase (mapPhase
// or reducePhase). the mapFiles argument holds the names of the files that
// are the inputs to the map phase, one per map task. nReduce is the
// number of reduce tasks. the registerChan argument yields a stream
// of registered workers; each item is the worker's RPC address,
// suitable for passing to call(). registerChan will yield all
// existing registered workers (if any) and new ones as they register.
//
func schedule(jobName string, mapFiles []string, nReduce int, phase jobPhase, registerChan chan string) {
	var ntasks int
	var n_other int // number of inputs (for reduce) or outputs (for map)

	// Adding a wait group to prevent using a channel when it is already being used
	var wait sync.WaitGroup

	switch phase {
	case mapPhase:
		ntasks = len(mapFiles)
		n_other = nReduce
		fmt.Printf("Map Schedule: %v %v tasks (%d I/Os)\n", ntasks, phase, n_other)

	case reducePhase:
		ntasks = nReduce
		n_other = len(mapFiles)
		fmt.Printf("Reduce Schedule: %v %v tasks (%d I/Os)\n", ntasks, phase, n_other)

	}

	// Creating a buffered channel for tasks, size is equal than number of tasks to keep it non-blocking
	var task = make(chan int, ntasks)

	// Assigning tasks
	for taskNumber := 0; taskNumber < ntasks; taskNumber++ {
		task <- taskNumber
	}
	// Closing the task channel
	close(task)

	// Retrieving workers from registerChan channel
	for channel := range registerChan {
		// When all the tasks are done, the registerChan gets "End" value signifying to stop polling for more workers
		if channel == "End" {
			break
		}
		// Creating a local channel variable so that a unique channel is given to a goroutine
		channel := channel
		// Incrementing wait value context of which is that this goroutine needs to complete before this channel is reassigned to other routine
		wait.Add(1)
		// Goroutine to perform a reduce operation
		go PerformOperation(&wait, registerChan, channel, jobName, mapFiles, phase, task, n_other, ntasks)
	}
	// Waiting for all routines to finish before task ends
	wait.Wait()

	// All ntasks tasks have to be scheduled on workers. Once all tasks
	// have completed successfully, schedule() should return.
	//
	// Your code here (Part III, Part IV).
	//
	fmt.Printf("Schedule: %v done\n", phase)
}

// A function common to both map and reduce phase, differing in fileName and phase name
func PerformOperation(wait *sync.WaitGroup, registerChan chan<- string, worker string, jobName string, mapFiles []string, phase jobPhase, task <-chan int, nOther int, ntasks int) {
	// Perform tasks contained in the task channel
	for taskNumber := range task {

		// Let this routine sleep to spawn or process any other routine going in parallel
		time.Sleep(time.Millisecond)

		// fileName take from the mapFiles array at the taskNumber index
		fileName := mapFiles[taskNumber]

		// For a reduce phase, fileName should not be given
		if phase != "mapPhase" {
			fileName = ""
		}

		// RPC Call made to the worker to performa map or reduce task
		call(worker, "Worker.DoTask", DoTaskArgs{jobName, fileName, phase, taskNumber, nOther}, nil)

		// If last task is done, then signal to stop polling for more workers using the registerChan channel
		if taskNumber == ntasks-1 {
			registerChan <- "End"
		}
	}
	// Wait is decremented signifying that the task of this routine is done
	wait.Done()
}

// A test function which can be used to add a new worker while map or reduce task is in progress
func TestNewWorkerInChannel(task chan<- int, registerChan chan<- string) {

	// A test worker
	newWorker := "/var/tmp/824-501/mr929-worker0"

	// Add new worker after 5 seconds when all map and reduce tasks are done
	time.Sleep(5 * time.Second)
	fmt.Println("Adding new tasks")

	// Add new tasks in the task channel
	for taskNumber := 0; taskNumber <= 5; taskNumber++ {
		//fmt.Println("Main: taskNumber - ", taskNumber)
		task <- taskNumber
	}

	close(task)
	fmt.Println("Adding a new worker")
	// Add a new worker to registerChan channel
	registerChan <- newWorker

}
