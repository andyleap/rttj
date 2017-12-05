package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	_ "time" //so we can format times

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

func main() {
	doc := dom.GetWindow().Document()
	templateNode := doc.GetElementByID("template")
	tpl := template.Must(template.New("main").Funcs(template.FuncMap{"action": Action}).Parse(templateNode.InnerHTML()))
	contentNode := doc.GetElementByID("content")
	curVal := []byte(``)

	view := &View{
		Tpl:  tpl,
		Root: contentNode,
		tree: &vdom.Tree{},
	}

	es := eventsource.New("/events")
	es.AddEventListener("full", true, func(d *js.Object) {
		inData := d.Get("data").String()
		buf, _ := base64.StdEncoding.DecodeString(inData)
		curVal = buf
		var curJson interface{}
		json.Unmarshal(buf, &curJson)

		err := view.Update(curJson)
		if err != nil {
			console.Log(err.Error())
		}
	})

	es.AddEventListener("update", true, func(d *js.Object) {
		inData := d.Get("data").String()
		buf, _ := base64.StdEncoding.DecodeString(inData)
		curVal, _ = jsonpatch.MergePatch(curVal, buf)
		var curJson interface{}

		json.Unmarshal(curVal, &curJson)

		err := view.Update(curJson)
		if err != nil {
			console.Log(err.Error())
		}
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
