package main

import (
	"fmt"
	"github.com/martini-contrib/render"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"gopkg.in/mgo.v2/bson"
	"net/http"
)

func GetSystemStatus(req *http.Request, r render.Render) {
	// get system status of benchmark machine
	var result = map[string]string{}
	var l, _ = load.Avg()
	result["load:1"] = fmt.Sprintf("%v", l.Load1)
	result["load:5"] = fmt.Sprintf("%v", l.Load5)
	result["load:15"] = fmt.Sprintf("%v", l.Load15)
	var m, _ = mem.VirtualMemory()
	result["mem:total"] = fmt.Sprintf("total:%vM", m.Total>>20)
	result["mem:free"] = fmt.Sprintf("free:%vM", m.Free>>20)
	result["mem:buffers"] = fmt.Sprintf("buffers:%vM", m.Buffers>>20)
	result["mem:cached"] = fmt.Sprintf("cached:%vM", m.Cached>>20)
	result["mem:wired"] = fmt.Sprintf("wired:%vM", m.Wired>>20)
	result["mem:used"] = fmt.Sprintf("used:%.2f%%", m.UsedPercent)
	r.JSON(200, result)
}

func GetVegetaJobState(req *http.Request, r render.Render) {
	var jobId = req.FormValue("job_id")
	var job VegetaJob
	err := G_MongoDB.C("vegeta_jobs").FindId(bson.ObjectIdHex(jobId)).One(&job)
	var result = map[string]interface{}{}
	if err != nil {
		result["is_running"] = false
		result["current_rate"] = 0
	} else {
		result["is_running"] = job.IsRunning()
		result["current_rate"] = job.CurrentRate
	}
	r.JSON(200, result)
}

func GetBoomJobState(req *http.Request, r render.Render) {
	var jobId = req.FormValue("job_id")
	var job BoomJob
	err := G_MongoDB.C("boom_jobs").FindId(bson.ObjectIdHex(jobId)).One(&job)
	var result = map[string]interface{}{}
	if err != nil {
		result["is_running"] = false
		result["current_concurrency"] = 0
	} else {
		result["is_running"] = job.IsRunning()
		result["current_concurrency"] = job.CurrentConcurrency
	}
	r.JSON(200, result)
}
