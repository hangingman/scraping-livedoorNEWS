# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean

BIN=scrape-livedoor

.PHONY: all

all: build

build:
	$(GOBUILD) -o $(BIN) main.go

clean:
	$(GOCLEAN)

run: build
	./$(BIN)

fmt:
	for go_file in `find . -name \*.go`; do \
		go fmt $${go_file}; \
	done
