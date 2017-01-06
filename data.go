package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"

	"gopkg.in/couchbase/gocb.v1"
	log "gopkg.in/inconshreveable/log15.v2"
)

type Benchmark struct {
	Build     string   `json:"build"`
	BuildURL  string   `json:"buildURL"`
	Component string   `json:"component"`
	DateTime  string   `json:"dateTime"`
	Metric    string   `json:"metric"`
	Snapshots []string `json:"snapshots"`
	Threshold int      `json:"threshold"`
	TestCase  string   `json:"testCase"`
	Value     float64  `json:"value"`
}

type dataStore struct {
	bucket *gocb.Bucket
}

func newDataStore() *dataStore {
	hostname := os.Getenv("CB_HOST")
	if hostname == "" {
		log.Error("missing Couchbase Server hostname")
		os.Exit(1)
	}
	password := os.Getenv("CB_PASS")
	if password == "" {
		log.Error("missing password")
		os.Exit(1)
	}

	connSpecStr := fmt.Sprintf("couchbase://%s", hostname)
	cluster, err := gocb.Connect(connSpecStr)
	if err != nil {
		log.Error("failed to connect to Couchbase Server", "err", err)
		os.Exit(1)
	}

	bucket, err := cluster.OpenBucket("daily", password)
	if err != nil {
		log.Error("failed to connect to bucket", "err", err)
		os.Exit(1)
	}

	return &dataStore{bucket}
}

func hash(strings ...string) string {
	h := md5.New()
	for _, s := range strings {
		h.Write([]byte(s))
	}
	return hex.EncodeToString(h.Sum(nil))

}

func (d *dataStore) addBenchmark(b *Benchmark) error {
	docId := hash(b.Component, b.TestCase, b.Metric, b.Build)

	_, err := d.bucket.Upsert(docId, b, 0)
	if err != nil {
		log.Error("failed to insert metric", "err", err)
	}
	return err

}

type Build struct {
	Build string `json:"build"`
}

func (d *dataStore) getBuilds() (*[]string, error) {
	var builds []string

	query := gocb.NewN1qlQuery(
		"SELECT DISTINCT `build` " +
			"FROM daily " +
			"ORDER BY `build`;")

	rows, err := ds.bucket.ExecuteN1qlQuery(query, []interface{}{})
	if err != nil {
		return nil, err
	}

	var row Build
	for rows.Next(&row) {
		builds = append(builds, row.Build)
	}
	return &builds, nil
}

type Result struct {
	Build     string   `json:"build"`
	Snapshots []string `json:"snapshots"`
	Value     float64  `json:"value"`
}

type Metric struct {
	Metric    string   `json:"metric"`
	TestCase  string   `json:"_testCase"`
	Threshold int      `json:"threshold"`
	Results   []Result `json:"results"`
}

type Comparison struct {
	Component string   `json:"component"`
	Metrics   []Metric `json:"metrics"`
}

func (d *dataStore) compare(build1, build2 string) (*[]Comparison, error) {
	comparison := []Comparison{}

	query := gocb.NewN1qlQuery(
		"SELECT q.component, " +
			"ARRAY_AGG({\"metric\": q.metric, \"_testCase\": q.testCase, \"threshold\": q.threshold, \"results\": q.results}) AS metrics " +
			"FROM ( " +
			"SELECT component, metric, testCase, threshold, " +
			"ARRAY_AGG({\"build\": `build`, \"snapshots\": snapshots, \"value\": `value`}) AS results " +
			"FROM daily " +
			"WHERE `build` = $1 OR `build` = $2 " +
			"GROUP BY component, metric, testCase, threshold) AS q " +
			"GROUP BY q.component " +
			"HAVING COUNT(*) > 0 " +
			"ORDER BY q.component;")
	params := []interface{}{build1, build2}

	rows, err := ds.bucket.ExecuteN1qlQuery(query, params)
	if err != nil {
		return nil, err
	}

	var row Comparison
	for rows.Next(&row) {
		comparison = append(comparison, row)
		row = Comparison{}
	}

	return &comparison, nil
}

func (d *dataStore) findPrevBuild(build string) (string, error) {
	previousBuild := ""

	query := gocb.NewN1qlQuery(
		"SELECT DISTINCT `build` " +
			"FROM daily " +
			"WHERE `build` < $1 " +
			"ORDER BY `build` DESC " +
			"LIMIT 1;")

	params := []interface{}{build}
	rows, err := ds.bucket.ExecuteN1qlQuery(query, params)
	if err != nil {
		return previousBuild, err
	}

	var row Result
	err = rows.One(&row)
	return row.Build, err
}

type Report struct {
	Component string   `json:"component"`
	Metric    string   `json:"metric"`
	TestCase  string   `json:"testCase"`
	Threshold int      `json:"threshold"`
	Results   []Result `json:"results"`
}

func (d *dataStore) getReport(build string) (string, []Report, error) {
	reports := []Report{}

	prevBuild, err := ds.findPrevBuild(build)
	if err != nil {
		return prevBuild, reports, err
	}

	query := gocb.NewN1qlQuery(
		"SELECT component, testCase, metric, threshold, ARRAY_AGG({\"build\": `build`, \"value\": `value`}) AS results " +
			"FROM daily " +
			"WHERE `build` = $1 OR `build` = $2 " +
			"GROUP BY component, testCase, metric, threshold " +
			"HAVING COUNT(*) > 1 " +
			"ORDER BY component, testCase;")

	params := []interface{}{prevBuild, build}
	rows, err := ds.bucket.ExecuteN1qlQuery(query, params)
	if err != nil {
		return prevBuild, reports, err
	}

	var row Report
	for rows.Next(&row) {
		reports = append(reports, row)
		row = Report{}
	}

	return prevBuild, reports, err
}

func (d *dataStore) getHistory(component, testCase, metric string) (*[][]interface{}, error) {
	history := [][]interface{}{}

	query := gocb.NewN1qlQuery(
		"SELECT `build`, `value` " +
			"FROM daily " +
			"WHERE component = $1 AND testCase = $2 AND metric = $3 " +
			"ORDER BY `build`;")

	params := []interface{}{component, testCase, metric}
	rows, err := ds.bucket.ExecuteN1qlQuery(query, params)
	if err != nil {
		return nil, err
	}

	var row Result
	for rows.Next(&row) {
		history = append(history, []interface{}{row.Build, row.Value})
		row = Result{}
	}

	return &history, nil
}
