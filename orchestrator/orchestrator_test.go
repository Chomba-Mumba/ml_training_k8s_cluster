package main

import (
	"context"
	"encoding/json"
	"fmt"
	"orchestrator/internal/collector"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"net/http"
	"net/http/httptest"
)

type migrationPostBody struct {
	Fitness string
	Body    string
}

type mockSQSClient struct {
	RecMsgFunc func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	DelMsgFunc func(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

func (m *mockSQSClient) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	return &sqs.ReceiveMessageOutput{
		Messages: []types.Message{
			{Body: aws.String("mock message")},
		},
	}, nil
}

func (m *mockSQSClient) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	return &sqs.DeleteMessageOutput{}, nil
}

func MockServer() (*httptest.Server, *http.Client) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("mock testing server request at: %v", r.URL)

		if r.URL.Path != "/migrant" {
			fmt.Printf("expected to request '/migrant', got: %s", r.URL.Path)
		}

		var req migrationPostBody

		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Printf("Error in POST body: %v", err)
		}

		if r.Header.Get("Accept") != "application/json" {
			fmt.Printf("Expected Accept: application/json header, got: %s", r.Header.Get("Accept"))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"value":"fixed"}`))
	}))

	client := server.Client()

	return server, client

}

func mockMessage(fitness int, hostname string, hyp map[string]interface{}, han string) collector.Message {
	msg := collector.Message{}
	msg.Fitness = fitness
	msg.Hostname = hostname
	msg.Hyperparameters = hyp
	msg.MessageHandle = han

	return msg
}

func TestMigHandler(t *testing.T) {
	server, client := MockServer()

	defer server.Close()

	mig := collector.Migrator{}
	mig.NewMigrator(client)

	var tests = []struct {
		name string
		msg  collector.Message
		want error
	}{
		//test cases
		{
			"single correct message",
			collector.Message{
				Fitness:         4,
				Hostname:        "1",
				Hyperparameters: map[string]interface{}{"h1": "test", "h2": "test"},
				MessageHandle:   "handle",
			},
			nil,
		},
	}

	//execut subtests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := mig.MigHandler(tt.msg)
			if resp != tt.want {
				t.Errorf("got %s, want %s", resp, tt.want)
			}
		})
	}
}

func TestMonitorHandler(t *testing.T) {
	mon := collector.Monitor{}
	server, client := MockServer()

	mon.NewMonitor(client)
	defer server.Close()

	var tests = []struct {
		name string
		msg  collector.Message
		want error
	}{
		//test cases
		{
			"single correct message",
			collector.Message{
				Fitness:         4,
				Hostname:        "1",
				Hyperparameters: map[string]interface{}{"h1": "test", "h2": "test"},
				MessageHandle:   "handle",
			},
			nil,
		},
	}

	//execute subtests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := mon.MonHandler(tt.msg)
			if resp != tt.want {
				t.Errorf("got %s, want %s", resp, tt.want)
			}
		})
	}

}

func TestNewCollector(t *testing.T) {
	t.Setenv("TOTAL_ISLANDS", "4")
	t.Setenv("WORKER_POOL", "10")
	server, client := MockServer()

	defer server.Close()
	var tests = []struct {
		name string
	}{
		{"correct test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mig := collector.Migrator{}
			mig.NewMigrator(client)

			//init monitor
			mon := collector.Monitor{}
			mon.NewMonitor(client)

			var mockClient collector.SQSAPI = &mockSQSClient{}
			println("creating new collector")
			coll := collector.NewCollector(mockClient)
			println("New Collector has been initalised...")

			workers := []*collector.Worker{mig.Worker, mon.Worker}
			coll.Collect(workers)

		})
	}

}
