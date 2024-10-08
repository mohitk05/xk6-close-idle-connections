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
		vu               modules.VU
		transportManager *TransportManager
	}
)

type TransportManager struct {
	vu      modules.VU
	started bool
	endFlag bool
}

// Ensure the interfaces are implemented correctly.
var (
	_ modules.Instance = &CloseIdleConn{}
	_ modules.Module   = &RootModule{}
)

// New returns a pointer to a new RootModule instance.
func New() *RootModule {
	return &RootModule{}
}

func (tm *TransportManager) Start(intervalSeconds int) {
	if tm.started {
		return
	}
	if intervalSeconds < 5 {
		tm.vu.State().Logger.Warn("intervalSeconds should be greater than 5 seconds, using default value 5 seconds")
		intervalSeconds = 5
	}

	transport := tm.vu.State().Transport.(*http.Transport)
	go func() {
		for {
			if tm.endFlag {
				tm.started = false
				break
			}
			time.Sleep(time.Duration(intervalSeconds) * time.Second)
			transport.CloseIdleConnections()
		}
	}()
	tm.started = true
	tm.endFlag = false
}

func (tm *TransportManager) End() {
	tm.endFlag = true
}

// NewModuleInstance implements the modules.Module interface returning a new instance for each VU.
func (*RootModule) NewModuleInstance(vu modules.VU) modules.Instance {
	return &CloseIdleConn{
		vu: vu,
		transportManager: &TransportManager{
			vu:      vu,
			started: false,
			endFlag: false,
		},
	}
}

func (mi *CloseIdleConn) Exports() modules.Exports {
	return modules.Exports{
		Default: mi.transportManager,
	}
}
