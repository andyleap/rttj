package rttj

//go:generate gopherjs build ./client
//go:generate -command asset go run asset.go
//go:generate asset -wrap=handler -var=clientCode client.js
//go:generate asset -wrap=handler -var=clientMap client.js.map

import (
	"net/http"
)

func handler(a asset) http.Handler {
	return a
}
