// Command server runs the web UI and HTTP API on :8080.
package main

import (
	"log"

	"go.temporal.io/sdk/client"

	"wordflow/internal/httpapi"
	"wordflow/internal/puzzles"
	"wordflow/solutions/13/api"
)

func main() {
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalln("unable to connect to Temporal:", err)
	}
	defer c.Close()

	backend := api.New(c, puzzles.NewStore())
	log.Fatal(httpapi.Serve(":8080", backend))
}
