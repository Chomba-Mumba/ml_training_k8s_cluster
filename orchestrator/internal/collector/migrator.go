package collector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
)

type Migrator struct {
	Worker       *Worker
	TotalIslands int
	HTTPClient   HTTPClientInterface
}

func (m *Migrator) NewMigrator(client HTTPClientInterface) {

	src := make(chan Message)
	qt := make(chan struct{})
	w := Worker{source: src, quit: qt, handler: m.MigHandler, function: "migrator"}

	tI, err := strconv.Atoi(os.Getenv("TOTAL_ISLANDS"))
	if err != nil {
		panic(err)
	}
	m.Worker = &w
	m.TotalIslands = int(tI)
	m.HTTPClient = client

}

func (m *Migrator) findRecepient(src string) (string, error) {
	s, err := strconv.Atoi(string(src[len(src)-1:]))
	if err != nil {
		return "", fmt.Errorf("error finding recepient: %v", err)
	}
	return strconv.Itoa((s + 1) % m.TotalIslands), nil
}

func (m *Migrator) MigHandler(msg Message) error {
	//send indiividual to relevant island
	des, err := m.findRecepient(msg.Hostname)
	if err != nil {
		panic(err)
	}

	//internal cluster endpoint
	posturl := fmt.Sprintf("http://%s.python-service.default.svc.cluster.local:5000/migrant", des)

	h, err := json.Marshal(msg.Hyperparameters)
	if err != nil {
		panic(err)
	}

	body := []byte(fmt.Sprintf(`{
		"fitness": %d,
		"hyperparameters": %s
		}`, msg.Fitness, h))

	res, err := m.HTTPClient.Post(posturl, "application/json", bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}

	if res.StatusCode != http.StatusOK {
		// panic(res.Status)
		return nil
	}

	return nil
}
