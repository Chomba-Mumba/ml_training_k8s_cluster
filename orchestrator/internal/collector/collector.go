package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	//TODO - mount IRSA to pod for AWS SDK creds
	//TODO - sort out warnings

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type Collector struct {
	FitQueueUrl    string
	client         SQSClientInterface
	maxMessages    int32
	waitTime       int32
	workerPoolSize int32
}

type Message struct {
	Fitness         int                    `json:"fitness"`
	Hyperparameters map[string]interface{} `json:"hyperparameters"`
	Hostname        string                 `json:"hostname"`
	MessageHandle   *string
}

type Worker struct {
	source   chan Message
	quit     chan struct{}
	function string
	handler  func(msg Message) error
	close    func() error
}

type SQSClientInterface interface {
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

func dispatch(msg Message, workers []*Worker) {
	//broadcast message to each workers source channel
	for _, worker := range workers {
		worker.source <- msg
	}
}

// start go routine for each type of worker in workers slice
func (w *Worker) StartWorker(handler func(msg Message) error, quit_channel chan struct{}, wg *sync.WaitGroup) {
	w.source = make(chan Message, 10) //buffer to avoid blocking
	w.quit = quit_channel
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case msg := <-w.source:
				handler(msg)
			case <-w.quit:
				return
			}
		}
	}()
	wg.Done()
}

func NewCollector(cli SQSClientInterface) Collector {
	//process messages from training queues
	fQ := os.Getenv("FIT_QUEUE_URL")

	wP, err := strconv.Atoi(os.Getenv("WORKER_POOL"))
	if err != nil {
		panic(err)
	}

	c := Collector{FitQueueUrl: fQ, client: cli,
		maxMessages: 10, waitTime: 1, workerPoolSize: int32(wP)}

	return c
}

func newMessage(response types.Message) Message {
	// receive message in JSON format and return parsed message item
	body := response.Body
	m := Message{}
	json.Unmarshal([]byte(*body), &m)
	return m
}

func (c *Collector) delete(m Message) error {
	// remove message from sqs queue
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := c.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      &c.FitQueueUrl,
		ReceiptHandle: m.MessageHandle,
	})
	if err != nil {
		return fmt.Errorf("failed to delete message from sqs queue:%v", err)
	}
	return nil
}

func (c *Collector) Collect(workers []*Worker) {
	globalQuit := make(chan struct{})

	fmt.Printf("Polling SQS Queue...")

	var wg sync.WaitGroup

	//pool of listeners
	for w := 1; w <= int(c.workerPoolSize); w++ {
		wg.Add(1)
		go c.listener(workers, &wg, globalQuit)
	}

	wg.Wait()
	close(globalQuit)

	//aggregate results from monitor
	for _, w := range workers {
		if w.function == "monitor" {
			w.close()
			break
		}
	}

}

func (c *Collector) listener(workers []*Worker, wg *sync.WaitGroup, globalQuit chan struct{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for {
		result, err := c.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(c.FitQueueUrl),
			MaxNumberOfMessages: c.maxMessages,
			WaitTimeSeconds:     c.waitTime,
		})

		if err != nil {
			continue
		}

		var workersWG sync.WaitGroup

		for _, m := range result.Messages {
			//start different types of workers (in go routines)
			for _, w := range workers {
				workersWG.Add(1)
				w.StartWorker(w.handler, globalQuit, &workersWG)
			}

			// send messages to workers job channels
			msg := newMessage(m)
			dispatch(msg, workers)

			//fully process message and terminate
			workersWG.Wait()
			c.delete(msg)
		}

		wg.Done()
	}
}
