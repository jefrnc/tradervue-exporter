.PHONY: build run-export run-summary clean

build:
	go build -o bin/tvue ./cmd/tvue

run-export: build
	./bin/tvue export

run-summary: build
	./bin/tvue summary

clean:
	rm -rf bin/
