package rttj

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/evanphx/json-patch"
)

type Server struct {
	template string
	OnAction func(name string)

	mu    sync.Mutex
	value []byte
	chans []chan event
}

type event struct {
	name string
	data string
}

func New(template string, value interface{}) (*Server, error) {
	v, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return &Server{
		template: template,
		value:    v,
	}, nil
}

func (s *Server) Update(new interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	newv, err := json.Marshal(new)
	if err != nil {
		return err
	}

	rawupdate, err := jsonpatch.CreateMergePatch(s.value, newv)
	if err != nil {
		return err
	}
	update := event{
		name: "update",
		data: base64.StdEncoding.EncodeToString(rawupdate),
	}

	for l1 := 0; l1 < len(s.chans); l1++ {
		select {
		case s.chans[l1] <- update:
		default:
			close(s.chans[l1])
			s.chans[l1] = s.chans[len(s.chans)-1]
			s.chans = s.chans[:len(s.chans)-1]
		}
	}

	s.value = newv
	return nil
}

func (s *Server) events(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	c := make(chan event, 5)
	flusher := rw.(http.Flusher)
	func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.chans = append(s.chans, c)

		data := base64.StdEncoding.EncodeToString(s.value)

		fmt.Fprintf(rw, `event: full
data: %s

`, data)
		flusher.Flush()
	}()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	go func() {
		for t := range ticker.C {
			c <- event{
				name: "tick",
				data: fmt.Sprint(t.Unix()),
			}
		}
	}()

	for update := range c {
		fmt.Fprintf(rw, `event: %s
data: %s

`, update.name, update.data)
		flusher.Flush()
	}
}

func (s *Server) action(rw http.ResponseWriter, req *http.Request) {
	name := req.FormValue("name")
	if s.OnAction != nil {
		s.OnAction(name)
	}
}

func (s *Server) index(rw http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(rw,
		`<html>
<head>
<link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/4.0.0-beta.2/css/bootstrap.min.css" integrity="sha384-PsH8R72JQ3SOdhVi3uxftmaW6Vc51MKb0q5P2rRUpPvrszuE4W1povHYgTpBfshb" crossorigin="anonymous">
</head>
<body>	
<div id="content"></div>
<script id="template" type="go/template">
%s
</script>
<script src="https://code.jquery.com/jquery-3.2.1.slim.min.js" integrity="sha384-KJ3o2DKtIkvYIK3UENzmM7KCkRr/rE9/Qpg6aAZGJwFDMVNA/GpGFF93hXpG5KkN" crossorigin="anonymous"></script>
<script src="https://cdnjs.cloudflare.com/ajax/libs/popper.js/1.12.3/umd/popper.min.js" integrity="sha384-vFJXuSJphROIrBnz7yo7oB41mKfc8JzQZiCq4NCceLEaO4IHwicKwpJf9c9IpFgh" crossorigin="anonymous"></script>
<script src="https://maxcdn.bootstrapcdn.com/bootstrap/4.0.0-beta.2/js/bootstrap.min.js" integrity="sha384-alpBpkh1PFOepccYVYDB4do5UnbKysX5WZXm3XxPqe5iKTfUKjNkCk9SaVuEZflJ" crossorigin="anonymous"></script>
<script src="/client.js"></script>
</body>
</html>`, s.template)
}

func (s *Server) Run() error {
	mux := http.NewServeMux()
	mux.Handle("/client.js", clientCode)
	mux.Handle("/client.js.map", clientMap)
	mux.HandleFunc("/events", s.events)
	mux.HandleFunc("/action", s.action)
	mux.HandleFunc("/", s.index)
	return http.ListenAndServe(":8080", mux)
}
