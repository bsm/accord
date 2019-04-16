default: test

test:
	go test ./...

.PHONY: default test

# proto ---------------------------------------------------------------

proto: proto.go
proto.go: internal/proto/accord.pb.go

.PHONY: proto proto.go proto.deps

%.pb.go: %.proto
	protoc --go_out=plugins=grpc,import_path=internal/proto:. --proto_path=.:$$GOPATH/src $<
