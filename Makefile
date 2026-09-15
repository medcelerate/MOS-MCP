.PHONY: all build test vet css run tidy

all: build

# Regenerate the embedded Tailwind CSS for the admin console.
css:
	sh scripts/build-css.sh

build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

run:
	go run ./cmd/mos-mcp --config config.yaml
