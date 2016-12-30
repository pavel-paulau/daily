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
		c.AbortWithError(400, errors.New("bad arguments"))
		return
	}
	comparison, err := ds.compare(build1, build2)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}
	c.IndentedJSON(200, comparison)
}

func httpEngine() *gin.Engine {
	router := gin.Default()

	rg := router.Group("/api/v1")

	rg.POST("benchmarks", addBenchmark)

	rg.GET("builds", getBuilds)

	rg.GET("comparison/:build1/:build2", compare)

	return router
}
