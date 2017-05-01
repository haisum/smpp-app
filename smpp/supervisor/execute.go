package supervisor

import (
	"bitbucket.org/codefreak/hsmpp/smpp"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
	"text/template"
)

// Group represents group of related processes
type Group struct {
	Name     string
	Programs []Program
}

// Program represents program's name and command to start
type Program struct {
	Name    string
	Command string
}

// TplData represents data passed to supervisord.conf template
type TplData struct {
	Groups []Group
	User   string
}

func (t *TplData) load(c smpp.Config) {

	var workers []Program
	for _, g := range c.ConnGroups {
		for _, w := range g.Conns {
			p := Program{
				Name:    fmt.Sprintf("smppworker-%s-%s", g.Name, w.ID),
				Command: fmt.Sprintf("./smppworker -cid='%s' -group='%s'", w.ID, g.Name),
			}
			workers = append(workers, p)
		}
	}
	t.Groups = []Group{
		{
			Name:     "workers",
			Programs: workers,
		},
		{
			Name: "Scheduler",
			Programs: []Program{
				{
					Name:    "scheduler",
					Command: fmt.Sprintf("./scheduler"),
				},
			},
		},
		{
			Name: "Soap",
			Programs: []Program{
				{
					Name:    "soapservice",
					Command: fmt.Sprintf("./soapservice"),
				},
			},
		},
	}
	currentuser, err := user.Current()
	if err != nil {
		log.Fatal("Couldn't get current user!")
	}
	if runtime.GOOS == "windows" {
		t.User = strings.Split(currentuser.Username, "\\")[1]
	} else {
		t.User = currentuser.Username
	}
}

func supervisorctl(args []string) ([]byte, error) {
	out, err := exec.Command("supervisorctl", args...).CombinedOutput()
	if err != nil {
		exec.Command("supervisord")
		out, err = exec.Command("supervisorctl", args...).CombinedOutput()
	}
	return out, err
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

	c, err := smpp.GetConfig()
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

func Execute(command string) ([]string, error) {
	var o []string
	tpl()
	var args []string
	if command == "reload" {
		args = []string{"reload"}
	} else {
		args = []string{fmt.Sprintf("%s all", command)}
	}
	exec.Command("supervisord").Output()
	out, err := supervisorctl(args)
	if err != nil {
		return o, err
	}
	s := string(out)
	log.WithField("out", s).Info("Executed ctl command.")
	o = strings.Split(s, "\n")
	return o, nil
}
