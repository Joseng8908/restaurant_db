// internal/model/cache_metadata.go (또는 model 패키지 내 다른 파일)

package model

import "time"

// CacheMetadata는 식당의 가중 평점, 리뷰 수 등 캐싱된 정보를 저장합니다.
type CacheMetadata struct {
	RestaurantID         int64     // FK (Restaurant 테이블 참조)
	LocationRefID        int64     // FK (Location 테이블 참조)
	CategoryRefID        int64     // FK (Category 테이블 참조)
	WeightedRating       float64   // 가중 평점
	TotalWeightedReviews int64     // 총 가중 리뷰 수
	CacheScore           float64   // 캐시 점수 (갱신 우선순위 결정용)
	LastCacheUpdatedAt   time.Time // 캐시 최종 업데이트 시간
}
