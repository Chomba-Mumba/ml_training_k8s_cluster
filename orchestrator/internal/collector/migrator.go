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
	totalIslands int
	handler      func(msg message) error
}

func (m *Migrator) NewMigrator() {

	src := make(chan message)
	qt := make(chan struct{})
	w := Worker{source: src, quit: qt, handler: m.mig_handler, function: "migrator"}

	tI, err := strconv.Atoi(os.Getenv("TOTAL_ISLANDS"))
	if err != nil {
		panic(err)
	}
	m.Worker = &w
	m.totalIslands = int(tI)

}

func (m *Migrator) findRecepient(src string) (string, error) {
	s, err := strconv.Atoi(string(src[len(src)-1:]))
	if err != nil {
		return "", fmt.Errorf("error finding recepient: %v", err)
	}
	return strconv.Itoa((s + 1) % m.totalIslands), nil
}

func (m *Migrator) mig_handler(msg message) error {
	//send indiividual to relevant island
	des, err := m.findRecepient(msg.hostname)
	if err != nil {
		return fmt.Errorf("error in handler: %v", err)
	}

	//internal cluster endpoint
	posturl := fmt.Sprintf("http://%s.python-service.default.svc.cluster.local:5000/receive_mirgrant", des)

	h, err := json.Marshal(msg.hyperparameters)
	if err != nil {
		return fmt.Errorf("error in handler: %v", err)
	}

	body := []byte(fmt.Sprintf(`{
		"fitness": %d,
		"body": %s
		}`, msg.fitness, h))

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
