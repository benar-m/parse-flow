package internal

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// consume from DB chans and also trigger snapshots
func (a *App) StartDbWriter() {
	db, err := sql.Open("sqlite3", "./logs.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	err = a.initTables(db)
	if err != nil {
		log.Fatal("Failed to create tables:", err)
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
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME,
		log_data TEXT, 
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	snapshotTable := `
	CREATE TABLE IF NOT EXISTS metric_snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		snapshot_time DATETIME,
		metrics_data TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
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
		"INSERT INTO raw_logs (timestamp, log_data) VALUES (?, ?)",
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

	stmt, err := tx.Prepare("INSERT INTO raw_logs (timestamp, log_data) VALUES (?, ?)")
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
		"INSERT INTO metric_snapshots (snapshot_time, metrics_data) VALUES (?, ?)",
		snapshot.Timestamp,
		string(snapshotJSON),
	)
	return err
}
