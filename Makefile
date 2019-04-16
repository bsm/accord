default: test

test:
	go test ./...

.PHONY: default test

# proto ---------------------------------------------------------------

proto: proto.go
proto.go: internal/proto/accord.pb.go

proto.deps:
	go get -u github.com/golang/protobuf

.PHONY: proto proto.go proto.deps

%.pb.go: %.proto proto.deps
	protoc --go_out=plugins=grpc,import_path=internal/proto:. --proto_path=.:$$GOPATH/src $<
