package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"restaurant_db/internal/model"
)

type CacheRepository interface {
	FindCacheByID(ctx context.Context, restaurantID int64) (*model.CacheMetadata, error)
}

type CacheRepoImpl struct {
	DB *sql.DB
}

func NewCacheRepository(db *sql.DB) CacheRepository {
	return &CacheRepoImpl{DB: db}
}

// FindCacheByID: 캐시 테이블에서 데이터를 조회합니다.
func (r *CacheRepoImpl) FindCacheByID(ctx context.Context, restaurantID int64) (*model.CacheMetadata, error) {
	cache := &model.CacheMetadata{}

	query := `
		SELECT 
			restaurant_id, location_ref_id, category_ref_id, weighted_rating, 
			total_weighted_reviews, cache_score, last_cache_updated_at 
		FROM Cache_Metadata 
		WHERE restaurant_id = ?`

	row := r.DB.QueryRowContext(ctx, query, restaurantID)

	var lastUpdatedStr string

	err := row.Scan(
		&cache.RestaurantID,
		&cache.LocationRefID,
		&cache.CategoryRefID,
		&cache.WeightedRating,
		&cache.TotalWeightedReviews,
		&cache.CacheScore,
		&lastUpdatedStr,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // 캐시 미스 (성능 분석을 위해 중요)
		}
		return nil, fmt.Errorf("failed to find cache by ID: %w", err)
	}

	const sqliteTimeFormat = "2006-01-02 15:04:05"
	cache.LastCacheUpdatedAt, err = time.Parse(sqliteTimeFormat, lastUpdatedStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cache updated_at: %w", err)
	}

	return cache, nil
}
