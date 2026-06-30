.PHONY: build test install-local clean

build:
	go build -o workflow-plugin-encrypted-spaces ./cmd/workflow-plugin-encrypted-spaces

test:
	go test ./...

install-local: build
	wfctl plugin install --local .

clean:
	rm -f workflow-plugin-encrypted-spaces
