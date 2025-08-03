package collector

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type Monitor struct {
	Worker         *Worker
	dynamoDBClient *dynamodb.Client
	df             dataframe.DataFrame
	trainCycle     int
	patience       int
	gauge          *prometheus.GaugeVec
	HTTPClient     HTTPClientInterface
}

type tableItem struct {
	trainCycle      int
	ID              int
	fitness         int //TODO - change to float value
	bestFitness     int //TODO - different average?
	hyperparameters string
}

type DynamoClient interface {
}

func min(a int, b int) int {
	if a < b {
		return a
	}

	return b
}

func (m *Monitor) NewMonitor(HTTPClient HTTPClientInterface) {

	// create and assign gauge
	g := initGauge()
	m.gauge = g

	// serve prometheus connection
	servePromConn()

	// init monitor
	m.trainCycle = 0

	m.HTTPClient = HTTPClient

	// init worker
	src := make(chan Message)
	qt := make(chan struct{})
	w := Worker{source: src, quit: qt, handler: m.MonHandler, function: "monitor", close: m.combineResults}

	m.Worker = &w

	df := dataframe.New(
		series.New(nil, series.Int, "trainCylce"),
		series.New(nil, series.Int, "island"),
		series.New(nil, series.Int, "fitness"),
		series.New(nil, series.Int, "bestFitness"),
		series.New(nil, series.String, "hyperparameters"),
	)
	m.df = df

}

func (m *Monitor) addRow(trainCycle int, id int, fit int, best int, hyper map[string]interface{}) {
	new := dataframe.New(
		series.New(trainCycle, series.Int, "trainCylce"),
		series.New(id, series.Int, "island"),
		series.New(fit, series.Int, "fitness"),
		series.New(best, series.Int, "bestFitness"),
		series.New(hyper, series.String, "hyperparameters"),
	)
	m.df = m.df.RBind(new)
}

func (m *Monitor) MonHandler(msg Message) error {

	host := msg.Hostname

	if m.trainCycle < m.patience {
		//get most recent fitness values

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

		// check if fitness has improved for 'patience' rows
		mostRecent := fitnessInts[0]
		hasImproved := false
		for _, v := range fitnessInts[1:] {
			if v > mostRecent {
				hasImproved = true
				break
			}
		}

		if !hasImproved {
			//stop training for host
			requestURL := fmt.Sprintf("http://%s.python-service.default.svc.cluster.local:5000/stop_training", msg.Hostname)

			res, err := m.HTTPClient.Get(requestURL)
			if err != nil {
				return fmt.Errorf("error making request: %v", err)
			}

			if res.StatusCode != http.StatusOK {
				return fmt.Errorf("client response: status code: %d", res.StatusCode)
			}

			//send notification
			return nil
		}
	}

	//send metrics to prometheus
	m.gauge.WithLabelValues(msg.Hostname, "training").Set(float64(msg.Fitness))

	m.trainCycle += 1

	hostID, err := strconv.Atoi(host)
	if err != nil {
		return fmt.Errorf("error getting hostID %v", err)
	}

	//add details to df
	m.addRow(m.trainCycle, hostID, msg.Fitness, 0, msg.Hyperparameters)
	return nil
}

func (m *Monitor) combineResults() error {
	//check last num islands rows populated

	if m.trainCycle < 1 {
		return nil
	}

	cycle := m.trainCycle
	//get most recent fitness
	fil := m.df.Filter(
		dataframe.F{Colname: "trainCycle", Comparator: series.Eq, Comparando: cycle},
	)
	newBest, err := fil.Elem(0, 3).Int()
	if err != nil {
		return fmt.Errorf("error getting best fitness: %v", err)
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
	//send message to quit channel

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
