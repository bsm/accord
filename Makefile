default: test

test:
	go test ./...

.PHONY: default test

release:
	goreleaser --rm-dist

# proto ---------------------------------------------------------------

proto: rpc.go
rpc.go: rpc/accord.pb.go

.PHONY: proto rpc.go

%.pb.go: %.proto
	protoc --go_out=plugins=grpc,paths=source_relative:. $<
