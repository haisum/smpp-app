package main

import (
	"bitbucket.com/codefreak/hsmpp/smpp"
	log "github.com/Sirupsen/logrus"
	"os"
	"os/exec"
)

var (
	c smpp.Config
)

func supervisorctl(args []string) {
	out, err := exec.Command("supervisorctl", args...).Output()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("\n%s", out)
}

func tpl() {

}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: \n%s [status|start (all|httpserver|smppworker)|stop (all|httpserver|smppworker)]", os.Args[0])
	}
	err := c.LoadFile("settings.json")
	if err != nil {
		log.WithField("err", err).Fatal("Could not read settings.")
	}
	tpl()
	supervisorctl(os.Args[1:])
}
