package rpc

import (
	"time"

	_ "github.com/golang/protobuf/proto" // include in go mod
	_ "google.golang.org/grpc"           // include in go mod
)

// IsDone returns true if resource is marked as done.
func (h *Handle) IsDone() bool {
	return h.DoneTms != 0
}

// ExpTime converts ExpTms to time.Time.
func (h *Handle) ExpTime() time.Time {
	return millisToTime(h.ExpTms)
}

// DoneTime converts DoneTms to time.Time.
func (h *Handle) DoneTime() time.Time {
	return millisToTime(h.DoneTms)
}

func millisToTime(ms int64) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.Unix(ms/1e3, ms%1e3*1e6)
}
