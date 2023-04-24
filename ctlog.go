package ctlog

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	log "github.com/root4loot/ctlog/pkg/log"
)

type Runner struct {
	Options Options         // options for the runner
	client  *http.Client    // http client
	Results chan Result     // channel to receive results
	Visited map[string]bool // map of visited targets
}

// Options contains options for the runner
type Options struct {
	Concurrency int    // number of concurrent requests
	Timeout     int    // timeout in seconds
	Delay       int    // delay in seconds
	DelayJitter int    // delay jitter in seconds
	Verbose     bool   // hide info messages
	UserAgent   string // user agent
}

type Results struct {
	Results []Result
}

type Result struct {
	Query          string `json:"query"`
	Error          error  `json:"error"`
	IssuerCaID     int    `json:"issuer_ca_id"`
	IssuerName     string `json:"issuer_name"`
	CommonName     string `json:"common_name"`
	NameValue      string `json:"name_value"`
	ID             int64  `json:"id"`
	EntryTimestamp string `json:"entry_timestamp"`
	NotBefore      string `json:"not_before"`
	NotAfter       string `json:"not_after"`
	SerialNumber   string `json:"serial_number"`
}

var seen map[string]bool // map of seen domains

// DefaultOptions returns default options
func DefaultOptions() *Options {
	return &Options{
		Concurrency: 10,
		Timeout:     30,
		Delay:       0,
		DelayJitter: 0,
		UserAgent:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:103.0) Gecko/20100101 Firefox/103.0",
	}
}

// NewRunner returns a new runner
func NewRunner() *Runner {
	options := DefaultOptions()
	return &Runner{
		Results: make(chan Result),
		Visited: make(map[string]bool),
		Options: *options,
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
				MaxIdleConnsPerHost:   options.Concurrency,
				ResponseHeaderTimeout: time.Duration(options.Timeout) * time.Second,
			},
			Timeout: time.Duration(options.Timeout) * time.Second,
		},
	}
}

// Run starts the runner
func (r *Runner) Run(targets ...string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.Options.Timeout)*time.Second)
	seen = make(map[string]bool)
	defer cancel()

	sem := make(chan struct{}, r.Options.Concurrency)
	var wg sync.WaitGroup

	for _, target := range targets {
		if !r.Visited[target] {
			r.Visited[target] = true

			sem <- struct{}{}
			wg.Add(1)
			go func(u string) {
				defer func() { <-sem }()
				defer wg.Done()
				r.query(ctx, u, r.client)
				time.Sleep(time.Millisecond * 100) // make room for processing results
			}(target)
			time.Sleep(r.getDelay() * time.Millisecond) // delay between requests
		}
	}
	wg.Wait()
}

func (r *Result) Domain() string {
	domain := strings.Trim(r.CommonName, "*.")
	u, err := url.Parse("http://" + domain)
	if err != nil {
		log.Warningf("%v", err.Error())
		return ""
	}
	return u.Hostname()
}

func (r *Runner) query(ctx context.Context, target string, client *http.Client) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://crt.sh/?q="+url.QueryEscape(target)+"&output=json", nil)

	if err != nil {
		log.Warningf("%v", err.Error())
		r.Results <- Result{Error: err}
	}

	if r.Options.UserAgent != "" {
		req.Header.Add("User-Agent", r.Options.UserAgent)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Warningf("%v", err.Error())
		r.Results <- Result{Query: target, Error: err}
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Warningf("%v", err.Error())
		r.Results <- Result{Query: target, Error: err}
	}

	var results []Result
	err = json.Unmarshal(bodyBytes, &results)
	if err != nil {
		log.Warningf("%v", err.Error())
		r.Results <- Result{Query: target, Error: err}
	}

	for _, result := range results {
		if !seen[result.CommonName] {
			seen[result.CommonName] = true
			result.Query = target
			r.Results <- result
		}
	}
}

// delay returns total delay from options
func (r *Runner) getDelay() time.Duration {
	if r.Options.DelayJitter != 0 {
		return time.Duration(r.Options.Delay + rand.Intn(r.Options.DelayJitter))
	}
	return time.Duration(r.Options.Delay)
}