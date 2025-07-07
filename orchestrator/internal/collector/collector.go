package collector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

	SqsClient    *sqs.Client
	maxMessages  int32
	waitTime     int32
	totalIslands int
	workerPool   int32
}
type message struct {
	fitness         int                    `json:"fitness"`
	hyperparameters map[string]interface{} `json:"hyperparameters"`
	hostname        string                 `json:"hostname"`
	messageHandle   *string
}

func NewCollector() Collector {
	fQ := os.Getenv("FIT_QUEUE_URL")

	tI, err := strconv.Atoi(os.Getenv("TOTAL_ISLANDS"))
	if err != nil {
		panic(err)
	}

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
		maxMessages: 10, waitTime: 1, totalIslands: int(tI), workerPool: int32(wP)}

	return c
}
func (c *Collector) findRecepient(src string) (string, error) {
	s, err := strconv.Atoi(string(src[len(src)-1:]))
	if err != nil {
		return "", fmt.Errorf("Error finding recepient: %v", err)
	}
	return strconv.Itoa((s + 1) % c.totalIslands), nil
}

// sqs message handler
func (c *Collector) handler(m message) error {
	//send island to relevant island

	des, err := c.findRecepient(m.hostname)
	if err != nil {
		return fmt.Errorf("Error in handler: %v", err)
	}

	//internal cluster endpoint
	posturl := fmt.Sprintf("http://%s.python-service.default.svc.cluster.local:5000/receive_mirgrant", des)

	h, err := json.Marshal(m.hyperparameters)
	if err != nil {
		return fmt.Errorf("Error in handler: %v", err)
	}

	body := []byte(fmt.Sprintf(`{
		"fitness": %d,
		"body": %s
		}`, m.fitness, h))

	r, err := http.NewRequest("POST", posturl, bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}

	r.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		panic(err)
	}

	if res.StatusCode != http.StatusOK {
		panic(res.Status)
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
