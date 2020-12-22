GOCMD=go
GOBUILD=GOOS=linux $(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOTIDY=$(GOCMD) mod tidy

BINARY_NAME=$(shell basename "$(PWD)")


REMOTE_IP=127.0.0.1
REMOTE_PORT=9922
REMOTE_PATH=/root/

.PHONY: build build2remote

build:
		$(GOBUILD) -o $(BINARY_NAME)

build2remote: build
		scp -P$(REMOTE_PORT) rocker root@$(REMOTE_IP):$(REMOTE_PATH)


