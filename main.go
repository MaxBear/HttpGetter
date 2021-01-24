package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

var urlsMutex = sync.RWMutex{}
var hostsMutex = sync.RWMutex{}

type Request struct {
	Idx int
	Url string
}

type Result struct {
	Idx        int
	Url        string
	OutputFile string
	Err        error
}

func HttpGet(url string, method string, message interface{}) (html string, err error) {

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	var bytesRepresentation []byte
	if message != nil {
		bytesRepresentation, err = json.Marshal(message)
	}
	if err != nil {
		return
	}

	var req *http.Request
	if method == "GET" {
		req, err = http.NewRequest(method, url, nil)
	} else if method == "POST" {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(bytesRepresentation))
		log.Print(bytes.NewBuffer(bytesRepresentation))
		req.Header.Set("Content-Type", "application/json")
	}
	if err != nil {
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	bodyText, err := ioutil.ReadAll(resp.Body)
	html = string(bodyText)
	return
}

func getUrls(fname string) (urls []string, err error) {
	urls = make([]string, 0)

	file, err := os.Open(fname)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		urls = append(urls, scanner.Text())
	}
	return
}

func parseUrl(url string) (hostname string) {
	tokens := strings.Split(url, "/")
	if len(tokens) >= 2 {
		hostname = tokens[2]
	}
	return
}

func doGet(req Request, results chan Result, urls map[string]int, hosts map[string]int, hostname string) {
	result := Result{
		Idx: req.Idx,
		Url: req.Url,
	}
	urlsMutex.RLock()
	idx, processed := urls[req.Url]
	urlsMutex.RUnlock()
	if !processed {
		urlsMutex.Lock()
		urls[req.Url] = req.Idx
		urlsMutex.Unlock()
		html, err := HttpGet(req.Url, "GET", nil)
		if err != nil {
			result.Err = err
		} else {
			fname := fmt.Sprintf("url_%d.html", req.Idx)
			f, err := os.Create(fname)
			if err == nil {
				f.Write([]byte(html))
				f.Close()
			}
			result.OutputFile = fname
		}
	} else {
		result.OutputFile = fmt.Sprintf("url_%d.html", idx)
	}
	hostsMutex.Lock()
	hosts[hostname] -= 1
	hostsMutex.Unlock()
	results <- result
}

func httpGetter(requests chan Request, results chan Result) {
	urls := make(map[string]int)
	hosts := make(map[string]int)
	//backlogs := make(chan Request)
	for {
		select {
		case req, ok := <-requests:
			if !ok {
				return
			}
			hostname := parseUrl(req.Url)
			_, ook := hosts[hostname]
			if !ook {
				hosts[hostname] = 0
			}
			hosts[hostname] += 1
			//if hosts[hostname] > 3 {
			//		continue
			//		}

			go doGet(req, results, urls, hosts, hostname)

			//		case req, _ := <-backlogs:
			//			hostname := parseUrl(req.Url)
			//			go doGet(req, results, urls, hosts, hostname)
		}
	}
}

func main() {

	var (
		fname = flag.String("f", "", "Text file contains urls to test")
	)
	flag.Parse()

	if len(*fname) == 0 {
		fmt.Printf("Please enter a valid file name\n")
		return
	}

	urls, err := getUrls(*fname)
	if err != nil {
		log.Fatal(err)
	}
	requests := make(chan Request)
	results := make(chan Result)

	go func() {
		for i, url := range urls {
			requests <- Request{Idx: i, Url: url}
		}
		close(requests)
	}()

	httpGetter(requests, results)

	cnt := 0
	for result := range results {
		cnt += 1
		if result.Err == nil {
			log.Printf("%d %s output=> %s", result.Idx, result.Url, result.OutputFile)
		} else {
			log.Printf("%d %s error=> %s", result.Idx, result.Url, result.Err)
		}
		if len(urls) == cnt {
			close(results)
			break
		}
	}
}
