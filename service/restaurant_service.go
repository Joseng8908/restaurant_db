package service

import (
	"context"
	"fmt"
	"time"

	"restaurant_db/internal/model"
	"restaurant_db/internal/repository"
)

type RestaurantService struct {
	CacheRepo      repository.CacheRepository
	RestaurantRepo repository.RestaurantRepository
}

func NewRestaurantService(cacheRepo repository.CacheRepository, restaurantRepo repository.RestaurantRepository) *RestaurantService {
	return &RestaurantService{
		CacheRepo:      cacheRepo,
		RestaurantRepo: restaurantRepo,
	}
}

// FindRestaurantSummary: 캐시 우선 조회 로직 (성능 분석용)
func (s *RestaurantService) FindRestaurantSummary(ctx context.Context, restaurantID int64) (*model.CacheMetadata, error) {
	// 캐시 조회 시작 시간 기록
	startTime := time.Now()

	// 1. 캐시 조회 시도
	cache, err := s.CacheRepo.FindCacheByID(ctx, restaurantID)
	if err != nil {
		// DB 오류
		return nil, err
	}

	if cache != nil {
		// 캐시 히트
		duration := time.Since(startTime)
		fmt.Printf("[Read] CACHE HIT: Restaurant %d 조회 시간: %s\n", restaurantID, duration)
		return cache, nil
	}

	// 2. 캐시 미스: 릴레이션 직접 접근 시도
	fmt.Printf("[Read] CACHE MISS: 릴레이션 직접 접근 (느린 I/O 시뮬레이션 시작)\n")

	// RestaurantRepo는 느린 I/O를 시뮬레이션
	_, err = s.RestaurantRepo.FindByID(ctx, restaurantID)
	if err != nil {
		return nil, fmt.Errorf("failed to access primary relation: %w", err)
	}

	duration := time.Since(startTime)
	fmt.Printf("[Read] RELATIONAL ACCESS: Restaurant %d 조회 시간: %s\n", restaurantID, duration)

	// 실제로는 캐시를 재구성하고 반환해야 하지만, 분석을 위해 미스 처리
	return nil, fmt.Errorf("cache miss occurred")
}
