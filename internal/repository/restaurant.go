package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"restaurant_db/internal/model"
)

// RestaurantRepository: Restaurant 테이블에 접근합니다.
type RestaurantRepository interface {
	// FindByID: 캐시 미스 시 릴레이션에 직접 접근하여 식당 정보를 조회합니다.
	FindByID(ctx context.Context, restaurantID int64) (*model.Restaurant, error)
}

// RestaurantRepoImpl은 RestaurantRepository 인터페이스를 구현합니다.
type RestaurantRepoImpl struct {
	DB *sql.DB
}

func NewRestaurantRepository(db *sql.DB) RestaurantRepository {
	return &RestaurantRepoImpl{DB: db}
}

// FindByID: 캐시 미스 시 릴레이션에 직접 접근하여 식당 정보를 조회합니다. (느린 I/O 시뮬레이션)
func (r *RestaurantRepoImpl) FindByID(ctx context.Context, restaurantID int64) (*model.Restaurant, error) {
	// 성능 분석을 위해 릴레이션 접근 시간을 시뮬레이션합니다.
	time.Sleep(10 * time.Millisecond)

	restaurant := &model.Restaurant{}

	// 실제 쿼리 로직은 다음과 같지만, 최소 구현을 위해 간단히 반환합니다.
	/*
		query := `
			SELECT restaurant_id, restaurant_name, location_ref_id, category_ref_id, created_at
			FROM Restaurant
			WHERE restaurant_id = ?`

		row := r.DB.QueryRowContext(ctx, query, restaurantID)
		// ... (Scan 및 시간 파싱 로직)
	*/

	// 임시 데이터 반환
	restaurant.RestaurantID = restaurantID
	restaurant.RestaurantName = fmt.Sprintf("식당_%d", restaurantID)

	return restaurant, nil
}
