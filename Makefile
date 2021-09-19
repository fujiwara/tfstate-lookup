GOBIN ?= $(shell go env GOPATH)/bin

.PHONY: test install

test:
	go test ./...

install:
	go install github.com/fujiwara/tfstate-lookup/cmd/tfstate-lookup

CREDITS: $(GOBIN)/gocredits go.sum
	go mod tidy
	$(GOBIN)/gocredits -w .

$(GOBIN)/gocredits:
	go install github.com/Songmu/gocredits/cmd/gocredits@latest
