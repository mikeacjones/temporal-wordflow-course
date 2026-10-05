# Run your code:            make worker / make server
# Run a lesson's solution:  make worker S=05 / make server S=05
S ?=
SRC := $(if $(S),./solutions/$(S),./app)

.PHONY: temporal worker server replay history clean-workflows reset-temporal build course

temporal:
	mkdir -p .temporal
	temporal server start-dev --db-filename .temporal/temporal.db --ip 0.0.0.0

worker:
	go run $(SRC)/cmd/worker

server:
	go run $(SRC)/cmd/server

# Lesson 12: replay every saved history against your current Workflow code.
replay:
	go run $(SRC)/cmd/replay histories/*.json

# Lesson 12: save a Workflow's history.  make history ID=game-alice-1
history:
	mkdir -p histories
	temporal workflow show --workflow-id $(ID) --output json > histories/$(ID).json
	@echo "saved histories/$(ID).json"

# Terminate every running Workflow. Run this when new Workflow code cannot
# replay old executions (you will see "nondeterministic" in the Worker log).
clean-workflows:
	temporal workflow terminate --query 'ExecutionStatus="Running"' --reason "clean slate" --yes

# Delete every Workflow by wiping the dev server's database. Stop `make temporal` first.
reset-temporal:
	rm -rf .temporal

build:
	go build ./...
	go vet ./...

# The course website on port 8000. The Codespace starts it for you.
course:
	go run ./internal/coursesite
