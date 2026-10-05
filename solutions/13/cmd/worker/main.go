// Command worker runs your Workflows and Activities.
package main

import (
	"log"
	"net/http"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"wordflow/internal/puzzles"
	"wordflow/solutions/13/activities"
	"wordflow/solutions/13/shared"
	"wordflow/solutions/13/workflows"
)

func main() {
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalln("unable to connect to Temporal:", err)
	}
	defer c.Close()

	servicesURL := os.Getenv("SERVICES_URL")
	if servicesURL == "" {
		servicesURL = "http://localhost:8080/services"
	}

	w := worker.New(c, shared.TaskQueue, worker.Options{})
	w.RegisterWorkflow(workflows.GameWorkflow)
	w.RegisterWorkflow(workflows.PlayerWorkflow)
	w.RegisterWorkflow(workflows.LeaderboardWorkflow)
	w.RegisterActivity(&activities.PuzzleActivities{Store: puzzles.NewStore()})
	w.RegisterActivity(&activities.PlayerActivities{Client: c})
	w.RegisterActivity(&activities.DictionaryActivities{BaseURL: servicesURL + "/dictionary", HTTP: http.DefaultClient})
	w.RegisterActivity(&activities.FeedActivities{BaseURL: servicesURL + "/feed", HTTP: http.DefaultClient})

	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalln("worker stopped:", err)
	}
}
