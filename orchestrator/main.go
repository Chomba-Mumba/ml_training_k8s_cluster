package main

import (
	"fmt"
	"orchestrator/internal/collector"
	"orchestrator/internal/k8sclient"
)

//TODO - start orchestrator and start polling for results from the SQS queue.

func main() {
	k8s, err := k8sclient.NewK8sClient()
	if err != nil {
		fmt.Println(err)
	}

	//init migrator
	mig := collector.Migrator{}
	mig.NewMigrator()

	//init monitor
	mon := collector.Monitor{}
	mon.NewMonitor()

	//init collector and start collecting
	coll := collector.NewCollector()
	workers := []*collector.Worker{mig.Worker, mon.Worker}
	coll.Collect(workers)

	//start polling SQS queue

	fmt.Printf("main package")
}
