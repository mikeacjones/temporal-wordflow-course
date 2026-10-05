// Command worker runs your Workflows and Activities.
package main

import (
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"wordflow/internal/puzzles"
	"wordflow/solutions/08/activities"
	"wordflow/solutions/08/shared"
	"wordflow/solutions/08/workflows"
)

func main() {
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalln("unable to connect to Temporal:", err)
	}
	defer c.Close()

	w := worker.New(c, shared.TaskQueue, worker.Options{})
	w.RegisterWorkflow(workflows.GameWorkflow)
	w.RegisterWorkflow(workflows.PlayerWorkflow)
	w.RegisterActivity(&activities.PuzzleActivities{Store: puzzles.NewStore()})
	w.RegisterActivity(&activities.PlayerActivities{Client: c})

	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalln("worker stopped:", err)
	}
}
