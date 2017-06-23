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
			"WHERE `build` IS NOT MISSING " +
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

type Range struct {
	Max string `json:"max"`
	Min string `json:"min"`
}

func (d *dataStore) getRange(testCase string) (*Range, error) {
	query := gocb.NewN1qlQuery(
		"SELECT MAX(`build`) AS `max`, MIN(`build`) AS `min` " +
			"FROM daily " +
			"WHERE testCase = $1;")

	params := []interface{}{testCase}
	rows, err := ds.bucket.ExecuteN1qlQuery(query, params)
	if err != nil {
		return nil, err
	}

	var row Range
	err = rows.One(&row)
	return &row, err
}

func (d *dataStore) evalIncomplete(metric *Metric, build1, build2 string) string {
	buildRange, err := d.getRange(metric.TestCase)
	if err != nil {
		return "Incomplete"
	}

	var missingBuild string
	if metric.Results[0].Build == build1 {
		missingBuild = build2
	} else {
		missingBuild = build1
	}

	if missingBuild < buildRange.Min {
		return "New Feature"
	}
	return "Incomplete"
}

func (d *dataStore) evalComplete(metric *Metric) string {
	delta := 100 * (metric.Results[1].Value/metric.Results[0].Value - 1)

	if metric.Threshold < 0 && delta < metric.Threshold {
		return "Failed"
	}
	if metric.Threshold > 0 && delta > metric.Threshold {
		return "Failed"
	}

	return "Passed"
}

func (d *dataStore) evalStatus(build1, build2 string, comparison *[]Comparison) {
	for i, item := range *comparison {
		for j, metric := range item.Metrics {
			var status string
			if len(metric.Results) == 1 {
				status = d.evalIncomplete(&metric, build1, build2)
			} else {
				status = d.evalComplete(&metric)
			}
			(*comparison)[i].Metrics[j].Status = status
		}
	}
}

type Result struct {
	Build          string      `json:"build"`
	Snapshots      []string    `json:"snapshots"`
	Value          float64     `json:"value"`
	Annotation     interface{} `json:"annotation"`
	AnnotationText interface{} `json:"annotationText"`
}

type Metric struct {
	Metric    string   `json:"metric"`
	TestCase  string   `json:"_testCase"`
	Threshold float64  `json:"threshold"`
	Results   []Result `json:"results"`
	Status    string   `json:"status"`
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

func (d *dataStore) calcMovingAverage(build, testCase string) (float64, error) {
	query := gocb.NewN1qlQuery(
		"SELECT AVG(v.`value`) AS `value` " +
			"FROM (" +
			"SELECT `value` " +
			"FROM daily " +
			"USE INDEX (daily_ma) " +
			"WHERE `build` < $1 " +
			"AND testCase = $2 " +
			"ORDER BY `build` DESC " +
			"LIMIT 3) v;")

	params := []interface{}{build, testCase}
	rows, err := ds.bucket.ExecuteN1qlQuery(query, params)
	if err != nil {
		return 0, err
	}

	var row Result
	err = rows.One(&row)
	return row.Value, err
}

type Report struct {
	Component     string  `json:"component"`
	Metric        string  `json:"metric"`
	MovingAverage float64 `json:"movingAverage"`
	TestCase      string  `json:"testCase"`
	Threshold     int     `json:"threshold"`
	Value         float64 `json:"value"`
}

func (d *dataStore) getReport(build string) ([]Report, error) {
	reports := []Report{}

	query := gocb.NewN1qlQuery(
		"SELECT component, testCase, metric, threshold, `value` " +
			"FROM daily " +
			"WHERE `build` = $1 " +
			"ORDER BY component, testCase;")

	params := []interface{}{build}
	rows, err := ds.bucket.ExecuteN1qlQuery(query, params)
	if err != nil {
		return reports, err
	}

	var row Report
	for rows.Next(&row) {
		ma, err := d.calcMovingAverage(build, row.TestCase)
		if err == nil && ma > 0 {
			row.MovingAverage = ma
			reports = append(reports, row)
		}
		row = Report{}
	}

	return reports, err
}

type History struct {
	Build    string  `json:"build"`
	BuildURL string  `json:"buildURL"`
	Value    float64 `json:"value"`
}

func (d *dataStore) getHistory(component, testCase, metric string) (*[]History, error) {
	history := []History{}

	query := gocb.NewN1qlQuery(
		"SELECT `build`, buildURL, `value` " +
			"FROM daily " +
			"USE INDEX (daily_view) " +
			"WHERE component = $1 AND testCase = $2 AND metric = $3 " +
			"ORDER BY `build` DESC;")

	params := []interface{}{component, testCase, metric}
	rows, err := ds.bucket.ExecuteN1qlQuery(query, params)
	if err != nil {
		return nil, err
	}

	var row History
	for rows.Next(&row) {
		history = append(history, row)
	}

	return &history, nil
}

func (d *dataStore) getTimeline(component, testCase, metric string) (*[][]interface{}, error) {
	timeline := [][]interface{}{}

	query := gocb.NewN1qlQuery(
		"SELECT `build`, `value`, annotation, annotationText " +
			"FROM daily " +
			"USE INDEX (daily_view) " +
			"WHERE component = $1 AND testCase = $2 AND metric = $3 " +
			"ORDER BY `build`;")

	params := []interface{}{component, testCase, metric}
	rows, err := ds.bucket.ExecuteN1qlQuery(query, params)
	if err != nil {
		return nil, err
	}

	var row Result
	for rows.Next(&row) {
		timeline = append(timeline, []interface{}{row.Build, row.Value, row.Annotation, row.AnnotationText})
		row = Result{}
	}

	return &timeline, nil
}
