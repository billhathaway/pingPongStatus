// pingpongStatus project main.go
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

var (
	port    = "8888"
	verbose = false
	urlFile = "url.txt"
	url     = ""
	status  tableStatus
)

type sparkEvent struct {
	Data        string `json:"data"`
	Ttl         string `json:"ttl"`
	PublishedAt string `json:"publised_at"`
	Coreid      string `json:"coreid"`
}

type tableStatus struct {
	available   bool
	lastUpdated time.Time
}

func (t tableStatus) String() string {
	return fmt.Sprintf("available=%v lastUpdated=%s", t.available, t.lastUpdated.Format(time.RFC3339))
}

// fetch events forever, updating status
func fetchEvents(url string) {
	badLineCount := 0
	for {
		response, err := http.Get(url)
		if err != nil {
			log.Printf("event=api_call status=error message=%q\n", err.Error())
			time.Sleep(time.Minute)
			continue
		}
		var buf bytes.Buffer
		event := sparkEvent{}
		receivedTableStatus := false
		buffed := bufio.NewReader(response.Body)
		for {
			line, err := buffed.ReadBytes('\n')
			if err != nil {
				log.Printf("event=error_from_buffered_reader error=%q\n", err.Error())
				break
			}
			switch {
			// ignore lines starting with colon per spec
			case bytes.HasPrefix(line, []byte(":")):
			// skip per spec
			case bytes.HasPrefix(line, []byte("event:")):
				if string(line[7:]) == "tableStatus\n" {
					receivedTableStatus = true
				} else {
					log.Printf("Skipping event %s", line)
				}
			case bytes.HasPrefix(line, []byte("data:")):
				if receivedTableStatus {
					buf.Write(line[6:])
				}
			case len(line) == 1:
				if receivedTableStatus {
					receivedTableStatus = false
					err = json.Unmarshal(buf.Bytes(), &event)
					if err != nil {
						log.Printf("event=unmarshall_json status=error message=%q data=%s\n", err.Error(), buf.String())
						buf.Reset()
						continue
					}
					buf.Reset()
					status.available = event.Data == "free"
					status.lastUpdated = time.Now()
					log.Println(status)
				}
			default:
				log.Printf("event=unknown_line_received line=[%s]\n", line)
				badLineCount++
				if badLineCount > 1000 {
					log.Fatalln("Too many bad lines")
				}
				break
			}
		}
		response.Body.Close()
	}

}

func showStatus(w http.ResponseWriter, r *http.Request) {
	info := "Busy"
	color := "red"
	if status.available {
		color = "green"
		info = "Available"
	}
	fmt.Fprintf(w, `<html><head><title>%s</title><meta http-equiv="refresh" content="60"></head><body><p style="font-family:arial;color:%s;font-size:120px">%s</p>Last updated %s</body></html>`, info, color, info, status.lastUpdated.Format(time.RFC1123))
	remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	log.Printf("ip=%s event=showStatus state=%s\n", remoteIP, info)

}

func blackHole(w http.ResponseWriter, r *http.Request) {

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
		url = strings.TrimSpace(string(data))

	}
	go fetchEvents(url)
	http.HandleFunc("/", showStatus)
	http.HandleFunc("/favicon.ico", blackHole)
	err := http.ListenAndServe(":"+port, nil)
	log.Fatal(err)
}
