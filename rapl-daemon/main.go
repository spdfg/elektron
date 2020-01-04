package main

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
)

const powercapDir = "/sys/class/powercap/"

// Cap is a payload that is expected from Elektron to cap a node
type Cap struct {
	Percentage int
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Unsupported endpoint %s", html.EscapeString(r.URL.Path))
	})

	http.HandleFunc("/powercap", powercapEndpoint)
	log.Fatal(http.ListenAndServe(":9090", nil))
}

// Handler powercapping HTTP API endpoint
func powercapEndpoint(w http.ResponseWriter, r *http.Request) {
	var payload Cap
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&payload)
	if err != nil {
		http.Error(w, "Error parsing payload: "+err.Error(), 400)
		return
	}

	if payload.Percentage < 0 || payload.Percentage > 100 {
		http.Error(w, "Bad payload: percentage must be between 0 and 100", 400)
		return
	}

	err = capNode(powercapDir, payload.Percentage)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	fmt.Fprintf(w, "Capped node at %d percent", payload.Percentage)
}
