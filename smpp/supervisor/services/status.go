package services

import (
	"fmt"
	"os/exec"
	"strings"

	log "github.com/Sirupsen/logrus"
)

// Status reperesents status of single process executed by supervisord
type Status struct {
	Program string
	Status  string
	Ok      bool
}

// GetStatus returns status of running supervisor processes
func GetStatus() ([]Status, error) {
	var st []Status

	b, err := exec.Command("supervisorctl", "status").CombinedOutput()
	if err != nil {
		log.WithError(err).Error("Couldn't execute supervisor ctl command.")
		return st, fmt.Errorf("Couldn't execute supervisor ctl command.")
	}
	lines := strings.Split(string(b), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) < 5 {
			log.WithFields(log.Fields{
				"line":   line,
				"fields": fields,
				"lines":  lines,
			}).Error("Not enough fields.")
			return st, fmt.Errorf("Supervisorctl didn't return correct number of status lines in line %s", line)
		}
		st = append(st, Status{
			Program: fields[0],
			Status:  fields[1],
			Ok:      fields[1] == "RUNNING",
		})
	}

	return st, nil
}
