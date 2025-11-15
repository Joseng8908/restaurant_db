package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"restaurant_db/internal/model"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByID(ctx context.Context, userID int64) (*model.User, error)
	UpdateReliabilityScore(ctx context.Context, userID int64, newScore float64, newReviewCount int64, newBiasCount int64) error
}

type UserRepoImpl struct {
	DB *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &UserRepoImpl{DB: db}
}

// Create: 새로운 유저를 User 테이블에 추가하고 ID를 할당합니다.
func (r *UserRepoImpl) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO User (username) 
		VALUES (?)` // 나머지 필드는 DDL에서 기본값(DEFAULT)을 사용

	result, err := r.DB.ExecContext(ctx, query, user.Username)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	lastID, err := result.LastInsertId()
	if err == nil {
		user.UserID = lastID
	}
	return nil
}

// FindByID: user_id를 기반으로 유저 정보를 조회합니다.
func (r *UserRepoImpl) FindByID(ctx context.Context, userID int64) (*model.User, error) {
	user := &model.User{}

	query := `
		SELECT 
			user_id, username, review_count, reliability_score, bias_count, created_at
		FROM User 
		WHERE user_id = ?`

	row := r.DB.QueryRowContext(ctx, query, userID)

	var createdAtStr string

	err := row.Scan(
		&user.UserID,
		&user.Username,
		&user.ReviewCount,
		&user.ReliabilityScore,
		&user.BiasCount,
		&createdAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // 유저 없음
		}
		return nil, fmt.Errorf("failed to find user by ID: %w", err)
	}

	const sqliteTimeFormat = "2006-01-02 15:04:05"
	user.CreatedAt, err = time.Parse(sqliteTimeFormat, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user created_at: %w", err)
	}

	return user, nil
}

// UpdateReliabilityScore: 유저의 신뢰도 점수와 카운트 정보를 업데이트합니다. (Worker가 사용)
func (r *UserRepoImpl) UpdateReliabilityScore(
	ctx context.Context,
	userID int64,
	newScore float64,
	newReviewCount int64,
	newBiasCount int64,
) error {
	query := `
		UPDATE User 
		SET 
			reliability_score = ?,
			review_count = ?,
			bias_count = ?
		WHERE user_id = ?`

	_, err := r.DB.ExecContext(
		ctx,
		query,
		newScore,
		newReviewCount,
		newBiasCount,
		userID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user reliability score (ID: %d): %w", userID, err)
	}

	return nil
}
