package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	FitQueueUrl string
	migQueueUrl string

	SqsClient    *sqs.Client
	maxMessages  int32
	waitTime     int32
	totalIslands int
	workerPool   int32
}
type message struct {
	fitness         int                    `json:"fitness"`
	hyperparameters map[string]interface{} `json:"hyperparameters"`
	source          int                    `json:"source"`
	messageHandle   *string
}

// sqs message handler
func (c *Collector) handler(m message) error {
	//read from registry of sub-populations and place individual there
	h, err := json.Marshal(m.hyperparameters)
	if err != nil {
		return fmt.Errorf("unable to create a client: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = c.SqsClient.SendMessage(ctx, &sqs.SendMessageInput{
		MessageAttributes: map[string]types.MessageAttributeValue{
			"Destination": {
				DataType:    aws.String("String"),
				StringValue: aws.String(strconv.Itoa((m.source + 1) % c.totalIslands)),
			},
			"Source": {
				DataType:    aws.String("String"),
				StringValue: aws.String(strconv.Itoa(m.source)),
			},
			"Fitness": {
				DataType:    aws.String("String"),
				StringValue: aws.String(strconv.Itoa(m.fitness)),
			},
		},
		MessageBody: aws.String(string(h)),
		QueueUrl:    &c.migQueueUrl,
	})
	if err != nil {
		return fmt.Errorf("failed to send message to sqs: %v", err)
	}
	return nil
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
		for _, m := range result.Messages {
			wg.Add(1)
			go func(m message) {
				defer wg.Done()
				if err := c.handler(m); err != nil {
					log.Printf("Error handling request: %v", err)
					return
				}
				c.delete(m)
			}(newMessage(m))
		}
		wg.Wait()
	}
}
