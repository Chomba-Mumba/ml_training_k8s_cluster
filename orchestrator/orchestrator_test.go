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
	client collector.SQSClientInterface
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
		fmt.Println("mock testing server request @: %v", r.URL)

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

func TestMigHandler(t *testing.T) {
	server, client := MockServer()

	defer server.Close()

	m := collector.Migrator{}
	m.NewMigrator(client)
	handle := "handle"

	msg := collector.Message{}
	msg.Fitness = 32
	msg.Hostname = "1"
	msg.Hyperparameters = map[string]interface{}{"h1": "test", "h2": "test"}
	msg.MessageHandle = &handle

	err := m.MigHandler(msg)

	if err != nil {
		t.Errorf("expected nil, got an error %v", err)
	}
}

func TestNewCollector(t *testing.T) {
	server, client := MockServer()

	defer server.Close()
	mig := collector.Migrator{}
	mig.NewMigrator(client)

	//init monitor
	mon := collector.Monitor{}
	mon.NewMonitor()

	mock := mockSQSClient{}

	coll := collector.NewCollector(mock.client)

	workers := []*collector.Worker{mig.Worker, mon.Worker}
	coll.Collect(workers)
}
