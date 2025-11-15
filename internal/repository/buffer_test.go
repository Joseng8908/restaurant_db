package repository_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"restaurant_db/internal/model"
	"restaurant_db/internal/repository"
)

// setupTestDB는 TDD를 위해 테스트용 SQLite 인메모리 DB를 설정하고 초기화합니다.
func setupTestDB(t *testing.T) *sql.DB {
	// 1. SQLite 인메모리 DB 연결 (테스트 후 자동 삭제됨)
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("could not open database connection: %v", err)
	}

	// 2. 스키마 로드 및 실행 (DDL 실행)
	// 실제 프로젝트에서는 internal/db/db.go의 InitDB 함수를 사용합니다.
	schemaSQL, err := os.ReadFile("../db/schema.sql")
	if err != nil {
		t.Fatalf("could not read schema file: %v", err)
	}
	_, err = db.Exec(string(schemaSQL))
	if err != nil {
		t.Fatalf("could not execute schema: %v", err)
	}

	return db
}

// TestAddLog는 BufferRepository.AddLog 함수의 TDD 테스트 케이스입니다.
func TestAddLog(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := repository.NewBufferRepository(db)
	ctx := context.Background()

	// 1. Given: 테스트에 필요한 Mock 데이터 준비 (예: Review INSERT 명령)
	mockLog := model.BufferLog{
		TransactionType: "INSERT",
		TargetTable:     "Review",
		Payload:         `{"review_id": 100, "rating": 5.0}`,
		TargetRecordID:  0, // INSERT는 ID가 나중에 할당됨
	}

	// 2. When: 함수 실행
	err := repo.AddLog(ctx, &mockLog)

	// 3. Then: 결과 검증
	if err != nil {
		t.Fatalf("AddLog failed: %v", err)
	}

	// DB에 로그가 성공적으로 들어갔는지 확인하는 추가 쿼리 로직
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM Buffer_Log WHERE target_table = 'Review'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count log: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 log, got %d", count)
	}
}

// helper function: 테스트를 위한 mock 로그를 DB에 삽입합니다.
func insertMockLog(t *testing.T, db *sql.DB, log model.BufferLog) {
	query := `
		INSERT INTO Buffer_Log (
			transaction_type, 
			target_table, 
			payload, 
			target_record_id,
            log_updated_at
		) VALUES (?, ?, ?, ?, ?)`

	_, err := db.Exec(
		query,
		log.TransactionType,
		log.TargetTable,
		log.Payload,
		log.TargetRecordID,
		time.Now().Format("2006-01-02 15:04:05"), // SQLite TEXT 포맷에 맞춰 시간 삽입
	)

	if err != nil {
		t.Fatalf("Failed to insert mock log: %v", err)
	}
}

// TestBatchCommitLifecycle은 버퍼 로그의 전체 배치 커밋 주기를 테스트합니다.
func TestBatchCommitLifecycle(t *testing.T) {
	db := setupTestDB(t) // DB 초기화 및 연결
	defer db.Close()

	repo := repository.NewBufferRepository(db)
	ctx := context.Background()

	// --- 1. Given: 5개의 Pending 로그를 DB에 직접 삽입 (AddLog 구현 전에도 테스트 가능) ---
	// TDD 원칙에 따라, repo.AddLog 대신 DB를 직접 사용해 데이터 준비 (현재는 AddLog 구현 완료됨)
	for i := 1; i <= 5; i++ {
		log := model.BufferLog{
			TransactionType: "UPDATE",
			TargetTable:     "User",
			Payload:         fmt.Sprintf(`{"user_id": %d, "score": 0.5}`, i),
			TargetRecordID:  int64(i),
			IsCommitted:     0,
		}
		insertMockLog(t, db, log)
	}

	// --- 2. When: Pending 로그를 3개만 조회합니다 (LIMIT 테스트) ---
	const limit = 3
	pendingLogs, err := repo.GetPendingLogs(ctx, limit)

	// --- 3. Then (조회): 결과 검증 ---
	if err != nil {
		t.Fatalf("GetPendingLogs failed: %v", err)
	}
	if len(pendingLogs) != limit {
		t.Errorf("Expected %d pending logs, got %d", limit, len(pendingLogs))
	}
	// 첫 번째 로그의 ID가 1인지 확인 (ORDER BY log_id ASC 확인)
	if pendingLogs[0].LogID != 1 {
		t.Errorf("Expected first log ID to be 1, got %d", pendingLogs[0].LogID)
	}

	// --- 4. When: 조회된 로그(ID 1, 2, 3)를 커밋 완료로 표시합니다 ---
	var commitIDs []int64
	for _, log := range pendingLogs {
		commitIDs = append(commitIDs, log.LogID)
	}

	err = repo.UpdateCommitted(ctx, commitIDs)

	// --- 5. Then (커밋): 결과 검증 ---
	if err != nil {
		t.Fatalf("UpdateCommitted failed: %v", err)
	}

	// DB에서 log_id=1의 is_committed 상태를 확인합니다.
	var isCommitted int
	err = db.QueryRow("SELECT is_committed FROM Buffer_Log WHERE log_id = 1").Scan(&isCommitted)
	if err != nil {
		t.Fatalf("Failed to query committed status: %v", err)
	}
	if isCommitted != 1 {
		t.Errorf("Expected log ID 1 to be committed (1), got %d", isCommitted)
	}

	// --- 6. Final Check: 남은 Pending 로그가 2개인지 확인 (ID 4, 5) ---
	remainingLogs, err := repo.GetPendingLogs(ctx, limit)
	if err != nil {
		t.Fatalf("Final GetPendingLogs failed: %v", err)
	}
	if len(remainingLogs) != 2 {
		t.Errorf("Expected 2 remaining pending logs, got %d", len(remainingLogs))
	}
	// 남은 로그의 ID가 4인지 확인
	if remainingLogs[0].LogID != 4 {
		t.Errorf("Expected remaining log ID to be 4, got %d", remainingLogs[0].LogID)
	}
}
