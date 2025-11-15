package model

import (
	"time"
)

type BufferLog struct {
	// log_id INTEGER PRIMARY KEY
	LogID int64 `db:"log_id"`

	// transaction_type TEXT NOT NULL -- INSERT, UPDATE, DELETE
	TransactionType string `db:"transaction_type"`

	// taget_table TEXT NOT NULL -- 어느 테이블에 적용할지 결정하는 속성
	TargetTable string `db:"taget_table"` // DDL의 철자(taget_table)를 그대로 따름

	// payload TEXT NOT NULL -- 페이로드는 json (Go에서는 string으로 처리)
	Payload string `db:"payload"`

	// target_record_id INTEGER -- 실제 transaction_type에 따라 적용시킬 레코드의 id
	TargetRecordID int64 `db:"target_record_id"` // NULL 허용되지만, 구조체에서는 int64 포인터 대신 0으로 처리하거나 sql.NullInt64 사용 가능. 일단 단순화하여 int64로 정의

	// log_updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now'))
	LogUpdatedAt time.Time `db:"log_updated_at"`

	// is_committed INTEGER NOT NULL DEFAULT 0 -- splite에서는 boolean을 못쓴다네요..?
	IsCommitted int64 `db:"is_committed"` // SQLite의 INTEGER(0 또는 1)에 맞춰 int64로 정의
}
