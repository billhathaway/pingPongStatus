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
	"sync"
	"time"

	"github.com/ajstarks/svgo"
)

const (
	stateUnknown = iota
	stateAvailable
	stateBusy
	maxHistory       = 60
	graphMinuteWidth = 10
	graphHeight      = 30
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
	sync.Mutex
	history     []bool
	available   bool
	lastUpdated time.Time
}

func (t tableStatus) String() string {
	return fmt.Sprintf("available=%v lastUpdated=%s", t.available, t.lastUpdated.Format(time.RFC3339))
}

func keepHistory() {
	status.history = []bool{true, true, false, false, true, false, false, true, true, false, false, false, false, true}
	t := time.NewTicker(time.Minute)
	for {
		<-t.C
		status.Lock()
		status.history = append(status.history, status.available)
		historyLength := len(status.history)
		if historyLength > maxHistory {
			status.history = status.history[historyLength-maxHistory:]
		}
		status.Unlock()
	}
}

// fetch events forever, updating status
func fetchEvents(url string) {
	badLineCount := 0
	var buf bytes.Buffer
	buffed := &bufio.Reader{}
	for {
		response, err := http.Get(url)
		if err != nil {
			log.Printf("event=api_call status=error message=%q\n", err.Error())
			time.Sleep(time.Minute)
			continue
		}
		log.Printf("event=api_call status=success %s\n", url)
		event := sparkEvent{}
		receivedTableStatus := false
		buffed = bufio.NewReader(response.Body)
		for {
			line, err := buffed.ReadBytes('\n')
			if err != nil {
				log.Printf("event=error_from_buffered_reader error=%q\n", err.Error())
				response.Body.Close()
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
						log.Printf("event=unmarshall_json status=error message=%q data=[%s]\n", err.Error(), buf.String())
						buf.Reset()
						continue
					}
					status.available = event.Data == "free"
					status.lastUpdated = time.Now()
					badLineCount = 0
					log.Println(status)
				}
				buf.Reset()
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
		time.Sleep(time.Minute)
	}

}

func generateGraph(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	s := svg.New(w)
	status.Lock()
	defer status.Unlock()
	historyLength := len(status.history)
	s.Start(graphMinuteWidth*historyLength, graphHeight)
	var color string
	var lineHeight int
	for index, entry := range status.history {
		if entry {
			color = "green"
		} else {
			color = "red"
		}
		s.Rect(graphMinuteWidth*index, 10, graphMinuteWidth, graphHeight, "fill:"+color)
		if index%5 == 0 {
			lineHeight = graphHeight
		} else {
			lineHeight = graphHeight / 2
		}
		s.Line(graphMinuteWidth*index, 10, graphMinuteWidth*index, lineHeight, "stroke:black")

	}
	s.End()

}

func showStatus(w http.ResponseWriter, r *http.Request) {
	info := "Busy"
	color := "red"
	if status.available {
		color = "green"
		info = "Available"
	}
	fmt.Fprintf(w, `<html><head><title>%s</title><meta http-equiv="refresh" content="60"></head><body><p style="font-family:arial;color:%s;font-size:120px">%s</p>Last updated %s<p><img src="/graph"></body></html>`, info, color, info, status.lastUpdated.Format(time.RFC1123))
	remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	log.Printf("ip=%s event=showStatus state=%s\n", remoteIP, info)

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
	go keepHistory()
	http.HandleFunc("/", showStatus)
	http.HandleFunc("/graph", generateGraph)
	http.Handle("/favicon.ico", http.NotFoundHandler())
	err := http.ListenAndServe(":"+port, nil)
	log.Fatal(err)
}
