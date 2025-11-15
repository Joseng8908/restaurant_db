package repository_test

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3" // 드라이버 임포트
	"restaurant_db/internal/model"
	"restaurant_db/internal/repository"
)

// setupTestDB, insertMockLog 등 이전 버퍼 테스트에서 사용된 공통 함수가 여기에 포함되어야 합니다.

// --- TDD: TestCreateUser ---
func TestCreateUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	// 1. Given: 새로운 사용자 데이터
	mockUser := model.User{
		Username: "TestUser123",
	}

	// 2. When: 사용자 생성
	err := repo.Create(ctx, &mockUser)

	// 3. Then: 결과 검증
	if err != nil {
		t.Fatalf("Create user failed: %v", err)
	}
	// ID가 할당되었는지 확인
	if mockUser.UserID <= 0 {
		t.Errorf("Expected UserID to be assigned, got %d", mockUser.UserID)
	}

	// DB에 실제로 데이터가 들어갔는지 확인 (FindByID를 통해 간접 검증)
	fetchedUser, err := repo.FindByID(ctx, mockUser.UserID)
	if err != nil {
		t.Fatalf("FindByID failed after creation: %v", err)
	}
	if fetchedUser == nil {
		t.Fatalf("Expected user to be found, got nil")
	}
	if fetchedUser.Username != "TestUser123" {
		t.Errorf("Expected username 'TestUser123', got %s", fetchedUser.Username)
	}
	// DDL 기본값 검증 (0.5, 0)
	if fetchedUser.ReliabilityScore != 0.5 || fetchedUser.ReviewCount != 0 {
		t.Errorf("Expected default scores (0.5/0), got %.1f/%d", fetchedUser.ReliabilityScore, fetchedUser.ReviewCount)
	}
}

// --- TDD: TestUpdateReliabilityScore ---
func TestUpdateReliabilityScore(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	// 1. Given: 사용자 생성 및 초기 데이터 확인
	user := model.User{Username: "ScoreTester"}
	if err := repo.Create(ctx, &user); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	// 2. When: 신뢰도 점수 업데이트
	newScore := 0.85
	newReviewCount := int64(10)
	newBiasCount := int64(2)

	err := repo.UpdateReliabilityScore(ctx, user.UserID, newScore, newReviewCount, newBiasCount)

	// 3. Then: 업데이트 결과 검증
	if err != nil {
		t.Fatalf("UpdateReliabilityScore failed: %v", err)
	}

	// DB에서 업데이트된 데이터를 다시 조회하여 확인
	updatedUser, err := repo.FindByID(ctx, user.UserID)
	if err != nil {
		t.Fatalf("FindByID failed after update: %v", err)
	}

	if updatedUser.ReliabilityScore != newScore {
		t.Errorf("Expected score %.2f, got %.2f", newScore, updatedUser.ReliabilityScore)
	}
	if updatedUser.ReviewCount != newReviewCount {
		t.Errorf("Expected review count %d, got %d", newReviewCount, updatedUser.ReviewCount)
	}
	if updatedUser.BiasCount != newBiasCount {
		t.Errorf("Expected bias count %d, got %d", newBiasCount, updatedUser.BiasCount)
	}
}

// TestFindByIDNotFound: 존재하지 않는 ID 조회 테스트
func TestFindByIDNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	// 존재할 수 없는 ID (예: 99999)로 조회
	user, err := repo.FindByID(ctx, 99999)

	if err != nil {
		t.Fatalf("FindByID should not return error for not found, got: %v", err)
	}
	if user != nil {
		t.Errorf("Expected user to be nil when not found, got: %+v", user)
	}
}
