package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	//TODO - mount IRSA to pod for AWS SDK creds
	//TODO - sort out warnings

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type Collector struct {
	FitQueueUrl string

	SqsClient   *sqs.Client
	maxMessages int32
	waitTime    int32
	workerPool  int32
}
type message struct {
	fitness         int                    `json:"fitness"`
	hyperparameters map[string]interface{} `json:"hyperparameters"`
	hostname        string                 `json:"hostname"`
	messageHandle   *string
}

type Worker struct {
	source   chan message
	quit     chan struct{}
	function string
	handler  func(msg message) error
}

func dispatch(msg message, workers []Worker) {
	//broadcast message to each workers source channel
	for _, worker := range workers {
		worker.source <- msg //send message to channel
	}
}

func (w *Worker) Start(handler func(msg message) error, quit_channel chan struct{}) {
	w.source = make(chan message, 10) //buffer to avoid blocking
	w.quit = quit_channel
	//TODO - sync.waitgroup
	go func() {
		for {
			select {
			case msg := <-w.source:
				handler(msg)
			case <-w.quit:
				return
			}
		}
	}()
}

func NewCollector() Collector {
	fQ := os.Getenv("FIT_QUEUE_URL")

	wP, err := strconv.Atoi(os.Getenv("WORKER_POOL"))
	if err != nil {
		panic(err)
	}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	client := sqs.NewFromConfig(cfg)

	c := Collector{FitQueueUrl: fQ, SqsClient: client,
		maxMessages: 10, waitTime: 1, workerPool: int32(wP)}

	return c
}

func newMessage(response types.Message) message {
	// receive message in JSON format and return s
	body := response.Body
	m := message{}
	json.Unmarshal([]byte(*body), &m)
	return m
}

func (c *Collector) delete(m message) error {
	// remove message from sqs queue
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := c.SqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      &c.FitQueueUrl,
		ReceiptHandle: m.messageHandle,
	})
	if err != nil {
		return fmt.Errorf("failed to delete message from sqs queue:%v", err)
	}
	return nil
}

func (c *Collector) Collect(workers []Worker) {
	globalQuit := make(chan struct{})
	//start workers
	for _, w := range workers {
		w.Start(w.handler, globalQuit)
	}

	//create worker pool for listening to messages
	for w := 1; w <= int(c.workerPool); w++ {
		go c.listener(w, workers)
	}

	//TODO - close global quit
}

func (c *Collector) listener(id int, workers []Worker) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for {
		result, err := c.SqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(c.FitQueueUrl),
			MaxNumberOfMessages: c.maxMessages,
			WaitTimeSeconds:     c.waitTime,
		})

		if err != nil {
			continue
		}

		var wg sync.WaitGroup
		for _, m := range result.Messages {
			wg.Add(1)
			go func(m message) {
				defer wg.Done()
				dispatch(m, workers)
				c.delete(m)
			}(newMessage(m))
		}
		wg.Wait()
	}
}
