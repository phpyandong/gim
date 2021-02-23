# Go parameters
GOCMD=GO111MODULE=on go
GOBUILD=$(GOCMD) build

build:
	rm -rf target/
	$(GOBUILD) -o target/comet    cmd/comet/main.go
	$(GOBUILD) -o target/logic    cmd/logic/main.go
	$(GOBUILD) -o target/registry cmd/registry/main.go

run:
	nohup target/registry  2>&1 > target/registry.log &
	nohup target/logic  2>&1 > target/logic.log &
	nohup target/comet  2>&1 > target/comet.log &

stop:
	pkill -f target/logic
	pkill -f target/comet
	pkill -f target/registry
