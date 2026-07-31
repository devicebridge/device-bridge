BINARY=device-bridge

build:
	CGO_ENABLED=0 go build -o $(BINARY) ./cmd/device-bridge

run:
	go run ./cmd/device-bridge

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)
