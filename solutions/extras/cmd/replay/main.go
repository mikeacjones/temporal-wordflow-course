// Command replay checks saved Workflow histories against the current Workflow code.
//
//	go run ./app/cmd/replay histories/*.json
package main

import (
	"fmt"
	"log/slog"
	"os"

	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/worker"

	"wordflow/solutions/extras/workflows"
)

func main() {
	replayer := worker.NewWorkflowReplayer()
	replayer.RegisterWorkflow(workflows.GameWorkflow)
	replayer.RegisterWorkflow(workflows.PlayerWorkflow)
	replayer.RegisterWorkflow(workflows.LeaderboardWorkflow)
	replayer.RegisterWorkflow(workflows.TransferPointsWorkflow)
	replayer.RegisterWorkflow(workflows.PuzzleSubmissionWorkflow)
	replayer.RegisterWorkflow(workflows.DailyPuzzleWorkflow)

	// The error returned below says everything we need, so discard the SDK's logs.
	quiet := log.NewStructuredLogger(slog.New(slog.DiscardHandler))

	failed := false
	for _, path := range os.Args[1:] {
		if err := replayer.ReplayWorkflowHistoryFromJSONFile(quiet, path); err != nil {
			fmt.Printf("FAIL %s\n     %v\n", path, err)
			failed = true
			continue
		}
		fmt.Printf("ok   %s\n", path)
	}
	if failed {
		os.Exit(1)
	}
}
