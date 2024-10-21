package close_idle_conn

import (
	"fmt"
	"net/http"
	"time"

	"go.k6.io/k6/js/modules"
)

func init() {
	modules.Register("k6/x/close_idle_conn", New())
}

type (
	// RootModule is the global module instance that will create module
	// instances for each VU.
	RootModule struct{}

	// ModuleInstance represents an instance of the JS module.
	ModuleInstance struct {
		// vu provides methods for accessing internal k6 objects for a VU
		vu            modules.VU
		closeIdleConn *CloseIdleConn
	}
)

// Ensure the interfaces are implemented correctly.
var (
	_ modules.Instance = &ModuleInstance{}
	_ modules.Module   = &RootModule{}
)

// New returns a pointer to a new RootModule instance.
func New() *RootModule {
	return &RootModule{}
}

type CloseIdleConn struct {
	vu          modules.VU
	started     bool
	channelDone chan bool
}

func (cic *CloseIdleConn) Start(intervalSeconds int) {
	if cic.started {
		return
	}

	state := cic.vu.State()
	if state == nil {
		fmt.Println("k6/x/close_idle_conn: state is nil, cannot start close_idle_conn")
		return
	}

	if intervalSeconds < 5 {
		state.Logger.Warn("intervalSeconds should be greater than 5 seconds, using default value 5 seconds")
		intervalSeconds = 5
	}

	transport := state.Transport.(*http.Transport)
	go func() {
		ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
		defer ticker.Stop()
		cic.started = true
		for {
			select {
			case <-ticker.C:
				state.Logger.Debugln("k6/x/close_idle_conn: closing idle connections")
				transport.CloseIdleConnections()
			case <-cic.channelDone:
				state.Logger.Debugln("k6/x/close_idle_conn: received message to stop")
				cic.started = false
				return
			}
		}
	}()
}

func (cic *CloseIdleConn) End() {
	if !cic.started {
		return
	}
	cic.channelDone <- true
}

// NewModuleInstance implements the modules.Module interface returning a new instance for each VU.
func (*RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	return &ModuleInstance{
		vu: vu,
		closeIdleConn: &CloseIdleConn{
			vu:          vu,
			started:     false,
			channelDone: make(chan bool),
		},
	}
}

func (mi *ModuleInstance) Exports() modules.Exports {
	return modules.Exports{
		Named: map[string]interface{}{
			"start": mi.closeIdleConn.Start,
			"end":   mi.closeIdleConn.End,
		},
	}
}
