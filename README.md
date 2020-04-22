This is a very simple exporter that connects to a local MySQL database on
scrape. It then looks for all databases matching the prefix filter which are
supposed to contain the typical Moodle tables. From these, a count of
non-deleted users is exported for each database.

It is very simple, but it works well enough. It could also serve as a basis for
other SQL query based exporters, if necessary.

| Option   | Explanation |
|----------|-------------|
| `web.listen-address` | same as every other exporter out there, default is port `9720` |
| `web.telemetry-path` | same as every other exporter out there, default is `/metrics`  |
| `mysql.dsn`          | a [MySQL DSN spec](https://github.com/go-sql-driver/mysql#dsn-data-source-name), the default is useless and you will have to set this |
| `mysql.prefix`       | the prefix used to filter databases, as not every database contains a Moodle. We use a `db_` prefix to our `db_customername` scheme, so thats the default |

You'll likely use something like this invocation in the end:

```
./moodledb_exporter --mysql.dsn="exporter:someSecurePass4@(localhost:3306)/" --mysql.prefix="customer_"
```

The output/the metrics will look like:

```
# HELP moodle_users_total Number of users found in a MoodleDB
# TYPE moodle_users_total gauge
moodle_users_total{dbname="db_customer1"} 191
moodle_users_total{dbname="db_customer2"} 10
...
```

Along with the usual `promhttp` stuff.
