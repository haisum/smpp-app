package main

import (
	"bitbucket.com/codefreak/hsmpp/smpp"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"text/template"
)

type Group struct {
	Name     string
	Programs []Program
}

type Program struct {
	Name    string
	Command string
}

type TplData struct {
	Groups []Group
}

func (t *TplData) load(c smpp.Config) {
	path, err := os.Getwd()
	if err != nil {
		log.WithField("err", err).Fatal("Couldn't determine path of app. Weird, very weird.")
	}

	workers := make([]Program, 0)
	for _, w := range c.Conns {
		p := Program{
			Name:    fmt.Sprintf("smppworker-%s", w.Id),
			Command: fmt.Sprintf("%s/./smppworker -cid='%s'", path, w.Id),
		}
		workers = append(workers, p)
	}

	httpservers := []Program{{
		Name:    "httpserver",
		Command: fmt.Sprintf("%s/./httpserver", path),
	}}
	t.Groups = []Group{
		{
			Name:     "workers",
			Programs: workers,
		},
		{
			Name:     "httpservers",
			Programs: httpservers,
		},
	}
}

func supervisorctl(args []string) {
	out, err := exec.Command("supervisorctl", args...).Output()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("\n%s", out)
}

func tpl() {
	bf, err := ioutil.ReadFile("supervisord.conf.template")
	if err != nil {
		log.WithField("err", err).Fatal("Couldn't read template file supervisord.conf.template")
	}
	t := template.New("supervisord.conf.template")
	_, err = t.Parse(string(bf[:]))
	if err != nil {
		log.WithField("err", err).Fatal("Couldn't parse template.")
	}
	conffile, err := os.Create("supervisord.conf")
	if err != nil {
		log.WithField("err", err).Fatal("Couldn't create file supervisord.conf.")
	}

	var c smpp.Config
	err = c.LoadFile("settings.json")
	if err != nil {
		log.WithField("err", err).Fatal("Could not read settings.")
	}
	var td TplData
	td.load(c)
	err = t.Execute(conffile, td)
	if err != nil {
		log.WithField("err", err).Fatal("Couldn't execute template on supervisord.conf.")
	}
	err = conffile.Close()
	if err != nil {
		log.WithField("err", err).Fatal("Couldn't close connection to file.")
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: \n%s [status|start (all|httpserver|smppworker)|restart (all|httpserver|smppworker)|stop (all|httpserver|smppworker)]", os.Args[0])
	}
	tpl()
	supervisorctl(os.Args[1:])
}
