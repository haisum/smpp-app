package queue

import "testing"

func TestQueueItem_FromJSON(t *testing.T) {
	var q QueueItem
	err := q.FromJSON([]byte(`
			{
				"Msg" : "Hello world",
				"Dst" : "+9239",
				"Src" : "Source",
				"Enc" : "ucs",
				"Priority": 10
			}
		`))
	if err != nil {
		t.Fatalf("Error %s", err)
	}
	if q.Msg != "Hello world" || q.Priority != 10 {
		t.Fatalf("Values not correct in object. %+v", q)
	}
	err = q.FromJSON([]byte(`
			{
				"Msg" : "Hello world",
				"Dst" : "+9239",
				"Src" : "Source",
				"Enc" : "ucs",
				"Priority": "10"
			}
		`))
	if err == nil {
		t.Fatalf("Error didn't return on malformed json")
	}
}
func TestQueueItem_ToJSON(t *testing.T) {
	q := QueueItem{
		Msg:      "Hello world",
		Enc:      "UCS",
		Priority: 11,
		Src:      "My source",
		Dst:      "my dst",
	}
	expected := `{"Msg":"Hello world","Dst":"my dst","Src":"My source","Enc":"UCS","Priority":11}`
	b, _ := q.ToJSON()
	if string(b[:]) != expected {
		t.Fatalf("Given: \n%s\n. Expected: \n%s\n", b, expected)
	}
}
