package main

import (
	"errors"
	"github.com/gin-gonic/gin"

	log "gopkg.in/inconshreveable/log15.v2"
)

func addBenchmark(c *gin.Context) {
	var b Benchmark
	if err := c.BindJSON(&b); err != nil {
		c.IndentedJSON(400, gin.H{"message": err.Error()})
		log.Error("error adding benchmark", "err", err)
		return
	}
	err := ds.addBenchmark(&b)
	if err != nil {
		c.AbortWithError(500, err)
	}
}

func getBuilds(c *gin.Context) {
	builds, err := ds.getBuilds()
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.IndentedJSON(200, builds)
}

func compare(c *gin.Context) {
	build1 := c.Param("build1")
	build2 := c.Param("build2")
	if build1 == "" || build2 == "" {
		c.AbortWithError(400, errors.New("missing arguments"))
		return
	}
	comparison, err := ds.compare(build1, build2)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.IndentedJSON(200, comparison)
}

func getReport(c *gin.Context) {
	build := c.Param("build")
	if build == "" {
		c.AbortWithError(400, errors.New("missing arguments"))
		return
	}
	prevBuild, reports, err := ds.getReport(build)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	renderReport(c.Writer, prevBuild, build, reports)
}

func getHistory(c *gin.Context) {
	type payload struct {
		Component string `json:"component"`
		Metric    string `json:"metric"`
		TestCase  string `json:"testCase"`
	}
	var p payload
	if err := c.BindJSON(&p); err != nil {
		c.IndentedJSON(400, gin.H{"message": err.Error()})
		return
	}

	if p.Component == "" || p.TestCase == "" || p.Metric == "" {
		c.AbortWithError(400, errors.New("missing arguments"))
		return
	}
	history, err := ds.getHistory(p.Component, p.TestCase, p.Metric)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.IndentedJSON(200, history)
}

func httpEngine() *gin.Engine {
	router := gin.Default()

	router.StaticFile("/", "./app/index.html")
	router.Static("/static", "./app")

	rg := router.Group("/api/v1")

	rg.POST("benchmarks", addBenchmark)

	rg.GET("builds", getBuilds)

	rg.GET("comparison/:build1/:build2", compare)

	rg.GET("report/:build", getReport)

	rg.POST("history", getHistory)

	return router
}
