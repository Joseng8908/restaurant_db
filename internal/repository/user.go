package repository

import (
	"context"
	"database/sql"
	"restaurant_db/internal/model"
)

type UserRepository interface {
	// 새로운 유저를 User테이블에 추가
	CreateUser(ctx context.Context, user *model.User) error
	// 유저를 User테이블에서 찾음
	FindUserByID(ctx context.Context, userID int64) (*model.User, error)
	// 유저의 id를 사용해서 신뢰도 점수와 카운터 정보들을 업데이트
	UpdateReliabilityScore(ctx context.Context, userID int64, newScore float64, newReviewCount int64, newBiasCount int64) error
}

type UserRepoImpl struct {
	DB *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &UserRepoImpl{DB: db}
}

func (r *UserRepoImpl) CreateUser(ctx context.Context, user *model.User) error {
	return nil
}

func (r *UserRepoImpl) FindUserByID(ctx context.Context, userID int64) (*model.User, error) {
	return nil, nil
}

func (r *UserRepoImpl) UpdateReliabilityScore(ctx context.Context, userID int64, newScore float64, newReviewCount int64, newBiasCount int64) error {
	return nil
}
