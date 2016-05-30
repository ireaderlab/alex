package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/streadway/quantile"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"
)

type result struct {
	err        error
	statusCode int
	duration   time.Duration
}

type IShooter interface {
	// interface for shooting requests
	Next() *http.Request
}

type RandomShooter struct {
	// random requests shooter from seeds provided
	Method  string
	Urls    []string
	Headers []http.Header
	Bodies  [][]byte
	L       int
}

func (s *RandomShooter) Next() *http.Request {
	// generate next requests
	var i = rand.Intn(s.L)
	req, _ := http.NewRequest(s.Method, s.Urls[i], bytes.NewReader(s.Bodies[i]))
	for k, vs := range s.Headers[i] {
		req.Header[k] = make([]string, len(vs))
		copy(req.Header[k], vs)
	}
	if host := req.Header.Get("Host"); host != "" {
		req.Host = host
	}
	return req
}

type Boomer struct {
	Shooter            IShooter      // requests shooter
	Duration           time.Duration // time for attacking
	Concurrency        int           // go routines count
	Timeout            int           // timeout in seconds for each requests
	DisableCompression bool          // do not decompress gzipped content
	DisableKeepAlive   bool          // keepalive the connection
	results            [][]*result
}

func (b *Boomer) Run() *Report {
	b.results = make([][]*result, b.Concurrency)
	s := time.Now()
	b.runWorkers()
	var report = newReport(b.results, b.Concurrency, time.Now().Sub(s))
	report.finalize()
	return report
}

func (b *Boomer) makeRequest(c *http.Client, i int) {
	s := time.Now()
	var code int
	resp, err := c.Do(b.Shooter.Next())
	if err == nil {
		code = resp.StatusCode
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}
	var res = result{
		statusCode: code,
		duration:   time.Now().Sub(s),
		err:        err,
	}
	b.results[i] = append(b.results[i], &res)
}

func (b *Boomer) runWorker(i int) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		DisableCompression:  b.DisableCompression,
		DisableKeepAlives:   b.DisableKeepAlive,
		TLSHandshakeTimeout: time.Duration(b.Timeout) * time.Millisecond,
	}
	client := &http.Client{Transport: tr}
	start := time.Now()
	b.results[i] = []*result{}
	for {
		if time.Now().Sub(start) > b.Duration {
			break
		}
		b.makeRequest(client, i)
	}
}

func (b *Boomer) runWorkers() {
	// run attacker
	var wg sync.WaitGroup
	wg.Add(b.Concurrency)
	for i := 0; i < b.Concurrency; i++ {
		go func(k int) {
			b.runWorker(k)
			wg.Done()
		}(i)
	}
	wg.Wait()
}

type Report struct {
	Latency        time.Duration  // average latency
	Latency_P99    time.Duration  // p99 latency
	Latency_P95    time.Duration  // p95 latency
	Qps            float64        // qps
	Concurrency    int            // go routines count
	Requests       int            // total requests sent
	SuccessRatio   float64        // success ratio
	Duration       time.Duration  // time for attacking
	ErrorDist      map[string]int // error map
	StatusCodeDist map[string]int // status codes map

	avgTotal  float64
	results   [][]*result
	latencies *quantile.Estimator
}

func newReport(results [][]*result, concurrency int, duration time.Duration) *Report {
	return &Report{
		results:        results,
		Concurrency:    concurrency,
		Duration:       duration,
		StatusCodeDist: make(map[string]int),
		ErrorDist:      make(map[string]int),
		latencies: quantile.New(
			quantile.Known(0.50, 0.01),
			quantile.Known(0.95, 0.001),
			quantile.Known(0.99, 0.0005),
		),
	}
}

func (r *Report) finalize() {
	// 汇总报告
	var total = 0
	var success = 0
	for _, wresults := range r.results {
		for _, res := range wresults {
			if res.err != nil {
				r.ErrorDist[strings.Replace(res.err.Error(), ".", ":", -1)]++
			} else {
				r.latencies.Add(res.duration.Seconds())
				r.avgTotal += res.duration.Seconds()
				r.StatusCodeDist[fmt.Sprintf("%v", res.statusCode)]++
				success++
			}
			total++
		}
	}
	r.Qps = float64(r.latencies.Samples()) / r.Duration.Seconds()
	r.Latency = time.Duration(r.avgTotal*1000/float64(r.latencies.Samples())) * time.Millisecond
	r.Latency_P99 = time.Duration(r.latencies.Get(0.99)*1000) * time.Millisecond
	r.Latency_P95 = time.Duration(r.latencies.Get(0.95)*1000) * time.Millisecond
	r.Requests = total
	r.SuccessRatio = float64(success) * 100 / float64(total)
}
