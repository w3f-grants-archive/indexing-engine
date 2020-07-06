package prometheusmetrics

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/figment-networks/indexing-engine/metrics"
	"github.com/stretchr/testify/require"
)

var httptestSrv *httptest.Server

func TestMain(m *testing.M) {

	metric := New()
	err := metrics.DetaultMetrics.AddEngine(metric)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", metric.Handler())
	httptestSrv = httptest.NewServer(mux)
	defer httptestSrv.Close()

	st := m.Run()
	os.Exit(st)

}
func TestMetrics_NewCounterWithTags(t *testing.T) {

	t.Run("New counter with tags", func(t *testing.T) {

		got, err := metrics.NewCounterWithTags(metrics.Options{
			Namespace: "a",
			Subsystem: "b",
			Name:      "c",
			Desc:      "d",
			Tags:      []string{"e", "f", "g"},
		})
		require.NoError(t, err)

		counter := got.WithLabels([]string{"e1", "f1", "g1"})
		counter.Inc()
		counter.Inc()
		counter.Inc()

		res, err := http.Get(httptestSrv.URL + "/metrics")
		data, err := ioutil.ReadAll(res.Body)
		res.Body.Close()

		require.Contains(t, string(data), `a_b_c{e="e1",f="f1",g="g1"} 3`)

	})

	t.Run("New gauge with tags", func(t *testing.T) {

		got, err := metrics.NewGaugeWithTags(metrics.Options{
			Namespace: "a",
			Subsystem: "b",
			Name:      "c1",
			Desc:      "d",
			Tags:      []string{"e", "g"},
		})
		require.NoError(t, err)

		gauge := got.WithLabels([]string{"e1", "g1"})
		gauge.Inc()
		gauge.Dec()
		gauge.Inc()

		res, err := http.Get(httptestSrv.URL + "/metrics")
		data, err := ioutil.ReadAll(res.Body)
		res.Body.Close()

		require.Contains(t, string(data), `a_b_c1{e="e1",g="g1"} 1`)
	})

	t.Run("New observer with tags", func(t *testing.T) {

		got, err := metrics.NewHistogramWithTags(metrics.HistogramOptions{
			Namespace: "a",
			Subsystem: "b",
			Name:      "c2",
			Desc:      "d",
			Tags:      []string{"e", "f", "g"},
		})
		require.NoError(t, err)

		hist := got.WithLabels([]string{"e1", "f1", "g1"})
		hist.Observe(123.456)

		res, err := http.Get(httptestSrv.URL + "/metrics")
		data, err := ioutil.ReadAll(res.Body)
		res.Body.Close()

		require.Contains(t, string(data), `a_b_c2_sum{e="e1",f="f1",g="g1"} 123.456`)

	})
}
