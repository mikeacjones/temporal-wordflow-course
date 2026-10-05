// Command gift moves points from one player to another.
//
//	go run ./app/cmd/gift -from alice -to bob -amount 5
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"go.temporal.io/sdk/client"

	"wordflow/solutions/extras/shared"
	"wordflow/solutions/extras/workflows"
)

func main() {
	from := flag.String("from", "", "player giving points")
	to := flag.String("to", "", "player receiving points")
	amount := flag.Int("amount", 0, "points to give")
	flag.Parse()

	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalln("unable to connect to Temporal:", err)
	}
	defer c.Close()

	ctx := context.Background()
	run, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        fmt.Sprintf("gift-%s-%s", *from, *to),
		TaskQueue: shared.TaskQueue,
	}, workflows.TransferPointsWorkflow, shared.TransferInput{From: *from, To: *to, Amount: *amount})
	if err != nil {
		log.Fatalln("unable to start transfer:", err)
	}
	fmt.Println("started", run.GetID(), run.GetRunID())

	if err := run.Get(ctx, nil); err != nil {
		log.Fatalln("transfer failed:", err)
	}
	fmt.Printf("%s gave %s %d points\n", *from, *to, *amount)
}
