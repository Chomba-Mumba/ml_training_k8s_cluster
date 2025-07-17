package collector

import (
	"context"
	"fmt"

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
	cycle := m.trainCycle
	fil := m.df.Filter(
		dataframe.F{Colname: "trainCycle", Comparator: series.Eq, Comparando: cycle},
	)
	//if patience == 0 and acc hasnt improved halt training else reset patience
	best, err := fil.Elem(0, 3).Int()
	if err != nil {
		return fmt.Errorf("error getting previous best fitness: %v", err)
	}
	if msg.fitness < best {
		m.patience -= 1
		if m.patience < 0 {
			//halt training
			//TODO - halting training for a single sub population?
		}
	}

	//send metrics to prometheus
	m.gauge.WithLabelValues(msg.hostname, "training").Set(float64(msg.fitness))

	m.trainCycle += 1
	return nil
}
