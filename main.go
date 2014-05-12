// pingpongStatus project main.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

var (
	port    = "8888"
	verbose = false
	urlFile = "url.txt"
	url     = ""
)

type sparkResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	Info             string `json:"info"`
	Command          string `json:"cmd"`
	Name             string `json:"name"`
	Result           int    `json:"result"`
	CoreInfo         struct {
		LastHandshake string `json:"last_handshake_at"`
		Connected     bool   `json:"connected"`
	}
}

func querySparkAPI(w http.ResponseWriter, r *http.Request) {

	webResponse, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(w, "Problem contacting spark API\n%s\n", err.Error())
		return
	}

	body, err := ioutil.ReadAll(webResponse.Body)
	if err != nil {
		fmt.Fprintf(w, "Problem reading response\n%s\n", err.Error())
		return
	}

	response := sparkResponse{}
	err = json.Unmarshal(body, &response)
	if verbose {
		log.Printf("%#v", response)
	}

	if err != nil {
		fmt.Fprintf(w, "Problem unmarshalling json\n%s\n", err.Error())
		return
	}

	if response.Error != "" {
		fmt.Fprintf(w, "Problem with response\n%+v\n", response)
		return
	}

	status := "Available"
	color := "green"
	if response.Result < 30 {
		status = "Busy"
		color = "red"
	}

	fmt.Fprintf(w, `<html><head><title>%s</title><meta http-equiv="refresh" content="30"></head><body><p style="font-family:arial;color:%s;font-size:120px">%s</p></body></html>`, status, color, status)
}

func main() {
	flag.StringVar(&port, "p", port, "port to listen on")
	flag.BoolVar(&verbose, "v", verbose, "verbose logging")
	flag.StringVar(&url, "url", url, "URL to query")
	flag.StringVar(&urlFile, "file", urlFile, "file containing url to query")
	flag.Parse()
	if url == "" {
		data, err := ioutil.ReadFile(urlFile)
		if err != nil {
			log.Fatal(err)
		}
		url = string(data)
	}
	http.HandleFunc("/", querySparkAPI)
	err := http.ListenAndServe(":"+port, nil)
	log.Fatal(err)
}
