package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"time"

	"github.com/albrow/vdom"
	"github.com/evanphx/json-patch"
	"github.com/gopherjs/eventsource"
	"github.com/gopherjs/gopherjs/js"
	"honnef.co/go/js/console"
	"honnef.co/go/js/dom"
)

func Action(name, label string) template.HTML {
	return template.HTML(`<button type="button" data-name="` + name + `">` + label + `</button>`)
}

type View struct {
	Tpl  *template.Template
	Root dom.Element
	tree *vdom.Tree
}

func (v *View) Update(data interface{}) error {
	buf := bytes.NewBuffer([]byte{})
	if err := v.Tpl.Execute(buf, data); err != nil {
		return err
	}

	newTree, err := vdom.Parse(buf.Bytes())
	if err != nil {
		return err
	}

	patches, err := vdom.Diff(v.tree, newTree)
	if err != nil {
		return err
	}

	if err := patches.Patch(v.Root); err != nil {
		return err
	}

	v.tree = newTree
	return nil
}

var view *View
var curData []byte

func main() {
	doc := dom.GetWindow().Document()
	templateNode := doc.GetElementByID("template")
	tpl := template.Must(template.New("main").Funcs(template.FuncMap{"action": Action}).Parse(templateNode.InnerHTML()))
	contentNode := doc.GetElementByID("content")

	view = &View{
		Tpl:  tpl,
		Root: contentNode,
		tree: &vdom.Tree{},
	}

	ConnectES()

	doc.AddEventListener("click", true, func(evt dom.Event) {
		target := evt.Target()
		if !target.HasAttribute("data-name") {
			return
		}
		name := target.GetAttribute("data-name")
		vals := url.Values{}
		vals.Set("name", name)
		go http.PostForm("/action", vals)
	})
}

func ConnectES() {
	es := eventsource.New("/events")
	es.AddEventListener("full", true, FullEvent)
	es.AddEventListener("update", true, UpdateEvent)
	missedTicks := 0

	es.AddEventListener("tick", true, func(d *js.Object) {
		missedTicks = 0
	})

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			missedTicks++
			if missedTicks > 10 {
				break
			}
		}
		es.Close()
		ConnectES()
	}()
}

func FullEvent(d *js.Object) {
	inData := d.Get("data").String()
	buf, _ := base64.StdEncoding.DecodeString(inData)
	curData = buf
	var curJson interface{}
	json.Unmarshal(buf, &curJson)

	err := view.Update(curJson)
	if err != nil {
		console.Log(err.Error())
	}
}

func UpdateEvent(d *js.Object) {
	inData := d.Get("data").String()
	buf, _ := base64.StdEncoding.DecodeString(inData)
	curData, _ = jsonpatch.MergePatch(curData, buf)
	var curJson interface{}

	json.Unmarshal(curData, &curJson)

	err := view.Update(curJson)
	if err != nil {
		console.Log(err.Error())
	}
}
