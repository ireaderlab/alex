package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/martini-contrib/render"
	"gopkg.in/mgo.v2/bson"
)

type ConcurrencyPeriod struct {
	// concurrency step setting
	Concurrency int
	Duration    int
}

type BoomJob struct {
	// Benchmark Job using boom for stabilize concurrency
	Id bson.ObjectId `json:"id"        bson:"_id,omitempty"`
	// Job Name
	Name string
	// Http API Team Name
	Team string
	// Http API Project Name
	Project string
	// Http API Url
	Url string
	// Hosts Pool for randomize choice
	Hosts []string
	// Http API Method ["GET", "POST" ...]
	Method string
	// Parameters Pool for randomize choice
	Jsonified bool // application/json
	Seeds     []RequestSeed
	CreateTs  int64
	LastRunTs int64
	// Disable http keepalive default false
	DisableKeepAlive bool
	// Disable http gzip compression default false
	DisableCompression bool
	// Timeout duration for each request
	Timeout int
	// Concurrency Steppings
	Periods []ConcurrencyPeriod
	// Concurrent Job Concurrency in running
	CurrentConcurrency int
}

func (job *BoomJob) IsRunning() bool {
	// job is running?
	return G_RunningBoomJobs.Exists(job.Id.Hex())
}

func GetBoomJobs(req *http.Request, r render.Render) {
	var team = req.FormValue("team")
	var project = req.FormValue("project")
	var url = req.FormValue("url")
	var page = req.FormValue("p")
	var condition = bson.M{}
	if team != "" {
		condition["team"] = team
	}
	if project != "" {
		condition["project"] = project
	}
	if url != "" {
		condition["url"] = bson.M{"$regex": bson.RegEx{fmt.Sprintf("^%s", url), ""}}
	}
	if len(condition) == 0 {
		condition = nil
	}
	total, err := G_MongoDB.C("boom_jobs").Find(condition).Count()
	if err != nil {
		log.Panic(err)
	}
	var pager = NewPager(20, total)
	pager.CurrentPage, err = strconv.Atoi(page)
	pager.UrlPattern = fmt.Sprintf("/boom/?p=%%d&team=%s&project=%s", team, project)
	var jobs []BoomJob
	err = G_MongoDB.C("boom_jobs").Find(condition).Skip(pager.Offset()).Sort("-lastrunts").Limit(pager.Limit()).All(&jobs)
	if err != nil {
		log.Panic(err)
	}
	var context = make(map[string]interface{})
	context["jobs"] = jobs
	context["teams"] = GenTeamSelectors(team)
	context["project"] = project
	context["url"] = url
	context["pager"] = pager
	RenderTemplate(r, "boom_jobs", context)
}

func CreateBoomJob(req *http.Request, r render.Render) {
	var name = req.FormValue("name")
	var team = req.FormValue("team")
	var project = req.FormValue("project")
	var job = BoomJob{
		Id:                 bson.NewObjectId(),
		Name:               name,
		Team:               team,
		Hosts:              []string{"localhost:8000"},
		Project:            project,
		Jsonified:          false,
		Seeds:              []RequestSeed{RequestSeed{}},
		CreateTs:           time.Now().Unix(),
		LastRunTs:          time.Now().Unix(),
		DisableKeepAlive:   false,
		DisableCompression: false,
		Timeout:            10,
		Periods:            []ConcurrencyPeriod{ConcurrencyPeriod{10, 5}},
	}
	err := G_MongoDB.C("boom_jobs").Insert(&job)
	if err != nil {
		log.Panic(err)
	}
	r.Redirect(fmt.Sprintf("/boom/edit?job_id=%s", job.Id.Hex()))
}

type BoomEditForm struct {
	Job     *BoomJob
	Methods []MethodSelector
	Teams   []TeamSelector
}

func EditBoomJobPage(req *http.Request, r render.Render) {
	var jobId = req.FormValue("job_id")
	var job BoomJob
	err := G_MongoDB.C("boom_jobs").FindId(bson.ObjectIdHex(jobId)).One(&job)
	if err != nil {
		log.Panic(err)
	}
	var context = make(map[string]interface{})
	var form = BoomEditForm{Job: &job}
	form.Methods = GenMethodSelectors(job.Method)
	form.Teams = GenTeamSelectors(job.Team)
	context["form"] = form
	RenderTemplate(r, "boom_edit", context)
}

func EditBoomJob(req *http.Request, r render.Render) {
	req.ParseForm()
	var jobId = req.FormValue("job_id")
	var job BoomJob
	err := G_MongoDB.C("boom_jobs").FindId(bson.ObjectIdHex(jobId)).One(&job)
	if err != nil {
		log.Panic(err)
	}
	job.Name = req.FormValue("name")
	job.Team = req.FormValue("team")
	job.Project = req.FormValue("project")
	job.Method = req.FormValue("method")
	job.Url = req.FormValue("url")
	job.Jsonified = req.FormValue("jsonified") != ""
	var hosts []string
	for _, host := range req.Form["host"] {
		hosts = append(hosts, host)
	}
	job.Hosts = hosts
	var headerSeeds = []map[string]interface{}{}
	var paramSeeds = []map[string]interface{}{}
	var dataSeeds = []map[string]interface{}{}
	var jsonDataSeeds = []string{}
	for _, header := range req.Form["header"] {
		var seed map[string]interface{}
		json.Unmarshal([]byte(header), &seed)
		headerSeeds = append(headerSeeds, seed)
	}
	for _, param := range req.Form["param"] {
		var seed map[string]interface{}
		json.Unmarshal([]byte(param), &seed)
		paramSeeds = append(paramSeeds, seed)
	}
	if job.Jsonified {
		for _, data := range req.Form["data"] {
			jsonDataSeeds = append(jsonDataSeeds, data)
		}
	} else {
		for _, data := range req.Form["data"] {
			var seed map[string]interface{}
			json.Unmarshal([]byte(data), &seed)
			dataSeeds = append(dataSeeds, seed)
		}
	}
	job.Seeds = make([]RequestSeed, len(headerSeeds))
	for i := 0; i < len(headerSeeds); i++ {
		job.Seeds[i] = RequestSeed{headerSeeds[i], paramSeeds[i], nil, ""}
		if len(dataSeeds) > 0 {
			job.Seeds[i].Data = dataSeeds[i]
		} else {
			job.Seeds[i].JsonData = jsonDataSeeds[i]
		}
	}
	var changed = bson.M{
		"name":      job.Name,
		"team":      job.Team,
		"project":   job.Project,
		"method":    job.Method,
		"url":       job.Url,
		"hosts":     job.Hosts,
		"jsonified": job.Jsonified,
		"seeds":     job.Seeds,
	}
	var op = bson.M{"$set": changed}
	err = G_MongoDB.C("boom_jobs").UpdateId(job.Id, op)
	if err != nil {
		log.Panic(err)
	}
	r.Redirect("/boom/")
}

type BoomRunForm struct {
	Job *BoomJob
}

func RunBoomJobPage(req *http.Request, r render.Render) {
	var jobId = req.FormValue("job_id")
	if G_RunningBoomJobs.Exists(jobId) {
		r.Redirect(req.Referer())
		return
	}
	var job BoomJob
	err := G_MongoDB.C("boom_jobs").FindId(bson.ObjectIdHex(jobId)).One(&job)
	if err != nil {
		log.Panic(err)
	}
	var form = BoomRunForm{&job}
	var context = make(map[string]interface{})
	context["form"] = form
	RenderTemplate(r, "boom_run", context)
}

func RunBoomJob(req *http.Request, r render.Render) {
	req.ParseForm()
	var jobId = req.FormValue("job_id")
	var job BoomJob
	err := G_MongoDB.C("boom_jobs").FindId(bson.ObjectIdHex(jobId)).One(&job)
	if err != nil {
		log.Panic(err)
	}
	var timeout, _ = strconv.Atoi(req.FormValue("timeout"))
	var disableKeepAlive = req.FormValue("disable_keepalive") != ""
	var disableCompression = req.FormValue("disable_compression") != ""
	var concurrencies = req.Form["concurrency"]
	var durations = req.Form["duration"]
	var comment = req.FormValue("comment")
	var periods = []ConcurrencyPeriod{}
	for i, _ := range concurrencies {
		var concurrency, _ = strconv.Atoi(concurrencies[i])
		var duration, _ = strconv.Atoi(durations[i])
		periods = append(periods, ConcurrencyPeriod{concurrency, duration})
	}
	job.Timeout = timeout
	job.DisableKeepAlive = disableKeepAlive
	job.DisableCompression = disableCompression
	job.Periods = periods
	var changed = bson.M{
		"timeout":            job.Timeout,
		"disablekeepalive":   job.DisableKeepAlive,
		"disablecompression": job.DisableCompression,
		"periods":            job.Periods,
		"lastrunts":          time.Now().Unix(),
	}
	var op = bson.M{"$set": changed}
	err = G_MongoDB.C("boom_jobs").UpdateId(job.Id, op)
	if err != nil {
		log.Panic(err)
	}
	G_RunningBoomJobs.Put(job.Id.Hex())
	go AttackBoomJob(&job, comment)
	r.Redirect("/boom/")
}

func DeleteBoomJob(req *http.Request, r render.Render) {
	var jobId = req.FormValue("job_id")
	G_RunningBoomJobs.Delete(jobId)
	err := G_MongoDB.C("boom_jobs").RemoveId(bson.ObjectIdHex(jobId))
	if err != nil {
		log.Panic(err)
	}
	r.Redirect("/boom/")
}

func StopBoomJob(req *http.Request, r render.Render) {
	var jobId = req.FormValue("job_id")
	if G_RunningBoomJobs.Exists(jobId) {
		G_StoppingBoomJobs.Put(jobId)
	}
	r.Redirect(req.Referer())
}

func GetBoomLogs(req *http.Request, r render.Render) {
	var jobId = req.FormValue("job_id")
	var page = req.FormValue("p")
	var logs []AttackBoomLog
	var condition = bson.M{}
	if jobId != "" {
		condition = bson.M{"jobid": jobId}
	} else {
		condition = nil
	}
	total, err := G_MongoDB.C("boom_logs").Find(condition).Count()
	if err != nil {
		log.Panic(err)
	}
	var pager = NewPager(20, total)
	pager.CurrentPage, err = strconv.Atoi(page)
	pager.UrlPattern = fmt.Sprintf("/boom/logs?&p=%%d&job_id=%s", jobId)
	err = G_MongoDB.C("boom_logs").Find(condition).Skip(pager.Offset()).Sort("-startts").Limit(pager.Limit()).All(&logs)
	if err != nil {
		log.Panic(err)
	}
	var context = make(map[string]interface{})
	context["logs"] = logs
	context["jobId"] = jobId
	context["pager"] = pager
	RenderTemplate(r, "boom_logs", context)
}

func DeleteBoomLog(req *http.Request, r render.Render) {
	var logId = bson.ObjectIdHex(req.FormValue("log_id"))
	err := G_MongoDB.C("boom_logs").RemoveId(logId)
	if err != nil {
		log.Panic(err)
	}
	r.Redirect(req.Referer())
}

func GetBoomMetrics(req *http.Request, r render.Render) {
	var lg AttackBoomLog
	var lgId = bson.ObjectIdHex(req.FormValue("log_id"))
	err := G_MongoDB.C("boom_logs").FindId(lgId).One(&lg)
	if err != nil {
		log.Panic(err)
	}
	var context = make(map[string]interface{})
	context["log"] = &lg
	RenderTemplate(r, "boom_metrics", context)
}

type AttackBoomLog struct {
	Id        bson.ObjectId `json:"id"        bson:"_id,omitempty"`
	JobId     string
	JobName   string
	JobUrl    string
	JobDetail *BoomJob
	Comment   string
	State     string
	// Report List matching job stepping settings
	MetricsList []*Report
	StartTs     int64
	EndTs       int64
}

func (log *AttackBoomLog) IsRunning() bool {
	return log.State == "Running"
}

func (log *AttackBoomLog) ConcurrencyLatencyMetrics() string {
	var buffer bytes.Buffer
	for _, metrics := range log.MetricsList {
		buffer.WriteString(fmt.Sprintf("%v,%v\n", metrics.Concurrency, metrics.Latency))
	}
	return buffer.String()
}

func (log *AttackBoomLog) StatusCodesList() map[string]bool {
	var codeList = make(map[string]bool)
	for _, metrics := range log.MetricsList {
		for code, _ := range metrics.StatusCodeDist {
			codeList[fmt.Sprintf("%v", code)] = true
		}
	}
	return codeList
}

func (log *AttackBoomLog) StatusCodesMetrics() string {
	var buffer bytes.Buffer
	var codeList = log.StatusCodesList()
	for _, metrics := range log.MetricsList {
		buffer.WriteString(fmt.Sprintf("%v", metrics.Concurrency))
		for code, _ := range codeList {
			var count, ok = metrics.StatusCodeDist[code]
			if !ok {
				count = 0
			}
			buffer.WriteString(fmt.Sprintf(",%v", count))
		}
		buffer.WriteString("\n")
		buffer.WriteString(fmt.Sprintf("%v", metrics.Concurrency))
		for code, _ := range codeList {
			var count, ok = metrics.StatusCodeDist[code]
			if !ok {
				count = 0
			}
			buffer.WriteString(fmt.Sprintf(",%v", count))
		}
		buffer.WriteString("\n")
	}
	return buffer.String()
}

func AttackBoomJob(job *BoomJob, comment string) {
	// Begin attack target services
	var log = LogAttackBoomStart(job, comment)
	var metricsList []*Report
	shooter := NewRandomBoomShooter(job)
	for _, period := range job.Periods {
		var duration = time.Duration(period.Duration) * time.Second
		var boomer = Boomer{
			Shooter:            shooter,
			Duration:           duration,
			Concurrency:        period.Concurrency,
			Timeout:            job.Timeout,
			DisableCompression: job.DisableCompression,
			DisableKeepAlive:   job.DisableKeepAlive,
		}
		UpdateJobCurrentConcurrency(job, period.Concurrency)
		var metrics = boomer.Run()
		metricsList = append(metricsList, metrics)
		if G_StoppingBoomJobs.Exists(job.Id.Hex()) {
			G_StoppingBoomJobs.Delete(job.Id.Hex())
			break
		}
	}
	G_RunningBoomJobs.Delete(job.Id.Hex())
	UpdateJobCurrentConcurrency(job, 0)
	LogAttackBoomEnd(log, metricsList)
}

func UpdateJobCurrentConcurrency(job *BoomJob, concurrency int) {
	// realtime update job concurrency for displaying
	var op = bson.M{"$set": bson.M{"currentconcurrency": concurrency}}
	err := G_MongoDB.C("boom_jobs").UpdateId(job.Id, op)
	if err != nil {
		log.Panic(err)
	}
}

func LogAttackBoomStart(job *BoomJob, comment string) *AttackBoomLog {
	// Record attack log before attack starts
	var lg = AttackBoomLog{
		Id:        bson.NewObjectId(),
		JobId:     job.Id.Hex(),
		JobName:   job.Name,
		JobUrl:    job.Url,
		JobDetail: job,
		Comment:   comment,
		State:     "Running",
		StartTs:   time.Now().Unix(),
		EndTs:     0,
	}
	err := G_MongoDB.C("boom_logs").Insert(&lg)
	if err != nil {
		log.Panic(err)
	}
	return &lg
}

func LogAttackBoomEnd(lg *AttackBoomLog, metricsList []*Report) {
	// Record job reports after job finished
	var op = bson.M{"$set": bson.M{"metricslist": metricsList, "state": "End", "endts": time.Now().Unix()}}
	for k, v := range metricsList[0].ErrorDist {
		fmt.Printf("%#v, %#v\n", k, v)
	}
	err := G_MongoDB.C("boom_logs").UpdateId(lg.Id, op)
	if err != nil {
		log.Panic(err)
	}
}

func NewRandomBoomShooter(job *BoomJob) *RandomShooter {
	// Generate Request generator for boom
	var headers []http.Header
	var urls []string
	var bodies [][]byte
	var l = 0
	for _, host := range job.Hosts {
		for i := 0; i < len(job.Seeds); i++ {
			var header = http.Header{}
			for k, v := range job.Seeds[i].Header {
				switch v.(type) {
				case []interface{}:
					for _, vi := range v.([]interface{}) {
						header.Add(k, fmt.Sprintf("%v", vi))
					}
				default:
					header.Add(k, fmt.Sprintf("%v", v))
				}
			}
			var param = job.Seeds[i].Param
			var data = job.Seeds[i].Data
			var jsonData = job.Seeds[i].JsonData
			headers = append(headers, header)
			urls = append(urls, Urlcat(host, job.Url, param))
			if job.Jsonified {
				bodies = append(bodies, []byte(jsonData))
			} else {
				bodies = append(bodies, BodyBytes(data))
			}
			l++
		}
	}
	return &RandomShooter{
		Method:  job.Method,
		Urls:    urls,
		Bodies:  bodies,
		Headers: headers,
		L:       l,
	}
}
