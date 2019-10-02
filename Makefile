GITHUB_TOKEN:=$(shell sed ':a;N;$!ba;s/\n/ /g' ~/.netrc | grep -Po 'machine *github.com *login *\w+' | grep -Po '\w+$$')

default: test

test:
	go test ./...

.PHONY: default test

release:
	GITHUB_TOKEN=$(GITHUB_TOKEN) goreleaser --rm-dist

# proto ---------------------------------------------------------------

proto: rpc.go
rpc.go: rpc/accord.pb.go

.PHONY: proto rpc.go

%.pb.go: %.proto
	protoc --go_out=plugins=grpc,import_path=rpc:. --proto_path=.:$$GOPATH/src $<
