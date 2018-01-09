package worker

import "bitbucket.org/codefreak/hsmpp/smpp/process"

// manager is worker manager for managing worker processes
type manager struct {
}

func (m *manager) Start(p ...[]process.Process) error {
	return nil
}
func (m *manager) Stop(name ...[]string) error {
	return nil
}
func (m *manager) Status() []process.Status {
	return nil
}

var m *manager

// GetManager returns worker manager
func GetManager() process.Manager {
	return m
}
