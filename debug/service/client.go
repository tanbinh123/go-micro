// Package service provides the service log
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/micro/go-micro/client"

	"github.com/micro/go-micro/debug/log"
	pb "github.com/micro/go-micro/debug/service/proto"
)

// Debug provides debug service client
type debugClient struct {
	Client pb.DebugService
}

// NewClient provides a debug client
func NewClient(name string) *debugClient {
	// create default client
	cli := client.DefaultClient

	return &debugClient{
		Client: pb.NewDebugService(name, cli),
	}
}

// Logs queries the services logs and returns a channel to read the logs from
func (d *debugClient) Log(since time.Time, count int, stream bool) (log.Stream, error) {
	req := &pb.LogRequest{}
	if !since.IsZero() {
		req.Since = since.Unix()
	}

	if count > 0 {
		req.Count = int64(count)
	}

	// set whether to stream
	req.Stream = stream

	// get the log stream
	serverStream, err := d.Client.Log(context.Background(), req)
	if err != nil {
		return nil, fmt.Errorf("failed getting log stream: %s", err)
	}

	lg := &logStream{
		stream: make(chan log.Record),
		stop:   make(chan bool),
	}

	// go stream logs
	go d.streamLogs(lg, serverStream)

	return lg, nil
}

func (d *debugClient) streamLogs(lg *logStream, stream pb.Debug_LogService) {
	defer stream.Close()
	defer lg.Stop()

	for {
		resp, err := stream.Recv()
		if err != nil {
			break
		}

		metadata := make(map[string]string)
		for k, v := range resp.Metadata {
			metadata[k] = v
		}

		record := log.Record{
			Timestamp: time.Unix(resp.Timestamp, 0),
			Value:     resp.Value,
			Metadata:  metadata,
		}

		select {
		case <-lg.stop:
			return
		case lg.stream <- record:
		}
	}
}