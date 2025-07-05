package collector

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	//TODO - mount IRSA to pod for AWS SDK creds
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type Collector struct {
	FitQueueUrl string
	migQueueUrl        string

	SqsClient   *sqs.Client
	maxMessages int32
	waitTime    int32
	//values for go routine
	workerPool  int32
	workerCount int32
	// TODO - Add migrator

}
type message struct {
	fitness int `json:"fitness"`
	hyperparameters map[string]interface{} `json:"hyperparameters"`
	source string `json:"source"`
}
// sqs message handler
func handler(m message,  messageChannel <- chan message) error {
	//read from registry of sub-populations and place individual there
	for m
}

func newMessage(response types.Message) message {
	// receive message in JSON format and return s
	body := response.Body
	m := message{}
	json.Unmarshal([]byte(*body),&m)
	return m
}

func (c *Collector) delete(m message) error {

}

func (c *Collector) Collect() {
	//create go routine for workers consuming messages from sqs
	messageChannel := make(chan message)
	for w := 1; w <= int(c.workerPool); w++ {
		go c.worker(w, messageChannel)
	}
}

func (c *Collector) worker(id int, messageChannel <- chan message) {
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
			// TODO - process m using newMessage(), add m to messageChannel, in the handler pull values from messageChannel to be handled by workers
			wg.Add(1)
			go func(m message, messageChannel <- chan message) {
				defer wg.Done()
				if err := handler(m, messageChannel); err != nil {
					log.Printf("Error handling request: %v", err)
					return
				}
				c.delete(m)
			}(newMessage(m))
		}
		wg.Wait()
	}
}
