.PHONY: test install

test:
	go test ./...

install:
	go install github.com/fujiwara/tfstate-lookup/cmd/tfstate-lookup
