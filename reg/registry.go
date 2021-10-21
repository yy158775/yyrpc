package main

import (
	"RPC/registry"
	"net/http"
)

var registryPath = "localhost:9091/registry"

func main() {
	registry.DefaultYyRegister.HandleHTTP("/registry")
	http.ListenAndServe(":9091",nil)
}