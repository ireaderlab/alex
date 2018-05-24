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
	vegeta "github.com/tsenart/vegeta/lib"
	"gopkg.in/mgo.v2/bson"
)

type RatePeriod struct {
	// qps step setting
	Rate     uint64
	Duration uint
}

type VegetaJob struct {
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
	Hosts     []string
	Method    string
	Jsonified bool // application/json
	// Parameters Pool for randomize choice
	Seeds     []RequestSeed
	CreateTs  int64
	LastRunTs int64
	// Initial concurrency for vegeta
	Workers uint64
	// Timeout duration for each request
	Timeout int
	// Redirect times for each request
	Redirects int
	// http keepalive
	Keepalive bool
	// qps step settings
	Periods []RatePeriod
	// current qps in running
	CurrentRate uint64
}

func (job *VegetaJob) IsRunning() bool {
	return G_RunningVegetaJobs.Exists(job.Id.Hex())
}

func GetVegetaJobs(req *http.Request, r render.Render) {
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
	total, err := G_MongoDB.C("vegeta_jobs").Find(condition).Count()
	if err != nil {
		log.Panic(err)
	}
	var pager = NewPager(20, total)
	pager.CurrentPage, err = strconv.Atoi(page)
	pager.UrlPattern = fmt.Sprintf("/vegeta/?p=%%d&team=%s&project=%s", team, project)
	var jobs []VegetaJob
	err = G_MongoDB.C("vegeta_jobs").Find(condition).Skip(pager.Offset()).Sort("-lastrunts").Limit(pager.Limit()).All(&jobs)
	if err != nil {
		log.Panic(err)
	}
	var context = make(map[string]interface{})
	context["jobs"] = jobs
	context["teams"] = GenTeamSelectors(team)
	context["project"] = project
	context["url"] = url
	context["pager"] = pager
	RenderTemplate(r, "vegeta_jobs", context)
}

func CreateVegetaJob(req *http.Request, r render.Render) {
	var name = req.FormValue("name")
	var team = req.FormValue("team")
	var project = req.FormValue("project")
	var job = VegetaJob{
		Id:        bson.NewObjectId(),
		Name:      name,
		Team:      team,
		Hosts:     []string{"localhost:8000"},
		Project:   project,
		Jsonified: false,
		Seeds:     []RequestSeed{RequestSeed{}},
		CreateTs:  time.Now().Unix(),
		LastRunTs: time.Now().Unix(),
		Workers:   100,
		Timeout:   10,
		Redirects: 1,
		Keepalive: true,
		Periods:   []RatePeriod{RatePeriod{10, 5}},
	}
	err := G_MongoDB.C("vegeta_jobs").Insert(&job)
	if err != nil {
		log.Panic(err)
	}
	r.Redirect(fmt.Sprintf("/vegeta/edit?job_id=%s", job.Id.Hex()))
}

type VegetaEditForm struct {
	Job     *VegetaJob
	Methods []MethodSelector
	Teams   []TeamSelector
}

func EditVegetaJobPage(req *http.Request, r render.Render) {
	var jobId = req.FormValue("job_id")
	var job VegetaJob
	err := G_MongoDB.C("vegeta_jobs").FindId(bson.ObjectIdHex(jobId)).One(&job)
	if err != nil {
		log.Panic(err)
	}
	var context = make(map[string]interface{})
	var form = VegetaEditForm{Job: &job}
	form.Methods = GenMethodSelectors(job.Method)
	form.Teams = GenTeamSelectors(job.Team)
	context["form"] = form
	RenderTemplate(r, "vegeta_edit", context)
}

func EditVegetaJob(req *http.Request, r render.Render) {
	req.ParseForm()
	var jobId = req.FormValue("job_id")
	var job VegetaJob
	err := G_MongoDB.C("vegeta_jobs").FindId(bson.ObjectIdHex(jobId)).One(&job)
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
	err = G_MongoDB.C("vegeta_jobs").UpdateId(job.Id, op)
	if err != nil {
		log.Panic(err)
	}
	r.Redirect("/vegeta/")
}

type VegetaRunForm struct {
	Job *VegetaJob
}

func RunVegetaJobPage(req *http.Request, r render.Render) {
	var jobId = req.FormValue("job_id")
	// same job, no concurrent
	if G_RunningVegetaJobs.Exists(jobId) {
		r.Redirect(req.Referer())
		return
	}
	var job VegetaJob
	err := G_MongoDB.C("vegeta_jobs").FindId(bson.ObjectIdHex(jobId)).One(&job)
	if err != nil {
		log.Panic(err)
	}
	var form = VegetaRunForm{&job}
	var context = make(map[string]interface{})
	context["form"] = form
	RenderTemplate(r, "vegeta_run", context)
}

func RunVegetaJob(req *http.Request, r render.Render) {
	req.ParseForm()
	var jobId = req.FormValue("job_id")
	var job VegetaJob
	err := G_MongoDB.C("vegeta_jobs").FindId(bson.ObjectIdHex(jobId)).One(&job)
	if err != nil {
		log.Panic(err)
	}
	var workers, _ = strconv.Atoi(req.FormValue("workers"))
	var timeout, _ = strconv.Atoi(req.FormValue("timeout"))
	var redirects, _ = strconv.Atoi(req.FormValue("redirects"))
	var keepalive = req.FormValue("keepalive") != ""
	var rates = req.Form["rate"]
	var durations = req.Form["duration"]
	var comment = req.FormValue("comment")
	var periods = []RatePeriod{}
	for i, _ := range rates {
		var rate, _ = strconv.Atoi(rates[i])
		var duration, _ = strconv.Atoi(durations[i])
		periods = append(periods, RatePeriod{uint64(rate), uint(duration)})
	}
	job.Workers = uint64(workers)
	job.Timeout = timeout
	job.Periods = periods
	var changed = bson.M{
		"workers":   job.Workers,
		"timeout":   job.Timeout,
		"redirects": redirects,
		"keepalive": keepalive,
		"periods":   job.Periods,
		"lastrunts": time.Now().Unix(),
	}
	var op = bson.M{"$set": changed}
	err = G_MongoDB.C("vegeta_jobs").UpdateId(job.Id, op)
	if err != nil {
		log.Panic(err)
	}
	G_RunningVegetaJobs.Put(job.Id.Hex())
	go AttackVegetaJob(&job, comment)
	r.Redirect("/vegeta/")
}

func DeleteVegetaJob(req *http.Request, r render.Render) {
	var jobId = req.FormValue("job_id")
	G_RunningVegetaJobs.Delete(jobId)
	err := G_MongoDB.C("vegeta_jobs").RemoveId(bson.ObjectIdHex(jobId))
	if err != nil {
		log.Panic(err)
	}
	r.Redirect("/vegeta/")
}

func StopVegetaJob(req *http.Request, r render.Render) {
	// stop vegeta jobs
	var jobId = req.FormValue("job_id")
	if G_RunningVegetaJobs.Exists(jobId) {
		G_StoppingVegetaJobs.Put(jobId)
	}
	r.Redirect(req.Referer())
}

func GetVegetaLogs(req *http.Request, r render.Render) {
	var jobId = req.FormValue("job_id")
	var page = req.FormValue("p")
	var logs []AttackVegetaLog
	var condition = bson.M{}
	if jobId != "" {
		condition = bson.M{"jobid": jobId}
	} else {
		condition = nil
	}
	total, err := G_MongoDB.C("vegeta_logs").Find(condition).Count()
	if err != nil {
		log.Panic(err)
	}
	var pager = NewPager(20, total)
	pager.CurrentPage, err = strconv.Atoi(page)
	pager.UrlPattern = fmt.Sprintf("/vegeta/logs?&p=%%d&job_id=%s", jobId)
	err = G_MongoDB.C("vegeta_logs").Find(condition).Skip(pager.Offset()).Sort("-startts").Limit(pager.Limit()).All(&logs)
	if err != nil {
		log.Panic(err)
	}
	var context = make(map[string]interface{})
	context["logs"] = logs
	context["jobId"] = jobId
	context["pager"] = pager
	RenderTemplate(r, "vegeta_logs", context)
}

func DeleteVegetaLog(req *http.Request, r render.Render) {
	var logId = bson.ObjectIdHex(req.FormValue("log_id"))
	err := G_MongoDB.C("vegeta_logs").RemoveId(logId)
	if err != nil {
		log.Panic(err)
	}
	r.Redirect(req.Referer())
}

func GetVegetaMetrics(req *http.Request, r render.Render) {
	var lg AttackVegetaLog
	var lgId = bson.ObjectIdHex(req.FormValue("log_id"))
	err := G_MongoDB.C("vegeta_logs").FindId(lgId).One(&lg)
	if err != nil {
		log.Panic(err)
	}
	var context = make(map[string]interface{})
	context["log"] = &lg
	RenderTemplate(r, "vegeta_metrics", context)
}

type AttackVegetaLog struct {
	Id          bson.ObjectId `json:"id"        bson:"_id,omitempty"`
	JobId       string
	JobName     string
	JobUrl      string
	JobDetail   *VegetaJob
	Comment     string
	State       string
	MetricsList []*vegeta.Metrics
	StartTs     int64
	EndTs       int64
}

func (log *AttackVegetaLog) IsRunning() bool {
	return log.State == "Running"
}

func (log *AttackVegetaLog) LatencyMetrics() string {
	var buffer bytes.Buffer
	var startTime = 0.0
	for _, metrics := range log.MetricsList {
		var latency = metrics.Latencies.Mean.Seconds() * 1000
		buffer.WriteString(fmt.Sprintf("%v,%v\n", startTime, latency))
		startTime += metrics.Duration.Seconds()
		buffer.WriteString(fmt.Sprintf("%v,%v\n", startTime, latency))
	}
	return buffer.String()
}

func (log *AttackVegetaLog) RateMetrics() string {
	var buffer bytes.Buffer
	var startTime = 0.0
	for _, metrics := range log.MetricsList {
		var rate = metrics.Rate
		buffer.WriteString(fmt.Sprintf("%v,%v\n", startTime, rate))
		startTime += metrics.Duration.Seconds()
		buffer.WriteString(fmt.Sprintf("%v,%v\n", startTime, rate))
	}
	return buffer.String()
}

func (log *AttackVegetaLog) RateLatencyMetrics() string {
	var buffer bytes.Buffer
	for _, metrics := range log.MetricsList {
		var rate = metrics.Rate
		var latency = metrics.Latencies.Mean.Seconds() * 1000
		buffer.WriteString(fmt.Sprintf("%v,%v\n", rate, latency))
	}
	return buffer.String()
}

func (log *AttackVegetaLog) StatusCodesList() map[string]bool {
	var codeList = make(map[string]bool)
	for _, metrics := range log.MetricsList {
		for code, _ := range metrics.StatusCodes {
			codeList[code] = true
		}
	}
	return codeList
}

func (log *AttackVegetaLog) StatusCodesMetrics() string {
	var buffer bytes.Buffer
	var startTime = 0.0
	var codeList = log.StatusCodesList()
	for _, metrics := range log.MetricsList {
		buffer.WriteString(fmt.Sprintf("%v", startTime))
		for code, _ := range codeList {
			var count, ok = metrics.StatusCodes[code]
			if !ok {
				count = 0
			}
			buffer.WriteString(fmt.Sprintf(",%v", count))
		}
		buffer.WriteString("\n")
		startTime += metrics.Duration.Seconds()
		buffer.WriteString(fmt.Sprintf("%v", startTime))
		for code, _ := range codeList {
			var count, ok = metrics.StatusCodes[code]
			if !ok {
				count = 0
			}
			buffer.WriteString(fmt.Sprintf(",%v", count))
		}
		buffer.WriteString("\n")
	}
	return buffer.String()
}

func AttackVegetaJob(job *VegetaJob, comment string) {
	// start attacking target servers
	var log = LogAttackVegetaStart(job, comment)
	var metricsList []*vegeta.Metrics
	attacker := vegeta.NewAttacker(
		vegeta.Timeout(time.Duration(job.Timeout)*time.Second),
		vegeta.Workers(job.Workers),
		vegeta.KeepAlive(job.Keepalive),
		vegeta.Redirects(job.Redirects))
	targeter := NewRandomVegetaTargeter(job)
	for _, period := range job.Periods {
		var metrics vegeta.Metrics
		var rate = period.Rate
		var duration = time.Duration(period.Duration) * time.Second
		UpdateJobCurrentRate(job, rate)
		for res := range attacker.Attack(targeter, rate, duration) {
			metrics.Add(res)
		}
		metrics.Close()
		metricsList = append(metricsList, &metrics)
		if G_StoppingVegetaJobs.Exists(job.Id.Hex()) {
			G_StoppingVegetaJobs.Delete(job.Id.Hex())
			break
		}
	}
	G_RunningVegetaJobs.Delete(job.Id.Hex())
	UpdateJobCurrentRate(job, 0)
	LogAttackVegetaEnd(log, metricsList)
}

func UpdateJobCurrentRate(job *VegetaJob, rate uint64) {
	// realtime update job's current rate for displaying
	var op = bson.M{"$set": bson.M{"currentrate": rate}}
	err := G_MongoDB.C("vegeta_jobs").UpdateId(job.Id, op)
	if err != nil {
		log.Panic(err)
	}
}

func LogAttackVegetaStart(job *VegetaJob, comment string) *AttackVegetaLog {
	// record attack log before job starts
	var lg = AttackVegetaLog{
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
	err := G_MongoDB.C("vegeta_logs").Insert(&lg)
	if err != nil {
		log.Panic(err)
	}
	return &lg
}

func LogAttackVegetaEnd(lg *AttackVegetaLog, metricsList []*vegeta.Metrics) {
	// record attack reports after job finished
	var op = bson.M{"$set": bson.M{"metricslist": metricsList, "state": "End", "endts": time.Now().Unix()}}
	err := G_MongoDB.C("vegeta_logs").UpdateId(lg.Id, op)
	if err != nil {
		log.Panic(err)
	}
}

func NewRandomVegetaTargeter(job *VegetaJob) vegeta.Targeter {
	// Generate http requests for vegeta job's attack
	var targets []vegeta.Target
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
			var body []byte
			if job.Jsonified {
				body = []byte(jsonData)
			} else {
				body = BodyBytes(data)
			}
			var target = vegeta.Target{
				Method: job.Method,
				URL:    Urlcat(host, job.Url, param),
				Body:   body,
				Header: header,
			}
			targets = append(targets, target)
		}
	}
	return vegeta.NewStaticTargeter(targets...)
}
