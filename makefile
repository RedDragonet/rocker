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

ifeq '$(findstring ;,$(PATH))' ';'
    detected_OS := Windows
else
    detected_OS := $(shell uname 2>/dev/null || echo Unknown)
    detected_OS := $(patsubst CYGWIN%,Cygwin,$(detected_OS))
    detected_OS := $(patsubst MSYS%,MSYS,$(detected_OS))
    detected_OS := $(patsubst MINGW%,MSYS,$(detected_OS))
endif

#MACOS
## x86_64-linux-musl-gcc =》brew install FiloSottile/musl-cross/musl-cross
ifeq ($(detected_OS),Darwin)
	CROSS=GOARCH=amd64 CC=x86_64-linux-musl-gcc  CGO_ENABLED=1
	GOBUILD_OPTION=-ldflags "-extldflags -static"
endif

build:
		$(CROSS) $(GOBUILD) -o $(BINARY_NAME) $(GOBUILD_OPTION)

build2remote: build
		scp -P$(REMOTE_PORT) rocker root@$(REMOTE_IP):$(REMOTE_PATH)


