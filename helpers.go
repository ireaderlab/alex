package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/martini-contrib/render"
	"html/template"
	"net/url"
	"sync"
	"time"
)

var Pipelines = template.FuncMap{
	"strftime": Strftime,
	"json":     Json,
}

func Strftime(ts int64) string {
	if ts == 0 {
		return ""
	}
	t := time.Unix(ts, 0)
	return t.Format("2006-01-02 15:04:05")
}

func Json(value interface{}) string {
	if value == nil {
		return "{}"
	}
	result, _ := json.Marshal(value)
	return string(result)
}

func BodyBytes(data map[string]interface{}) []byte {
	var buffer bytes.Buffer
	i := 0
	for k, v := range data {
		var item = fmt.Sprintf("%s=%v", k, v)
		buffer.WriteString(item)
		if i < len(data)-1 {
			buffer.WriteString("&")
		}
		i++
	}
	return buffer.Bytes()
}

func Urlcat(host string, urls string, params map[string]interface{}) string {
	var protocol = "http"
	var u, _ = url.Parse(fmt.Sprintf("%s://%s%s", protocol, host, urls))
	var values, _ = url.ParseQuery(u.RawQuery)
	for k, v := range params {
		values.Add(k, fmt.Sprintf("%v", v))
	}
	u.RawQuery = values.Encode()
	return u.String()
}

func GenMethodSelectors(method string) []MethodSelector {
	methods := make([]MethodSelector, 5)
	methods[0] = MethodSelector{"GET", false}
	methods[1] = MethodSelector{"POST", false}
	methods[2] = MethodSelector{"PUT", false}
	methods[3] = MethodSelector{"DELETE", false}
	methods[4] = MethodSelector{"HEAD", false}
	if method == "GET" {
		methods[0].Selected = true
	} else if method == "POST" {
		methods[1].Selected = true
	} else if method == "PUT" {
		methods[2].Selected = true
	} else if method == "DELETE" {
		methods[3].Selected = true
	} else if method == "HEADER" {
		methods[4].Selected = true
	} else {
		methods[0].Selected = true
	}
	return methods
}

func GenTeamSelectors(team string) []TeamSelector {
	var teams = make([]TeamSelector, len(G_AlexTeams)+1)
	teams[0] = TeamSelector{"", false}
	for i := 1; i <= len(G_AlexTeams); i++ {
		teams[i] = TeamSelector{G_AlexTeams[i-1], false}
	}
	for i := 0; i <= len(G_AlexTeams); i++ {
		if teams[i].Team == team {
			teams[i].Selected = true
		}
	}
	return teams
}

func MaxInt(nums ...int) int {
	max := nums[0]
	for _, num := range nums {
		if num > max {
			max = num
		}
	}
	return max
}

func RenderTemplate(r render.Render, tmpl string, context map[string]interface{}) {
	context["ShowLayout"] = G_ShowLayout
	r.HTML(200, tmpl, context)
}

type ConcurrentSet struct {
	// thread safe string set
	d     map[string]bool
	mutex sync.Mutex
}

func NewConcurrentSet() *ConcurrentSet {
	return &ConcurrentSet{map[string]bool{}, sync.Mutex{}}
}

func (this *ConcurrentSet) Empty() bool {
	return len(this.d) == 0
}

func (this *ConcurrentSet) Size() int {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	return len(this.d)
}

func (this *ConcurrentSet) Put(key string) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.d[key] = true
}

func (this *ConcurrentSet) Exists(key string) bool {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	_, ok := this.d[key]
	return ok
}

func (this *ConcurrentSet) Delete(key string) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	delete(this.d, key)
}
