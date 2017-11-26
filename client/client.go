package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"

	"github.com/gopherjs/eventsource"
	"github.com/gopherjs/gopherjs/js"
	"honnef.co/go/js/console"
	"honnef.co/go/js/dom"
	"github.com/evanphx/json-patch"
)

func Action(name, label string) template.HTML {
	return template.HTML(`<button type="button" data-name="`+name+`">`+label+`</button>`)
}

func main() {
	doc := dom.GetWindow().Document()
	templateNode := doc.GetElementByID("template")
	tpl := template.Must(template.New("main").Funcs(template.FuncMap{"action": Action}).Parse(templateNode.InnerHTML()))
	contentNode := doc.GetElementByID("content")
	curVal := []byte(``)

	es := eventsource.New("/events")
	es.AddEventListener("full", true, func(d *js.Object) {
		inData := d.Get("data").String()
		buf, _ := base64.StdEncoding.DecodeString(inData)
		curVal = buf
		var curJson interface{}
		json.Unmarshal(buf, &curJson)

		var tplBuf bytes.Buffer
		err := tpl.Execute(&tplBuf, curJson)
		if err != nil {
			console.Log(err.Error())
		}
		contentNode.SetInnerHTML(tplBuf.String())
	})

	es.AddEventListener("update", true, func(d *js.Object) {
		inData := d.Get("data").String()
		buf, _ := base64.StdEncoding.DecodeString(inData)
		console.Log(string(buf), string(curVal))
		curVal, _ = jsonpatch.MergePatch(curVal, buf)
		console.Log(string(curVal))
		var curJson interface{}

		json.Unmarshal(curVal, &curJson)
		
		var tplBuf bytes.Buffer
		err := tpl.Execute(&tplBuf, curJson)
		if err != nil {
			console.Log(err.Error())
		}
		contentNode.SetInnerHTML(tplBuf.String())
	})

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
