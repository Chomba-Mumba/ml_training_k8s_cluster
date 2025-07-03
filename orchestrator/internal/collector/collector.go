package collector

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	//TODO - mount IRSA to pod for AWS SDK creds
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type Collector struct {
	FitQueueUrl string
	migQ        string
	fitVals     []string
	SqsClient   *sqs.Client
	maxMessages int32
	waitTime    int32
	//values for go routine
	workerPool  int32
	workerCount int32
	// TODO - Add migrator

}

// sqs message handler
func handler(msg types.Message) error {
	//
}

func newMessage(msg types.Message) types.Message {

}

func (c *Collector) delete(m types.Message) error {

}

func (c *Collector) run(m types.Message) error {
	defer atomic.AddInt32(&c.workerCount, -1)
	err := handler(m)
	if err != nil {
		return err
	}
	return c.delete(m)
}

func (c *Collector) Collect() {
	//create go routine for workers consuming messages from sqs
	for w := 1; w <= int(c.workerPool); w++ {
		go c.worker(w)
	}
}

func (c *Collector) worker(id int) {
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
		for _, message := range result.Messages {
			wg.Add(1)
			go func(message types.Message) {
				defer wg.Done()
				if err := handler(message); err != nil {
					log.Printf("Error handling request: %v", err)
					return
				}
				c.delete(message)
			}(newMessage(message))
		}
		wg.Wait()
	}
}
