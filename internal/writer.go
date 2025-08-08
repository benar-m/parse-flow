package internal

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

	_ "github.com/lib/pq"
)

// consume from DB chans and also trigger snapshots
func (a *App) StartDbWriter() {
	dbURL := "postgres://localhost/parseflow?sslmode=disable" // fallback default
	if a.Config != nil && a.Config.DatabaseURL != "" {
		dbURL = a.Config.DatabaseURL
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	err = a.initTables(db)
	if err != nil {
		log.Fatal("Failed to create tables:", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	const batchSize = 100
	const flushInterval = 5 * time.Second

	batch := make([]*ParsedLog, 0, batchSize)

	snapshotTicker := time.NewTicker(1 * time.Minute)
	defer snapshotTicker.Stop()

	flushTicker := time.NewTicker(flushInterval)
	defer flushTicker.Stop()

	for {
		select {
		case logEntry := <-a.DbRawWriteChan:
			batch = append(batch, logEntry)

			if len(batch) >= batchSize {
				err := a.writeBatchToDb(db, batch)
				if err != nil {
					log.Printf("Failed to write batch to DB: %v", err)
				}
				batch = batch[:0]
			}

		case <-flushTicker.C:
			if len(batch) > 0 {
				err := a.writeBatchToDb(db, batch)
				if err != nil {
					log.Printf("Failed to flush batch to DB: %v", err)
				}
				batch = batch[:0]
			}

		case <-snapshotTicker.C:
			if len(batch) > 0 {
				err := a.writeBatchToDb(db, batch)
				if err != nil {
					log.Printf("Failed to flush batch before snapshot: %v", err)
				}
				batch = batch[:0]
			}

			snapshot := a.GetMetricsSnapshot()
			err := a.writeSnapshotToDb(db, snapshot)
			if err != nil {
				log.Printf("Failed to write snapshot to DB: %v", err)
			}
		}
	}
}

func (a *App) initTables(db *sql.DB) error {
	logTable := `
	CREATE TABLE IF NOT EXISTS raw_logs (
		id SERIAL PRIMARY KEY,
		timestamp TIMESTAMPTZ,
		log_data JSONB, 
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);`

	snapshotTable := `
	CREATE TABLE IF NOT EXISTS metric_snapshots (
		id SERIAL PRIMARY KEY,
		snapshot_time TIMESTAMPTZ,
		metrics_data JSONB,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(logTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(snapshotTable)
	return err
}

func (a *App) writeLogToDb(db *sql.DB, logEntry *ParsedLog) error {
	logJSON, err := json.Marshal(logEntry)
	if err != nil {
		return err
	}
	_, err = db.Exec(
		"INSERT INTO raw_logs (timestamp, log_data) VALUES ($1, $2)",
		logEntry.Time,
		string(logJSON),
	)
	return err
}

func (a *App) writeBatchToDb(db *sql.DB, batch []*ParsedLog) error {
	if len(batch) == 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO raw_logs (timestamp, log_data) VALUES ($1, $2)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, logEntry := range batch {
		logJSON, err := json.Marshal(logEntry)
		if err != nil {
			return err
		}

		_, err = stmt.Exec(logEntry.Time, string(logJSON))
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (a *App) writeSnapshotToDb(db *sql.DB, snapshot *Metric) error {
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}

	_, err = db.Exec(
		"INSERT INTO metric_snapshots (snapshot_time, metrics_data) VALUES ($1, $2)",
		snapshot.Timestamp,
		string(snapshotJSON),
	)
	return err
}

// get metric snapshots
// percentage: 100 = 1 month, 50 = 2 weeks, 25 = 1 week, 0.3 = 1 day, etc.
func (a *App) GetHistoricalSnapshots(percentage float64) ([]*Metric, error) {
	dbURL := "postgres://localhost/parseflow?sslmode=disable" // fallback default
	if a.Config != nil && a.Config.DatabaseURL != "" {
		dbURL = a.Config.DatabaseURL
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	maxDuration := 30 * 24 * time.Hour // 1 month
	queryDuration := time.Duration(float64(maxDuration) * percentage / 100.0)
	if queryDuration < time.Hour {
		queryDuration = time.Hour
	}

	startTime := time.Now().Add(-queryDuration)

	query := `
		SELECT snapshot_time, metrics_data 
		FROM metric_snapshots 
		WHERE snapshot_time >= $1 
		ORDER BY snapshot_time ASC
	`

	rows, err := db.Query(query, startTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []*Metric
	for rows.Next() {
		var snapshotTime time.Time
		var metricsData string

		err := rows.Scan(&snapshotTime, &metricsData)
		if err != nil {
			return nil, err
		}

		var metric Metric
		err = json.Unmarshal([]byte(metricsData), &metric)
		if err != nil {
			return nil, err
		}

		snapshots = append(snapshots, &metric)
	}

	return snapshots, rows.Err()
}
