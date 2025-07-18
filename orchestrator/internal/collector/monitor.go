package collector

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"log"
)

type Monitor struct {
	c              Collector
	handler        func(data interface{})
	worker         Worker
	dynamoDBClient *dynamodb.Client
	tableName      string
	df             dataframe.DataFrame
	trainCycle     int
	patience       int
	gauge          *prometheus.GaugeVec
}

type tableItem struct {
	trainCycle      int
	ID              int
	fitness         int
	bestFitness     int //TODO - different average?
	hyperparameters string
}

func min(a int, b int) int {
	if a < b {
		return a
	}

	return b
}
func (m *Monitor) newMonitor() {

	// create and assign gauge
	g := initGauge()
	m.gauge = g

	// serve prometheus connection
	servePromConn()

	// init monitor
	m.trainCycle = 0

	// init worker
	src := make(chan message)
	qt := make(chan struct{})
	w := Worker{source: src, quit: qt, handler: m.monHandler, function: "migrator"}

	m.worker = w

	//initialise client
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	client := dynamodb.NewFromConfig(cfg)
	m.dynamoDBClient = client

	df := dataframe.New(
		series.New(nil, series.Int, "trainCylce"),
		series.New(nil, series.Int, "island"),
		series.New(nil, series.Int, "fitness"),
		series.New(nil, series.Int, "bestFitness"),
		series.New(nil, series.String, "hyperparameters"),
	)
	m.df = df

}

func (m *Monitor) addRow(trainCycle int, id int, fit int, best int, hyper int) error {
	new := dataframe.New(
		series.New(trainCycle, series.Int, "trainCylce"),
		series.New(id, series.Int, "island"),
		series.New(fit, series.Int, "fitness"),
		series.New(best, series.Int, "bestFitness"),
		series.New(hyper, series.String, "hyperparameters"),
	)
	m.df = m.df.RBind(new)
	return nil
}

func (m *Monitor) monHandler(msg message) error {
	//get most recent fitness
	host := msg.hostname
	fil := m.df.Filter(
		dataframe.F{Colname: "island", Comparator: series.Eq, Comparando: host},
	)
	sorted := fil.Arrange(
		dataframe.RevSort("trainCylce"),
	)

	//get top 'patience' rows
	recent := sorted.Subset([]int{0, min(m.patience, sorted.Nrow())})

	fitnessCol := recent.Col("fitness")
	fitnessValues := fitnessCol.Records()

	var fitnessInts []int
	for _, val := range fitnessValues {
		i, err := strconv.Atoi(val)
		if err == nil {
			fitnessInts = append(fitnessInts, i)
		}
	}

	mostRecent := fitnessInts[0]
	hasImproved := false
	for _, v := range fitnessInts[1:] {
		if v > mostRecent {
			hasImproved = true
			break
		}
	}

	if !hasImproved {
		//stop training
		requestURL := fmt.Sprintf("http://%s.python-service.default.svc.cluster.local:5000/receive_mirgrant", msg.hostname)
		res, err := http.Get(requestURL)
		if err != nil {
			fmt.Printf("error making request: %v", err)
		}

		//send notification
		fmt.Printf("client response: status code: %d\n", res.StatusCode)
	}

	//send metrics to prometheus
	m.gauge.WithLabelValues(msg.hostname, "training").Set(float64(msg.fitness))

	m.trainCycle += 1
	return nil
}

func (m *Monitor) combineResults() error {
	if m.trainCycle > 1 {
		//get most recent fitness
		cycle := m.trainCycle
		fil := m.df.Filter(
			dataframe.F{Colname: "trainCycle", Comparator: series.Eq, Comparando: cycle},
		)
		newBest, err := fil.Elem(0, 3).Int()
		if err != nil {
			return fmt.Errorf("error getting previous best fitness: %v", err)
		}

		//get pervious fitness
		oldCycle := m.trainCycle - 1
		filOld := m.df.Filter(
			dataframe.F{Colname: "trainCycle", Comparator: series.Eq, Comparando: oldCycle},
		)
		oldBest, errOld := filOld.Elem(0, 3).Int()
		if errOld != nil {
			return fmt.Errorf("error getting previous best fitness: %v", errOld)
		}

		if newBest < oldBest {
			m.patience -= 1
			if m.patience < 0 {

				// halt all training
				fil = m.df.Filter(
					dataframe.F{Colname: "trainCycle", Comparator: series.Eq, Comparando: cycle},
				)

				islands := fil.Col("island")
				islandValues := islands.Records()

				var wg sync.WaitGroup
				for _, val := range islandValues {
					wg.Add(1)
					go stopTraining(val, &wg)
				}
			}
		}
	}
	return nil
}

func stopTraining(island string, wg *sync.WaitGroup) {
	defer wg.Done()
	url := fmt.Sprintf("http://%s.python-service.default.svc.cluster.local:5000/receive_mirgrant", island)
	res, err := http.Get(url)
	if err != nil {
		fmt.Printf("error making request: %v", err)
	}

	fmt.Printf("client response: status code: %d\n", res.StatusCode)

}
