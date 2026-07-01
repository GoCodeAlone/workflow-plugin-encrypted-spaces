.PHONY: build test generate-contracts install-local clean

build:
	go build -o workflow-plugin-encrypted-spaces ./cmd/workflow-plugin-encrypted-spaces

test:
	go test ./...

generate-contracts:
	protoc --go_out=. --go_opt=paths=source_relative internal/contracts/spaces.proto

install-local: build
	wfctl plugin install --local .

clean:
	rm -f workflow-plugin-encrypted-spaces
