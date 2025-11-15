package model

import "time"

// Restaurant은 식당 자체의 기본 정보를 나타냅니다.
type Restaurant struct {
	RestaurantID   int64     // PK
	RestaurantName string    // 식당 이름
	LocationRefID  int64     // FK: Location 테이블 참조 (도시/지역 정보)
	CategoryRefID  int64     // FK: Category 테이블 참조 (음식 종류)
	CreatedAt      time.Time // 생성일
}
