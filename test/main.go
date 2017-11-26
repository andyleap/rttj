package main

import (
	"time"
	"math/rand"

	"github.com/andyleap/rttj"
)

var tpl = `
There are currently {{.Servers}} active servers! {{action "cancel" "Cancel"}}<br>
Canceled {{.Cancels}} times
<pre>{{range .ServerNames}}{{.}}
{{end}}</pre>
`

type tplData struct {
	Servers int
	ServerNames []string
	Cancels int
}

func main() {
	data := tplData{}
	data.ServerNames = []string{"kafka-1", "kafka-2", "kafka-3", "kafka-4", "kafka-5", "kafka-6"}
	data.Servers = 5
	data.Cancels = 0

	s, _ := rttj.New(tpl, data)
	s.OnAction = func(name string) {
		if name == "cancel" {
			data.Cancels++
			s.Update(data)
		}
	}

	go func() {
		for range time.NewTicker(5*time.Second).C {
			data.Servers = rand.Intn(100)
			s.Update(data)
		}
	}()

	s.Run()
}