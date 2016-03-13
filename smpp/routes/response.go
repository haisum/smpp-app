package routes

// Response represents json/xml response we give to requests
type Response struct {
	Obj     interface{} `xml:"Obj" json:"Response"`
	Errors  []string
	Ok      bool
	Request interface{}
}
