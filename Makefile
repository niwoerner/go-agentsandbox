.PHONY: build test test-unit test-integration test-linux clean

build:
	go build ./...

test: test-unit test-integration

test-unit:
	go test -v ./...

test-integration:
	go test -v -tags=integration ./sandbox/

# Run Linux integration tests in Docker container with bubblewrap
test-linux:
	docker run --rm \
		--privileged \
		-v $(PWD):/app \
		-w /app \
		golang:1.24 \
		sh -c "apt-get update && apt-get install -y bubblewrap && go test -v -tags=integration ./sandbox/"

clean:
	rm -f /tmp/agentsandbox
	go clean ./...
