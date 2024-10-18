package close_idle_conn

import (
	"net/http"
	"time"

	"go.k6.io/k6/js/modules"
)

func init() {
	modules.Register("k6/x/close_idle_conn", new(CloseIdleConn))
}

type (
	// RootModule is the global module instance that will create module
	// instances for each VU.
	RootModule struct{}

	// ModuleInstance represents an instance of the JS module.
	CloseIdleConn struct {
		// vu provides methods for accessing internal k6 objects for a VU
		vu          modules.VU
		started     bool
		channelDone chan bool
	}
)

// Ensure the interfaces are implemented correctly.
var (
	_ modules.Instance = &CloseIdleConn{}
	_ modules.Module   = &RootModule{}
)

// New returns a pointer to a new RootModule instance.
func New() *RootModule {
	return &RootModule{}
}

func (ci *CloseIdleConn) Start(intervalSeconds int) {
	if ci.started {
		return
	}

	if intervalSeconds < 5 {
		ci.vu.State().Logger.Warn("intervalSeconds should be greater than 5 seconds, using default value 5 seconds")
		intervalSeconds = 5
	}

	transport := ci.vu.State().Transport.(*http.Transport)
	go func() {
		ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
		ci.started = true
		for {
			select {
			case <-ticker.C:
				transport.CloseIdleConnections()
			case <-ci.channelDone:
				ticker.Stop()
				ci.started = false
				return
			}
		}
	}()
}

func (ci *CloseIdleConn) End() {
	ci.channelDone <- true
}

// NewModuleInstance implements the modules.Module interface returning a new instance for each VU.
func (*RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	return &CloseIdleConn{
		vu:          vu,
		started:     false,
		channelDone: make(chan bool),
	}
}

func (mi *CloseIdleConn) Exports() modules.Exports {
	return modules.Exports{
		Named: map[string]interface{}{
			"start": mi.Start,
			"end":   mi.End,
		},
	}
}
