package main

import (
	"context"
	"log"
	"orchestrator/internal/collector"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

//TODO - start orchestrator and start polling for results from the SQS queue.

func main() {
	// k8s, err := k8sclient.NewK8sClient()
	// if err != nil {
	// 	fmt.Println(err)
	// }

	//init migrator
	mig := collector.Migrator{}
	mig.NewMigrator()

	//init monitor
	mon := collector.Monitor{}
	mon.NewMonitor()

	//init collector and start polling queue
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
	cli := sqs.NewFromConfig(cfg)

	coll := collector.NewCollector(cli)

	workers := []*collector.Worker{mig.Worker, mon.Worker}
	coll.Collect(workers)

}
