package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"restaurant_db/internal/model"
	"strings"
	"time"
)

type BufferRepository interface {
	// 새로운 쓰기 명령을 Buffer_Log 테이블에 추가
	AddLog(ctx context.Context, log *model.BufferLog) error

	// pending중인 로그 목록을 가져옴, 즉 db에 반영이 아직 되지 않은 로그들을 가져오는 것
	GetPendingLogs(ctx context.Context, limit int) ([]model.BufferLog, error)

	// is_committed = 1로 업데이트 하는 메소드, 커밋 상태를 업데이트하는 함수
	UpdateCommitted(ctx context.Context, logIDs []int64) error
}

type BufferRepoImpl struct {
	DB *sql.DB
}

func NewBufferRepository(db *sql.DB) BufferRepository {
	return &BufferRepoImpl{DB: db}
}

func (r *BufferRepoImpl) AddLog(ctx context.Context, log *model.BufferLog) error {
	if log.IsCommitted != 0 {
		return errors.New("it is already committed")
	}

	query := `
	INSERT INTO Buffer_Log (
	transaction_type, 
	target_table,
	payload, 
	target_record_id
	) VALUES (?, ?, ?, ?)`

	_, err := r.DB.ExecContext(
		ctx,
		query,
		log.TransactionType,
		log.TargetTable,
		log.Payload,
		log.TargetRecordID,
	)
	if err != nil {
		return fmt.Errorf("failed to insert log: %w", err)
	}

	return nil
}

func (r *BufferRepoImpl) GetPendingLogs(ctx context.Context, limit int) ([]model.BufferLog, error) {
	query := `
	SELECT *
	FROM Buffer_Log
	WHERE is_committed = 0
	LIMIT ?`

	rows, err := r.DB.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending logs: %w", err)
	}
	// 메모리 해제 보장
	defer rows.Close()

	var logs []model.BufferLog

	for rows.Next() {
		var log model.BufferLog
		var targetRecordID sql.NullInt64
		var logUpdatedAtStr string

		err := rows.Scan(
			&log.LogID,
			&log.TransactionType,
			&log.TargetTable,
			&log.Payload,
			&targetRecordID,
			&logUpdatedAtStr,
			&log.IsCommitted,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		const sqliteTimeFormat = "2006-01-02 15:04:05"
		parsedTime, err := time.Parse(sqliteTimeFormat, logUpdatedAtStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse log updated at: %w", err)
		}
		log.LogUpdatedAt = parsedTime

		if targetRecordID.Valid {
			log.TargetRecordID = targetRecordID.Int64
		}
		logs = append(logs, log)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate rows: %w", err)
	}

	return logs, nil
}

func (r *BufferRepoImpl) UpdateCommitted(ctx context.Context, logIDs []int64) error {
	if len(logIDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(logIDs))
	for i := range logIDs {
		placeholders[i] = "?"
	}

	query := `
	UPDATE Buffer_Log
	SET is_committed = 1
	WHERE log_id IN (` + strings.Join(placeholders, ",") + `)`

	args := make([]interface{}, len(logIDs))
	for i, id := range logIDs {
		args[i] = id
	}

	_, err := r.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update committed logs: %w", err)
	}

	return nil
}
