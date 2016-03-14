package main

import (
	hsmpp "bitbucket.com/codefreak/hsmpp/smpp"
	"bitbucket.com/codefreak/hsmpp/smpp/soap"
	"encoding/xml"
	"flag"
	"fmt"
	smpp "github.com/CodeMonkeyKevin/smpp34"
	log "github.com/Sirupsen/logrus"
	"net/http"
	"os"
)

const (
	HTTPPort int = 80
)

var (
	host     = flag.String("host", "localhost", "SMPP host address.")
	port     = flag.Int("port", 2775, "SMPP host port.")
	username = flag.String("username", "", "Username to connect to smpp server.")
	password = flag.String("password", "", "Password to connect to smpp server.")
)

func main() {
	optionalFields := []string{"source_addr_ton", "source_addr_npi", "dest_addr_ton", "dest_addr_npi"}
	optionalFlags := make(map[string]*int)
	for _, v := range optionalFields {
		optionalFlags[v] = flag.Int(v, 0, fmt.Sprintf("Optional %s field", v))
	}

	flag.Parse()
	if *username == "" || *password == "" {
		flag.Usage()
		os.Exit(1)
	}
	// connect and bind
	s := hsmpp.Sender{}
	s.Connect(*host, *port, *username, *password)
	defer s.Close()

	params := smpp.Params{}
	for _, v := range optionalFields {
		if *optionalFlags[v] != 0 {
			params[v] = *optionalFlags[v]
		}
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		decoder := xml.NewDecoder(r.Body)
		var e soap.SOAPEnvelope
		err := decoder.Decode(&e)
		if err != nil {
			http.Error(w, "Couldn't understand soap request.", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		err = s.Send(e.Body.Request.Src, e.Body.Request.Dst, e.Body.Request.Message, e.Body.Request.Coding == 2, params)
		if err != nil {
			fmt.Fprintf(w, soap.SOAPResponse, "-1")
		} else {
			fmt.Fprintf(w, soap.SOAPResponse, "OK")
		}
		return
	})
	http.HandleFunc("/wsdl", func(w http.ResponseWriter, r *http.Request) {
		host := r.FormValue("host")
		if host == "" {
			host = "localhost"
		}
		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprintf(w, soap.WSDL, host)
		return
	})
	go s.ReadPDUs()
	log.Infof("Listening on port %s.", HTTPPort)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", HTTPPort), nil))
}
