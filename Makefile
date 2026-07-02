VERSION ?= 0.6.0

.PHONY: build test pipeline-test generate-contracts install-local clean

build:
	go build -ldflags "-X github.com/GoCodeAlone/workflow-plugin-encrypted-spaces/internal.Version=$(VERSION)" -o workflow-plugin-encrypted-spaces ./cmd/workflow-plugin-encrypted-spaces

test:
	go test ./...

pipeline-test:
	VERSION="$(VERSION)" ./scripts/run-pipeline-tests.sh

generate-contracts:
	protoc --go_out=. --go_opt=paths=source_relative internal/contracts/spaces.proto

install-local: build
	wfctl plugin install --local .

clean:
	rm -f workflow-plugin-encrypted-spaces
