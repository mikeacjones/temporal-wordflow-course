// Command worker runs your Workflows and Activities.
package main

import (
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"wordflow/internal/puzzles"
	"wordflow/solutions/04/activities"
	"wordflow/solutions/04/shared"
	"wordflow/solutions/04/workflows"
)

func main() {
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalln("unable to connect to Temporal:", err)
	}
	defer c.Close()

	w := worker.New(c, shared.TaskQueue, worker.Options{})
	w.RegisterWorkflow(workflows.GameWorkflow)
	w.RegisterActivity(&activities.PuzzleActivities{Store: puzzles.NewStore()})

	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalln("worker stopped:", err)
	}
}
