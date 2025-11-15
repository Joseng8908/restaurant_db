package model

import (
	"time"
)

type User struct {
	// user_id INTEGER PRIMARY KEY
	UserID int64 `db:"user_id"`

	// username TEXT NOT NULL UNIQUE
	Username string `db:"username"`

	// review_count INTEGER NOT NULL DEFAULT 0
	ReviewCount int64 `db:"review_count"`

	// reliability_score REAL NOT NULL DEFAULT .5
	ReliabilityScore float64 `db:"reliability_score"`

	// bias_count INTEGER NOT NULL DEFAULT 0
	BiasCount int64 `db:"bias_count"`

	// created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%d %H:%M:%S', 'now'))
	CreatedAt time.Time `db:"created_at"`
}
