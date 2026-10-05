// Command submit sends a new puzzle for review.
//
//	go run ./app/cmd/submit submissions/bread.json
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"go.temporal.io/sdk/client"

	"wordflow/internal/game"
	"wordflow/solutions/extras/shared"
	"wordflow/solutions/extras/workflows"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalln("usage: submit PUZZLE.json")
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}
	var puzzle game.Puzzle
	if err := json.Unmarshal(data, &puzzle); err != nil {
		log.Fatalln("bad puzzle file:", err)
	}

	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalln("unable to connect to Temporal:", err)
	}
	defer c.Close()

	id := shared.SubmissionWorkflowID(puzzle.ID)
	_, err = c.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{
		ID:        id,
		TaskQueue: shared.TaskQueue,
	}, workflows.PuzzleSubmissionWorkflow, shared.SubmissionInput{Puzzle: puzzle})
	if err != nil {
		log.Fatalln("unable to submit:", err)
	}
	fmt.Println("submitted", id)
	fmt.Printf("check it:  temporal workflow query --workflow-id %s --name %s\n", id, shared.QuerySubmission)
	fmt.Printf("approve:   temporal workflow signal --workflow-id %s --name %s --input '{\"approved\":true,\"reviewer\":\"you\"}'\n", id, shared.SignalReview)
}
