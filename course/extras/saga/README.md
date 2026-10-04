# Extra · Saga

**Goal:** players can gift points to each other. If the gift can't be delivered, the sender gets their points back.

## Concepts

**No transactions across Workflows.** A gift changes two Player Workflows. There's no way to change both in one step: the first can succeed and the second fail.

**Saga.** Run the steps in order. After each step succeeds, record a **compensation** that undoes it. If a later step fails, run the recorded compensations in reverse order.

**Compensations must finish.** They're Activities, so they're retried. Make them safe to repeat. Run them in a context that isn't cancelled, so they still run if the Workflow itself is cancelled.

**IDs for each step.** Each step's Update ID is built from the transfer's Run ID and the step name. A retried step can't apply twice, but two gifts between the same players are still separate.

## Patterns

- [Saga](https://docs.temporal.io/design-patterns/saga-pattern)

## What you'll build

- `withdraw` and `deposit` Updates on `PlayerWorkflow`, with validators.
- `Withdraw` and `Deposit` Activities. A rejection or a missing player is non-retryable.
- `TransferPointsWorkflow`: withdraw from the sender, record "refund the sender", deposit to the recipient. On failure, run the compensations.
- A small command-line client that runs a transfer and prints the result.

## Build it

Follow [`build.md`](build.md).

## Check it

Join as `alice` and `bob` first.

- [ ] Gift 5 points from `alice` to `bob`. It succeeds, and the player panels show 15 and 25.
- [ ] Gift 5 from `alice` to `nobody`. It fails with **no player named nobody**, and Alice still has 15. The transfer's history shows `Withdraw` completed, `Deposit` failed, and then a second `Deposit`, the refund, completed.
- [ ] Gift 100 from `alice`. It fails with **not enough points**. Nothing was withdrawn, so there's nothing to compensate.
- [ ] Gift 3 from `alice` to `bob` again. It succeeds: a new run with new Update IDs.
