package rttj

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/evanphx/json-patch"
)

type Server struct {
	template string
	OnAction func(name string)

	mu    sync.Mutex
	value []byte
	chans []chan string
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
	update := base64.StdEncoding.EncodeToString(rawupdate)

	for l1 := 0; l1 < len(s.chans); l1++ {
		select {
			case s.chans[l1]<-update:
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
	c := make(chan string, 1)
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

	for update := range c {
		fmt.Fprintf(rw, `event: update
data: %s

`, update)
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
</head>
<body>	
<div id="content"></div>
<script id="template" type="go/template">
%s
</script>
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
