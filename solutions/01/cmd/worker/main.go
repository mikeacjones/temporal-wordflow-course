// Command worker runs your Workflows and Activities.
package main

import (
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"wordflow/solutions/01/shared"
	"wordflow/solutions/01/workflows"
)

func main() {
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalln("unable to connect to Temporal:", err)
	}
	defer c.Close()

	w := worker.New(c, shared.TaskQueue, worker.Options{})
	w.RegisterWorkflow(workflows.GameWorkflow)

	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalln("worker stopped:", err)
	}
}
