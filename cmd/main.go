package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3" // DB 드라이버
	"restaurant_db/internal/model"
	"restaurant_db/internal/repository"
	"restaurant_db/internal/worker"
	"restaurant_db/service"
)

const (
	// 성능 분석을 위한 시뮬레이션 설정
	TestWriteCount  = 1000 // 쓰기 요청 횟수
	TestReadCount   = 100  // 읽기 성능 측정 반복 횟수
	WorkerBatchSize = 100
)

// setupDB: 테스트용 인메모리 DB를 설정하고 스키마를 초기화합니다.
func setupDB() *sql.DB {
	// 1. SQLite 인메모리 DB 연결
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatalf("could not open database connection: %v", err)
	}

	// 2. 스키마 로드 및 실행
	// 경로는 현재 main.go를 기준으로 설정해야 합니다. (프로젝트 루트에서 실행 가정)
	schemaSQL, err := os.ReadFile("internal/db/schema.sql")
	if err != nil {
		// Go 파일 위치에 따라 경로를 조정해야 할 수 있습니다.
		log.Fatalf("could not read schema file: %v. Check path.", err)
	}
	_, err = db.Exec(string(schemaSQL))
	if err != nil {
		log.Fatalf("could not execute schema: %v", err)
	}

	return db
}

// initSystem: 모든 Repository와 Service, Worker를 초기화하고 연결합니다.
func initSystem(db *sql.DB) (repository.BufferRepository, repository.UserRepository, service.RestaurantService) {
	// Repository 초기화
	bufferRepo := repository.NewBufferRepository(db)
	userRepo := repository.NewUserRepository(db)
	cacheRepo := repository.NewCacheRepository(db)
	restaurantRepo := repository.NewRestaurantRepository(db)

	// Service 초기화 (캐싱/릴레이션 접근 로직 포함)
	restaurantService := service.NewRestaurantService(cacheRepo, restaurantRepo)

	// Worker 초기화 (버퍼 -> UserRepo 접근 로직 포함)
	_ = worker.NewCheckpointWorker(bufferRepo, userRepo, WorkerBatchSize, 100*time.Millisecond) // Worker는 시뮬레이션용이므로 Run은 하지 않습니다.

	return bufferRepo, userRepo, *restaurantService
}

func main() {
	// 1. 시스템 초기화 및 DB 설정
	db := setupDB() // main에서는 *testing.T 대신 nil을 전달하거나 구조를 조정
	defer db.Close()

	bufferRepo, userRepo, restaurantService := initSystem(db)
	ctx := context.Background()

	// 임시 User 생성 (업데이트 대상이 필요하므로)
	user := model.User{Username: "PerformanceTarget"}
	if err := userRepo.Create(ctx, &user); err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	fmt.Println("--- DB 서비스 성능 비교 시뮬레이션 시작 ---")

	// --- A. 쓰기 성능 비교 (버퍼링 vs 직접 반영) ---
	simulateBufferedWrite(ctx, bufferRepo, user.UserID)
	simulateDirectWrite(ctx, userRepo, user.UserID)

	// Worker의 COMMIT 로직을 실행하여 버퍼를 정리해야 읽기 시나리오를 시작할 수 있습니다.
	// 실제 Worker의 COMMIT 로직을 여기에 가져와 실행합니다.
	commitWorker := worker.NewCheckpointWorker(bufferRepo, userRepo, TestWriteCount, time.Minute)
	commitWorker.ProcessCheckpoint(ctx) // 버퍼의 1000개 로그를 모두 반영

	fmt.Println("\n" + "--- B. 읽기 성능 비교 (캐싱 vs 릴레이션 직접 접근) ---")

	// 임시 Cache 데이터 삽입 (Cache Hit 시뮬레이션용)
	insertMockCache(db, 1) // Restaurant ID 1에 캐시 데이터 삽입

	// 새로운 함수를 호출하여 평균 결과만 출력합니다.
	simulateReadScenario(ctx, restaurantService)
}

// simulateBufferedWrite: 1000개의 쓰기 요청을 버퍼에 담는 시간 측정
func simulateBufferedWrite(ctx context.Context, repo repository.BufferRepository, userID int64) {
	start := time.Now()
	for i := 0; i < TestWriteCount; i++ {
		payload := fmt.Sprintf(`{"user_id": %d, "new_score": 0.51, "new_review_count": 1, "new_bias_count": 0}`, userID)
		bufferLog := model.BufferLog{
			TransactionType: "UPDATE",
			TargetTable:     "User",
			Payload:         payload,
			TargetRecordID:  userID,
		}
		if err := repo.AddLog(ctx, &bufferLog); err != nil {
			log.Fatalf("Buffer Write failed: %v", err)
		}
	}
	elapsed := time.Since(start)
	fmt.Printf("[쓰기 시나리오 A - 버퍼링] %d건 AddLog 시간: %s (매우 빠름)\n", TestWriteCount, elapsed)
}

// simulateDirectWrite: 1000개의 쓰기 요청을 DB에 직접 반영하는 시간 측정
func simulateDirectWrite(ctx context.Context, repo repository.UserRepository, userID int64) {
	start := time.Now()
	for i := 0; i < TestWriteCount; i++ {
		// 실제로는 AddLog가 아닌, 직접 DB에 영향을 주는 UpdateReliabilityScore를 호출한다고 가정
		if err := repo.UpdateReliabilityScore(ctx, userID, 0.5, 1, 0); err != nil {
			log.Fatalf("Direct Write failed: %v", err)
		}
	}
	elapsed := time.Since(start)
	fmt.Printf("[쓰기 시나리오 B - 직접 반영] %d건 Update 시간: %s (느림 시뮬레이션)\n", TestWriteCount, elapsed)
}

// insertMockCache: 캐시 Hit 시뮬레이션을 위한 데이터 삽입
func insertMockCache(db *sql.DB, restaurantID int64) {
	query := `
       INSERT INTO Cache_Metadata (
          restaurant_id, location_ref_id, category_ref_id, weighted_rating, 
          total_weighted_reviews, cache_score, last_cache_updated_at
       ) VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(
		query,
		restaurantID, 1, 1, 4.5, 100, 1.0, time.Now().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		log.Fatalf("Failed to insert mock cache: %v", err)
	}
}

// simulateReadScenario: 캐싱 vs 릴레이션 접근 성능 비교 및 평균 시간 출력 (신규 추가)
func simulateReadScenario(ctx context.Context, s service.RestaurantService) {
	var totalCacheHitTime time.Duration
	var totalCacheMissTime time.Duration

	fmt.Printf("--- 읽기 시뮬레이션 시작 (총 %d회 반복) ---\n", TestReadCount*2)

	for i := 0; i < TestReadCount; i++ {
		// 1. 캐시 히트 시나리오 (Restaurant 1)
		startHit := time.Now()
		// isHit 등의 반환값은 여기서 사용하지 않음
		s.FindRestaurantSummary(ctx, 1)
		totalCacheHitTime += time.Since(startHit)

		// 2. 캐시 미스 시나리오 (Restaurant 99)
		startMiss := time.Now()
		s.FindRestaurantSummary(ctx, 99)
		totalCacheMissTime += time.Since(startMiss)
	}

	// 평균 계산
	avgHitTime := totalCacheHitTime / time.Duration(TestReadCount)
	avgMissTime := totalCacheMissTime / time.Duration(TestReadCount)

	// 최종 출력
	fmt.Println("--- 읽기 성능 결과 (평균) ---")
	fmt.Printf("[Read] ✅ CACHE HIT 평균 시간 (Restaurant 1): %s\n", avgHitTime)
	fmt.Printf("[Read] ❌ CACHE MISS 평균 시간 (Restaurant 99): %s\n", avgMissTime)
}
