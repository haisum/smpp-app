package process

import "context"

// Manager is process manager's interface
type Manager interface {
	Start(p ...[]Process) error
	Stop(name ...[]string) error
	Status() []Status
}

// Process is the process managed my process manager
type Process interface {
	Name() string
	Start(ctx context.Context) error
	Stop() error
	Status() (Status, error)
}

// State is one of WAITING|STARTED|STOPPED
type State string

const (
	// WAITING state is when a process is waiting for db connection or some other resource to be available
	WAITING State = "WAITING"
	// STARTED state is successful state where everything is running as expected
	STARTED State = "STARTED"
)

// Status is process status
type Status struct {
	ProcessName string
	State       State
	Message     string
}
