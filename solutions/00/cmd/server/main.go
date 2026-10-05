// Command server runs the web UI and HTTP API on :8080.
package main

import (
	"log"

	"wordflow/internal/httpapi"
	"wordflow/internal/puzzles"
	"wordflow/solutions/00/api"
)

func main() {
	// Lesson 1: connect to Temporal and pass the client to api.New.
	backend := api.New(nil, puzzles.NewStore())
	log.Fatal(httpapi.Serve(":8080", backend))
}
