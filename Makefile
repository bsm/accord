default: test

test:
	go test ./...

.PHONY: default test

# proto ---------------------------------------------------------------

proto: rpc.go
rpc.go: rpc/accord.pb.go

.PHONY: proto rpc.go

%.pb.go: %.proto
	protoc --go_out=plugins=grpc,import_path=rpc:. --proto_path=.:$$GOPATH/src $<
