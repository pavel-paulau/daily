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
	ds.evalStatus(build1, build2, comparison)
	c.IndentedJSON(200, comparison)
}

func getReport(c *gin.Context) {
	build := c.Param("build")
	if build == "" {
		c.AbortWithError(400, errors.New("missing arguments"))
		return
	}
	reports, err := ds.getReports(build)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	testCases, err := ds.getAllCases()
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	renderReport(c.Writer, reports, testCases)
}

func getHistory(c *gin.Context) {
	component := c.Query("component")
	testCase := c.Query("testCase")
	metric := c.Query("metric")

	if component == "" || testCase == "" || metric == "" {
		c.AbortWithError(400, errors.New("missing arguments"))
		return
	}
	history, err := ds.getHistory(component, testCase, metric)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.IndentedJSON(200, history)
}

func getTimeline(c *gin.Context) {
	component := c.Query("component")
	testCase := c.Query("testCase")
	metric := c.Query("metric")

	if component == "" || testCase == "" || metric == "" {
		c.AbortWithError(400, errors.New("missing arguments"))
		return
	}
	history, err := ds.getTimeline(component, testCase, metric)
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

	rg.GET("history", getHistory)

	rg.GET("timeline", getTimeline)

	return router
}
