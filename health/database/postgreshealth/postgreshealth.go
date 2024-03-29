package postgreshealth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/figment-networks/indexing-engine/metrics"

	"github.com/figment-networks/indexing-engine/health/database"
)

var (
	pingPostgres *metrics.GroupObserver
	sizePostgres *metrics.GroupGauge
)

type PingCheck struct {
	On       time.Time     `json:"on"`
	Duration time.Duration `json:"duration"`
	Status   string        `json:"status"`
	Error    string        `json:"error"`
}

type SizeCheck struct {
	On     time.Time `json:"on"`
	Size   uint64    `json:"size"`
	Status string    `json:"status"`
	Error  string    `json:"error"`
}

type PostgresMonitor struct {
	pc PingCheck
	sc SizeCheck

	db *sql.DB
	l  *zap.Logger
}

func NewPostgresMonitor(db *sql.DB, l *zap.Logger) *PostgresMonitor {
	return &PostgresMonitor{db: db, l: l}
}

func NewPostgresMonitorWithMetrics(db *sql.DB, l *zap.Logger) *PostgresMonitor {
	pingPostgres = database.PingMetric.WithLabels("postgres")
	sizePostgres = database.SizeMetric.WithLabels("postgres")
	return NewPostgresMonitor(db, l)
}

func (m *PostgresMonitor) Probe(ctx context.Context) (err error) {
	if err = m.ping(ctx); err != nil {
		return err
	}

	return m.dbSize(ctx)
}

func (m *PostgresMonitor) Readiness(ctx context.Context) (probetype, redinesstype string, contents interface{}, err error) {
	tCtx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	err = m.ping(tCtx)
	return "db", "postgres", m.pc, err
}

func (m *PostgresMonitor) ping(ctx context.Context) (err error) {
	tCtx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	t := time.Now()
	err = m.db.PingContext(tCtx)
	m.pc = PingCheck{On: t, Status: "ok", Duration: time.Since(t)}
	if err != nil {
		m.pc.Status = "err"
		m.pc.Error = err.Error()
		m.l.Error("[Health][Database][Postgres] Error pinging database", zap.Error(err))
	}

	pingPostgres.Observe(m.pc.Duration.Seconds())
	return err
}

func (m *PostgresMonitor) dbSize(ctx context.Context) error {
	tCtx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	t := time.Now()
	m.sc = SizeCheck{On: t, Status: "ok"}
	row := m.db.QueryRowContext(tCtx, "SELECT pg_database_size( current_database() ) AS raw_size")
	if row == nil {
		m.sc.Status = "err"
		m.l.Error("[Health][Database][Postgres] Error getting database size")
		return errors.New("Error getting database size")
	}
	err := row.Err()
	if err != nil {
		m.sc.Status = "err"
		m.sc.Error = err.Error()
		m.l.Error("[Health][Database][Postgres] Error getting database size", zap.Error(err))
		return err
	}

	if err = row.Scan(&m.sc.Size); err != nil {
		m.pc.Status = "err"
		m.pc.Error = err.Error()
		m.l.Error("[Health][Database][Postgres] Error getting database size", zap.Error(err))
		return err
	}

	sizePostgres.Set(float64(m.sc.Size))
	return nil
}
