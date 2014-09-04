// pingpongStatus project main.go
package main

import (
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

	"github.com/billhathaway/serverSentEvents"

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
	history     []int
	available   bool
	lastUpdated time.Time
}

func (t tableStatus) String() string {
	return fmt.Sprintf("available=%v lastUpdated=%s", t.available, t.lastUpdated.Format(time.RFC3339))
}

func keepHistory() {
	t := time.NewTicker(time.Minute)
	for {
		<-t.C
		status.Lock()
		switch {
		case status.lastUpdated.Before(time.Now().Add(-1 * time.Minute)):
			status.history = append(status.history, stateUnknown)
		case status.available:
			status.history = append(status.history, stateAvailable)
		case !status.available:
			status.history = append(status.history, stateBusy)
		}
		historyLength := len(status.history)
		if historyLength > maxHistory {
			status.history = status.history[historyLength-maxHistory:]
		}
		status.Unlock()
	}
}

// fetch events forever, updating status
func fetchEvents(url string) {
	for {
		listener, err := sse.Listen(url)
		if err != nil {
			time.Sleep(time.Minute)
			continue
		}
		for event := range listener.C {
			se := sparkEvent{}
			err := json.Unmarshal([]byte(event.Data), &se)
			if err != nil {
				log.Print("Unmarshall error", err)
				continue
			}
			status.Lock()
			status.available = se.Data != "busy"
			status.lastUpdated = time.Now()
			status.Unlock()
			log.Println("received event", se)
		}
		log.Println("restarting connection")
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
		switch entry {
		case stateUnknown:
			color = "blue"
		case stateAvailable:
			color = "green"
		case stateBusy:
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
	remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)

	if status.lastUpdated.Before(time.Now().Add(-3 * time.Minute)) {
		color = "blue"
		info = "Unknown"
		fmt.Fprintf(w, `<html><head><title>%s</title><meta http-equiv="refresh" content="60"></head><body><p style="font-family:arial;color:%s;font-size:120px">%s</p>Sensor data old - last updated %s<p><img src="/graph"></body></html>`, info, color, info, status.lastUpdated.Format(time.RFC1123))
		return
	}
	if status.available {
		color = "green"
		info = "Available"
	}
	fmt.Fprintf(w, `<html><head><title>%s</title><meta http-equiv="refresh" content="60"></head><body><p style="font-family:arial;color:%s;font-size:120px">%s</p><p>Last updated %s</p><p>%d minutes of history</p><img src="/graph"></body></html>`, info, color, info, status.lastUpdated.Format(time.RFC1123), len(status.history))
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
