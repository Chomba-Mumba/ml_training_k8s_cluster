package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"orchestrator/internal/collector"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"net/http"
)

type migrationRequest struct {
	Fitness         int                    `json:"fitness"`
	Hyperparameters map[string]interface{} `json:"hyperparameters"`
}

type mockSQSClient struct {
	RecMsg func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	DelMsg func(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

func (m *mockSQSClient) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	return m.RecMsg(ctx, params, optFns...)
}

func (m *mockSQSClient) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	return &sqs.DeleteMessageOutput{}, nil
}

type mockHTTPClientInterface struct {
	GetReq  func(url string) (resp *http.Response, err error)
	PostReq func(url, contentType string, body io.Reader) (resp *http.Response, err error)
}

func (m mockHTTPClientInterface) Get(url string) (*http.Response, error) {
	return m.GetReq(url)
}

func (m mockHTTPClientInterface) Post(url, contentType string, body io.Reader) (*http.Response, error) {
	return m.PostReq(url, contentType, body)
}

var sharedHttpMock = func(t *testing.T) collector.HTTPClientInterface {
	resp := http.Response{
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Body:       io.NopCloser(bytes.NewBufferString("dummy body")),
		Header:     make(http.Header),
	}

	t.Helper()

	return mockHTTPClientInterface{
		GetReq: func(url string) (*http.Response, error) {
			resp.Status = "200 OK"
			resp.StatusCode = 200
			t.Logf("GET request made to: %s", url)
			return &resp, nil
		},
		PostReq: func(url, contentType string, body io.Reader) (*http.Response, error) {
			var r migrationRequest
			err := json.NewDecoder(body).Decode(&r)
			if err != nil {
				resp.StatusCode = http.StatusBadRequest
				return &resp, err
			}
			resp.Status = "200 OK"
			resp.StatusCode = 200
			t.Logf("POST request made to %s", url)
			return &resp, nil
		},
	}
}

func TestMigHandler(t *testing.T) {
	var tests = []struct {
		name         string
		client       func(t *testing.T) collector.HTTPClientInterface
		msg          collector.Message
		totalIslands string
		want         error
	}{
		//test cases
		{
			name:   "single correct message",
			client: sharedHttpMock,
			msg: collector.Message{
				Fitness:         4,
				Hostname:        "1",
				Hyperparameters: map[string]interface{}{"h1": "test", "h2": "test"},
				MessageHandle:   "handle",
			},
			totalIslands: "10",
			want:         nil,
		},
	}

	//execute subtests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//initialise migrator
			mig := collector.Migrator{}
			t.Setenv("TOTAL_ISLANDS", tt.totalIslands)

			mig.NewMigrator(tt.client(t))

			resp := mig.MigHandler(tt.msg)
			if resp != tt.want {
				t.Errorf("got %s, want %s", resp, tt.want)
			}
		})
	}
}

func TestMonitorHandler(t *testing.T) {
	mon := collector.Monitor{}

	var tests = []struct {
		name   string
		client func(t *testing.T) collector.HTTPClientInterface
		msg    collector.Message
		want   error
	}{
		//test cases
		{
			name:   "single correct message",
			client: sharedHttpMock,
			msg: collector.Message{
				Fitness:         4,
				Hostname:        "1",
				Hyperparameters: map[string]interface{}{"h1": "test", "h2": "test"},
				MessageHandle:   "handle",
			},
			want: nil,
		},
	}

	//execute subtests
	for _, tt := range tests {
		mon.NewMonitor(tt.client(t))

		t.Run(tt.name, func(t *testing.T) {
			resp := mon.MonHandler(tt.msg)
			if resp != tt.want {
				t.Errorf("got %s, want %s", resp, tt.want)
			}
		})
	}

}

func TestCollect(t *testing.T) {
	t.Setenv("TOTAL_ISLANDS", "4")
	t.Setenv("WORKER_POOL", "10")
	var tests = []struct {
		name       string
		httpClient func(t *testing.T) collector.HTTPClientInterface
		msg        string
		msgs       int //number of messages
		want       error
	}{
		{
			name:       "single correct message",
			httpClient: sharedHttpMock,
			msg: `{
					"Fitness": 4,
					"Hostname": "1",
					"Hyperparameters": {
						"h1": "test",
						"h2": "test"
					},
					"MessageHandle": "handle"
				}`,
			msgs: 1,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mig := collector.Migrator{}
			mig.NewMigrator(tt.httpClient(t))

			//init monitor
			mon := collector.Monitor{}
			mon.NewMonitor(tt.httpClient(t))

			sqsClient := &mockSQSClient{
				RecMsg: func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {

					msgs := []types.Message{}
					for range tt.msgs {
						m := types.Message{Body: &tt.msg}
						msgs = append(msgs, m)
					}
					return &sqs.ReceiveMessageOutput{Messages: msgs}, nil
				},
			}

			coll := collector.NewCollector(sqsClient)

			workers := []*collector.Worker{mig.Worker, mon.Worker}
			coll.Collect(workers)

		})
	}

}
