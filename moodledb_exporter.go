package main

import (
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"os"
)

var (
	listenAddress = flag.String("web.listen-address", ":9720", "Address to listen on for web interface and telemetry.")
	metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	Prefix        = flag.String("mysql.prefix", "db_", "Prefix used for filtering relevant databases (those containing Moodles).")

	DSN = ""
)

type MoodleDBCollector struct {
	moodleUsers       *prometheus.Desc
	moodleActiveUsers *prometheus.Desc
}

func (c *MoodleDBCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.moodleUsers
}

func queryResult(db *sql.DB, query string, dbName string) (float64, error) {
	// passed query *must* be a simple COUNT(*)
	res, err := db.Query(fmt.Sprintf(query, dbName))
	if err != nil {
		return 0.0, fmt.Errorf("Error running query.")
	}
	resultCount := 0
	for res.Next() { // this should run just once
		res.Scan(&resultCount)
	}
	return float64(resultCount), nil
}

func (c *MoodleDBCollector) Collect(ch chan<- prometheus.Metric) {
	db, err := sql.Open("mysql", DSN)
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	// build filtered db list
	res, err := db.Query("SHOW DATABASES")
	if err != nil {
		fmt.Println("There is a problem with the database.")
		return
	}
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
		result, err := queryResult(db, "SELECT COUNT(*) FROM %s.mdl_user WHERE deleted=0", dbName)
		if err == nil {
			ch <- prometheus.MustNewConstMetric(
				c.moodleUsers,
				prometheus.GaugeValue,
				result,
				dbName,
			)
		}
		result, err = queryResult(db, "SELECT COUNT(*) FROM %s.mdl_user WHERE lastaccess > UNIX_TIMESTAMP(CURRENT_DATE());", dbName)
		if err == nil {
			ch <- prometheus.MustNewConstMetric(
				c.moodleActiveUsers,
				prometheus.GaugeValue,
				result,
				dbName,
			)
		}
	}
}

func NewMoodleDBCollector() *MoodleDBCollector {
	return &MoodleDBCollector{
		moodleUsers:       prometheus.NewDesc("moodle_users_total", "Number of users found in a MoodleDB", []string{"dbname"}, nil),
		moodleActiveUsers: prometheus.NewDesc("moodle_users_active", "Number of users found in a MoodleDB which were active today", []string{"dbname"}, nil),
	}
}

func init() {
	DSN = os.Getenv("DATA_SOURCE_NAME")
	if len(DSN) == 0 {
		fmt.Println("DATA_SOURCE_NAME needs to be set in environment.")
		os.Exit(1)
	} else {
		fmt.Printf("Trying to work with DSN: '%s'\n", DSN)
	}

	prometheus.MustRegister(NewMoodleDBCollector())
}

func main() {
	flag.Parse()

	http.Handle(*metricsPath, promhttp.Handler())
	http.ListenAndServe(*listenAddress, nil)
}
