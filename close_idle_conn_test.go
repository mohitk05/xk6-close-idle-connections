package close_idle_conn

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.k6.io/k6/js/modules/k6"
	k6Http "go.k6.io/k6/js/modules/k6/http"
	"go.k6.io/k6/js/modulestest"
	"go.k6.io/k6/lib"
	"go.k6.io/k6/lib/testutils/httpmultibin"
	"go.k6.io/k6/metrics"
	"gopkg.in/guregu/null.v3"
)

type ConnectionWatcher struct {
	newConnections int64
}

func (cw *ConnectionWatcher) OnStateChange(conn net.Conn, state http.ConnState) {
	switch state {
	case http.StateNew:
		cw.Add(1)
	}
}

func (cw *ConnectionWatcher) Add(c int64) {
	atomic.AddInt64(&cw.newConnections, c)
}

func (cw *ConnectionWatcher) NewConnections() int64 {
	return atomic.LoadInt64(&cw.newConnections)
}

func (cw *ConnectionWatcher) Reset() {
	atomic.StoreInt64(&cw.newConnections, 0)
}

func TestCloseIdleConnStart(t *testing.T) {
	t.Parallel()

	fmt.Println("TestCloseIdleConnStart")

	testCases := []struct {
		Name                string
		WaitTime            int
		EnableCloseIdleConn bool
		NoConnectionReuse   bool
		Script              string
		Tester              func(t *testing.T, newConnections int64)
	}{
		{
			Name:                "No close_idle_conn, 4 requests",
			WaitTime:            0,
			EnableCloseIdleConn: false,
			NoConnectionReuse:   false,
			Script:              `for (let i = 0; i < 4; i++) { http.get("HTTPBIN_URL/get") }`,
			Tester: func(t *testing.T, newConnections int64) {
				require.Equal(t, int64(1), newConnections)
			},
		},
		{
			Name:                "No close_idle_conn, 4 requests, with K6_NO_CONNECTION_REUSE",
			WaitTime:            0,
			EnableCloseIdleConn: false,
			NoConnectionReuse:   true,
			Script:              `for (let i = 0; i < 4; i++) { http.get("HTTPBIN_URL/get") }`,
			Tester: func(t *testing.T, newConnections int64) {
				require.Equal(t, int64(4), newConnections)
			},
		},
		{
			Name:                "With close_idle_conn.start(5), 4 requests, wait 1 second each",
			WaitTime:            0,
			EnableCloseIdleConn: true,
			NoConnectionReuse:   false,
			Script:              `for (let i = 0; i < 4; i++) { http.get("HTTPBIN_URL/get"); sleep(1); }`,
			Tester: func(t *testing.T, newConnections int64) {
				require.Equal(t, int64(1), newConnections)
			},
		},
		{
			Name:                "With close_idle_conn.start(5), 4 requests, wait 2 seconds each",
			WaitTime:            0,
			EnableCloseIdleConn: true,
			NoConnectionReuse:   false,
			Script:              `for (let i = 0; i < 4; i++) { http.get("HTTPBIN_URL/get"); sleep(2); }`,
			Tester: func(t *testing.T, newConnections int64) {
				require.Equal(t, int64(2), newConnections)
			},
		},
	}

	for _, tc := range testCases {
		println("Running test case:", tc.Name)
		tb := httpmultibin.NewHTTPMultiBin(t)
		defer tb.ServerHTTP.Close()
		tb.Mux.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		var cw ConnectionWatcher
		tb.ServerHTTP.Config.ConnState = cw.OnStateChange

		testRuntime, samples := newTestRuntime(t, tb, tc.NoConnectionReuse)

		fmt.Println("No connection reuse:", testRuntime.VU.State().Options.NoConnectionReuse)

		closeIdleConn := New().NewModuleInstance(testRuntime.VU).(*CloseIdleConn)
		if tc.EnableCloseIdleConn {
			closeIdleConn.Start(5)
		}

		_, err := testRuntime.VU.Runtime().RunString(tb.Replacer.Replace(tc.Script))
		require.NoError(t, err)

		bufSamples := metrics.GetBufferedSamples(samples)

		for _, container := range bufSamples {
			for _, sample := range container.GetSamples() {
				if sample.Metric.Name == "http_req_connecting" {
					fmt.Println("http_req_connecting:", sample.Value)
				}
			}
		}

		time.Sleep(time.Duration(tc.WaitTime) * time.Second)

		newConnections := cw.NewConnections()
		tc.Tester(t, newConnections)
		fmt.Println("Number of new connections:", newConnections)

		cw.Reset()
	}

}

func newTestRuntime(t *testing.T, tb *httpmultibin.HTTPMultiBin, noConnectionReuse bool) (*modulestest.Runtime, chan metrics.SampleContainer) {
	t.Helper()

	testRuntime := modulestest.NewRuntime(t)
	registry := metrics.NewRegistry()

	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.Out = io.Discard

	samples := make(chan metrics.SampleContainer, 1000)

	state := &lib.State{
		Options: lib.Options{
			SystemTags:        &metrics.DefaultSystemTagSet,
			UserAgent:         null.StringFrom("k6-test"),
			MaxRedirects:      null.IntFrom(10),
			Throw:             null.BoolFrom(true),
			Batch:             null.IntFrom(20),
			BatchPerHost:      null.IntFrom(20),
			NoConnectionReuse: null.BoolFrom(noConnectionReuse),
		},
		BuiltinMetrics: metrics.RegisterBuiltinMetrics(registry),
		Tags:           lib.NewVUStateTags(registry.RootTagSet()),
		Logger:         logger,
		Transport:      tb.HTTPTransport,
		BufferPool:     lib.NewBufferPool(),
		TLSConfig:      tb.TLSClientConfig,
		Samples:        samples,
	}

	testRuntime.MoveToVUContext(state)

	k6HttpModule := k6Http.New().NewModuleInstance(testRuntime.VU)
	testRuntime.VU.Runtime().Set("http", k6HttpModule.Exports().Default)

	k6Module := k6.New().NewModuleInstance(testRuntime.VU)
	testRuntime.VU.Runtime().Set("sleep", k6Module.Exports().Named["sleep"])

	return testRuntime, samples
}
