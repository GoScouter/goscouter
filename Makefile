BINARY := gs
CMD    := ./cmd

.PHONY: build run clean fmt test

build:
	go build -ldflags "-X 'main.BUILD_TIME=$$(date +%Y-%m-%dT%H:%M:%S)'" -o $(BINARY) $(CMD)

run:
	go run -ldflags "-X 'main.BUILD_TIME=$$(date +%Y-%m-%dT%H:%M:%S)'" $(CMD)

fmt:
	go fmt ./...

test:
	go test ./...

clean:
	rm -f $(BINARY)
