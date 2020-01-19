package main

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
)

const powercapDir = "/sys/class/powercap/"

// Cap is a payload that is expected from Elektron to cap a node.
type Cap struct {
	Percentage int
}

// CapResponse is the payload sent with information about the capping call
type CapResponse struct {
	CappedZones []string `json:"cappedZones"`
	FailedZones []string `json:"failedZones"`
	Error       *string  `json:"error"`
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Unsupported endpoint %s", html.EscapeString(r.URL.Path))
	})

	http.HandleFunc("/powercap", powercapEndpoint)
	log.Fatal(http.ListenAndServe(":9090", nil))
}

// Handler for the powercapping HTTP API endpoint.
func powercapEndpoint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var payload Cap
	var response CapResponse

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&payload)
	if err != nil {
		errorMsg := "error parsing payload: " + err.Error()
		response.Error = &errorMsg
		json.NewEncoder(w).Encode(response)
		return
	}

	cappedZones, failedZones, err := capNode(powercapDir, payload.Percentage)
	if err != nil {
		errorMsg := err.Error()
		response.Error = &errorMsg
	}

	response.CappedZones = cappedZones
	response.FailedZones = failedZones

	json.NewEncoder(w).Encode(response)
}
