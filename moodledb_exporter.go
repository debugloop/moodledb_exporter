package main

import (
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var (
	listenAddress = flag.String("web.listen-address", ":9720", "Address to listen on for web interface and telemetry.")
	metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	DSN           = flag.String("mysql.dsn", "user:pass@(localhost:3306)/", "Mysql Data Source Name")
	Prefix        = flag.String("mysql.prefix", "db_", "Prefix used for filtering relevant databases (those containing Moodles).")
)

type MoodleDBCollector struct {
	moodleUsers *prometheus.Desc
}

func (c *MoodleDBCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.moodleUsers
}

func (c *MoodleDBCollector) Collect(ch chan<- prometheus.Metric) {
	db, err := sql.Open("mysql", *DSN)
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	// build filtered db list
	res, _ := db.Query("SHOW DATABASES")
	moodledbs := []string{}
	for res.Next() {
		dbName := ""
		res.Scan(&dbName)
		if dbName[0:len(*Prefix)] == *Prefix {
			moodledbs = append(moodledbs, dbName)
		}
	}

	// query each db for their users
	for _, dbName := range moodledbs {
		res, err := db.Query(fmt.Sprintf("SELECT COUNT(*) AS userCount FROM %s.mdl_user WHERE deleted=0", dbName))
		if err != nil {
			continue
		}
		userCount := 0
		for res.Next() { // this will run just once
			res.Scan(&userCount)
		}
		ch <- prometheus.MustNewConstMetric(
			c.moodleUsers,
			prometheus.GaugeValue,
			float64(userCount),
			dbName,
		)
	}
}

func NewMoodleDBCollector() *MoodleDBCollector {
	return &MoodleDBCollector{
		moodleUsers: prometheus.NewDesc("moodle_users_total", "Number of users found in a MoodleDB", []string{"dbname"}, nil),
	}
}

func init() {
	prometheus.MustRegister(NewMoodleDBCollector())
}

func main() {
	flag.Parse()
	http.Handle(*metricsPath, promhttp.Handler())
	http.ListenAndServe(*listenAddress, nil)
}
