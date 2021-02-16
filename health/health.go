package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type Readiness struct {
	DB map[string]interface{} `json:"db"`
}

type Prober interface {
	// Probe to run all the necessary checks on
	Probe() error
	// Readiness should return information about the readiness of resource
	Readiness() (probetype, readinesstype string, contents interface{}, err error)
}

type Monitor struct {
	sync.RWMutex
	probers []Prober
}

func (m *Monitor) AddProber(ctx context.Context, p Prober) {
	m.Lock()
	m.probers = append(m.probers, p)
	m.Unlock()
}

func (m *Monitor) RunChecks(ctx context.Context, dur time.Duration) {
	tckr := time.NewTicker(dur)

	for {
		select {
		case <-ctx.Done():
			return
		case <-tckr.C:
			m.RLock()
			for _, p := range m.probers {
				p.Probe()
			}
			m.RUnlock()
		}
	}
}

func (m *Monitor) AttachHttp(ctx context.Context, mux *http.ServeMux) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/readiness", func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)

		var fErr error
		rSt := Readiness{}
		m.RLock()
		for _, p := range m.probers {
			ty, readinesstype, co, err := p.Readiness()
			if err != nil {
				fErr = err
			}
			switch ty {
			case "db":
				rSt.DB[readinesstype] = co
			}
		}
		m.RUnlock()

		if fErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		enc.Encode(r)
	})
}
