default: test

test:
	go test ./...

.PHONY: default test

lint:
	golangci-lint run

release-snapshot:
	goreleaser --rm-dist --snapshot

# proto ---------------------------------------------------------------

proto: rpc.go
rpc.go: rpc/accord.pb.go

.PHONY: proto rpc.go

%.pb.go: %.proto
	protoc --go_out=. --go_opt=paths=source_relative \
         --go-grpc_out=. --go-grpc_opt=paths=source_relative \
         $<
