# Accord

[![GoDoc](https://godoc.org/github.com/bsm/accord?status.svg)](https://godoc.org/github.com/bsm/accord)
[![Build Status](https://travis-ci.org/bsm/accord.svg)](https://travis-ci.org/bsm/accord)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Fast and efficient coordination of distributed processes.
Server and client implementation in [Go](https://golang.org/) on top of the [gRPC](https://grpc.io/) framework.

## Use Case

Imagine you are processing files within a directory or bucket, you would want to coordinate your workers
to pick a file each, lock it for the duration of the processing and then mark it as done once finished so
it's not picked up by another worker again.

## Architecture

    +------------+         +------------+     +--------------+
    |            |         |            |     |              |
    |   WORKER   | -gRPC-> |   SERVER   | --> |   BACKEND    |
    |   Client   | <------ |            | <-- |  PostgreSQL  |
    |            |         |            |     |              |
    +------------+         +------------+     +--------------+

* Clients are connected to (a cluster of) Accord Servers and communicate via gRPC.
* Servers are using a pluggable backend (currently, PostgreSQL only) to coordinate state.
* Clients maintain a local state cache to avoid hammering the server/backends.

## Documentation

Please see the [API documentation](https://godoc.org/github.com/bsm/accord) for
package and API descriptions and examples.
