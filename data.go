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
	Title     string   `json:"title"`
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
	docId := hash(b.Component, b.Title, b.Metric, b.Build)

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
		return &builds, err
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
	Threshold int      `json:"threshold"`
	Title     string   `json:"title"`
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
			"ARRAY_AGG({\"metric\": q.metric, \"title\": q.title, \"threshold\": q.threshold, \"results\": q.results}) AS metrics " +
			"FROM ( " +
			"SELECT component, title, metric, threshold, " +
			"ARRAY_AGG({\"build\": `build`, \"snapshots\": snapshots, \"value\": `value`}) AS results " +
			"FROM daily " +
			"WHERE `build` = $1 OR `build` = $2 " +
			"GROUP BY component, title, metric, threshold) AS q " +
			"GROUP BY q.component " +
			"HAVING COUNT(*) > 0 " +
			"ORDER BY q.component;")
	params := []interface{}{build1, build2}

	rows, err := ds.bucket.ExecuteN1qlQuery(query, params)
	if err != nil {
		return &comparison, err
	}

	var row Comparison
	for rows.Next(&row) {
		comparison = append(comparison, row)
		row = Comparison{}
	}

	return &comparison, nil
}

type BuildValuePair struct {
	Build string  `json:"build"`
	Value float64 `json:"value"`
}

func (d *dataStore) getHistory(component, title, metric string) (*[][]interface{}, error) {
	history := [][]interface{}{}

	query := gocb.NewN1qlQuery(
		"SELECT `build`, `value` " +
			"FROM daily " +
			"WHERE component = $1 AND title = $2 AND metric = $3 " +
			"ORDER BY `build`;")

	params := []interface{}{component, title, metric}
	rows, err := ds.bucket.ExecuteN1qlQuery(query, params)
	if err != nil {
		return &history, err
	}

	var row Result
	for rows.Next(&row) {
		history = append(history, []interface{}{row.Build, row.Value})
		row = Result{}
	}

	return &history, nil
}
