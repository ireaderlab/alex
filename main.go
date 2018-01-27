package main

import (
	"fmt"
	"html/template"
	"os"

	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
)

func main() {
	// load configuration file
	LoadConfig()
	// initialize Global Variables
	InitGlobals()

	// build martini
	m := martini.Classic()
	staticOptions := martini.StaticOptions{Prefix: "static"}
	m.Use(martini.Static("static", staticOptions))
	var renderOptions = render.Options{
		Layout: "layout",
		Funcs:  []template.FuncMap{Pipelines}}
	m.Use(render.Renderer(renderOptions))
	m.Use(martini.Logger())
	m.Get("/", func(r render.Render) {
		r.Redirect("/vegeta/")
	})
	m.Group("/api", func(r martini.Router) {
		r.Get("/system", GetSystemStatus)
		r.Get("/vegeta/state", GetVegetaJobState)
		r.Get("/boom/state", GetBoomJobState)
		r.Post("/param/test", TestParam)
	})
	m.Group("/vegeta", func(r martini.Router) {
		r.Get("/", GetVegetaJobs)
		r.Post("/create", CreateVegetaJob)
		r.Get("/edit", EditVegetaJobPage)
		r.Post("/edit", EditVegetaJob)
		r.Get("/delete", DeleteVegetaJob)
		r.Get("/run", RunVegetaJobPage)
		r.Post("/run", RunVegetaJob)
		r.Get("/stop", StopVegetaJob)
		r.Get("/logs", GetVegetaLogs)
		r.Get("/log/delete", DeleteVegetaLog)
		r.Get("/metrics", GetVegetaMetrics)
	})
	m.Group("/boom", func(r martini.Router) {
		r.Get("/", GetBoomJobs)
		r.Post("/create", CreateBoomJob)
		r.Get("/edit", EditBoomJobPage)
		r.Post("/edit", EditBoomJob)
		r.Get("/delete", DeleteBoomJob)
		r.Get("/run", RunBoomJobPage)
		r.Post("/run", RunBoomJob)
		r.Get("/stop", StopBoomJob)
		r.Get("/logs", GetBoomLogs)
		r.Get("/log/delete", DeleteBoomLog)
		r.Get("/metrics", GetBoomMetrics)
	})
	// Let's fly
	os.Setenv("HOST", G_AlexHost)
	os.Setenv("PORT", fmt.Sprintf("%d", G_AlexPort))
	m.Run()
}
