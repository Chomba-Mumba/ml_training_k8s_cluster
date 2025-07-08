package collector

type Monitor struct {
	c       Collector
	handler func(data interface{})
	worker  Worker
}

func (m *Monitor) newMonitor() {

	w := Worker{handler: m.mon_handler, function: "migrator"}

	m.worker = w
}

func (m *Monitor) mon_handler(msg message) error {
	return nil
}
