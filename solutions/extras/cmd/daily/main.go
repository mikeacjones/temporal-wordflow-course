// Command daily manages the puzzle-of-the-day Schedule.
//
//	go run ./app/cmd/daily -every 1m    create the Schedule
//	go run ./app/cmd/daily -delete      delete it
//	go run ./app/cmd/daily -in 30s      start one run after a delay, no Schedule
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"go.temporal.io/sdk/client"

	"wordflow/solutions/extras/shared"
	"wordflow/solutions/extras/workflows"
)

func main() {
	every := flag.Duration("every", 0, "create a Schedule that runs this often")
	del := flag.Bool("delete", false, "delete the Schedule")
	in := flag.Duration("in", 0, "start a single run after this delay")
	flag.Parse()

	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalln("unable to connect to Temporal:", err)
	}
	defer c.Close()
	ctx := context.Background()

	switch {
	case *every > 0:
		_, err := c.ScheduleClient().Create(ctx, client.ScheduleOptions{
			ID: shared.DailyScheduleID,
			Spec: client.ScheduleSpec{
				Intervals: []client.ScheduleIntervalSpec{{Every: *every}},
			},
			Action: &client.ScheduleWorkflowAction{
				ID:        "daily-puzzle-run",
				Workflow:  workflows.DailyPuzzleWorkflow,
				TaskQueue: shared.TaskQueue,
			},
		})
		if err != nil {
			log.Fatalln("unable to create schedule:", err)
		}
		fmt.Printf("created Schedule %s: runs every %v\n", shared.DailyScheduleID, *every)

	case *del:
		if err := c.ScheduleClient().GetHandle(ctx, shared.DailyScheduleID).Delete(ctx); err != nil {
			log.Fatalln("unable to delete schedule:", err)
		}
		fmt.Println("deleted Schedule", shared.DailyScheduleID)

	case *in > 0:
		run, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
			ID:         fmt.Sprintf("daily-puzzle-once-%d", time.Now().Unix()),
			TaskQueue:  shared.TaskQueue,
			StartDelay: *in,
		}, workflows.DailyPuzzleWorkflow)
		if err != nil {
			log.Fatalln("unable to start:", err)
		}
		fmt.Printf("started %s: it runs in %v\n", run.GetID(), *in)

	default:
		flag.Usage()
	}
}
