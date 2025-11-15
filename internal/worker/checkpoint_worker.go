package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"restaurant_db/internal/model"
	"restaurant_db/internal/repository"
	"time"
)

// CheckpointWorker는 주기적으로 Buffer_Log를 읽어 실제 DB에 반영합니다.
type CheckpointWorker struct {
	BufferRepo repository.BufferRepository
	UserRepo   repository.UserRepository

	BatchSize int
	Interval  time.Duration
}

func NewCheckpointWorker(
	bufferRepo repository.BufferRepository,
	userRepo repository.UserRepository,
	batchSize int,
	interval time.Duration,
) *CheckpointWorker {
	return &CheckpointWorker{
		BufferRepo: bufferRepo,
		UserRepo:   userRepo,
		BatchSize:  batchSize,
		Interval:   interval,
	}
}

// Run: 워커를 시작하는 메인 루프 (성능 분석 시 시뮬레이션에 사용됨)
func (w *CheckpointWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()

	fmt.Println("CheckpointWorker started. Interval:", w.Interval)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("CheckpointWorker stopped.")
			return
		case <-ticker.C:
			w.ProcessCheckpoint(ctx)
		}
	}
}

// ProcessCheckpoint: 버퍼에서 로그를 읽어와 DB에 반영하는 핵심 로직
func (w *CheckpointWorker) ProcessCheckpoint(ctx context.Context) {
	// 1. Pending 로그 조회
	logs, err := w.BufferRepo.GetPendingLogs(ctx, w.BatchSize)
	if err != nil {
		fmt.Println("Error getting pending logs:", err)
		return
	}
	if len(logs) == 0 {
		return
	}

	fmt.Printf("[Write] Processing %d logs...\n", len(logs))

	var committedIDs []int64

	// 2. 로그를 순회하며 실제 테이블에 반영 (COMMIT)
	for _, log := range logs {
		if err := w.processLog(ctx, log); err != nil {
			fmt.Printf("Failed to process log ID %d: %v\n", log.LogID, err)
			continue
		}
		committedIDs = append(committedIDs, log.LogID)
	}

	// 3. 반영 성공한 로그의 상태 업데이트
	if len(committedIDs) > 0 {
		if err := w.BufferRepo.UpdateCommitted(ctx, committedIDs); err != nil {
			fmt.Println("Error updating committed status:", err)
		}
		fmt.Printf("[Write] Successfully committed and marked %d logs.\n", len(committedIDs))
	}
}

// processLog: 단일 로그를 해석하여 적절한 Repository 메소드를 호출합니다.
func (w *CheckpointWorker) processLog(ctx context.Context, log model.BufferLog) error {
	switch log.TargetTable {
	case "User":
		// User 업데이트 페이로드를 해석
		var payload struct {
			UserID         int64   `json:"user_id"`
			NewScore       float64 `json:"new_score"`
			NewReviewCount int64   `json:"new_review_count"`
			NewBiasCount   int64   `json:"new_bias_count"`
		}
		// Go 1.22+에서는 encoding/json의 Unmarshal이 json.RawMessage 대신 string을 허용합니다.
		if err := json.Unmarshal([]byte(log.Payload), &payload); err != nil {
			return fmt.Errorf("failed to unmarshal User payload: %w", err)
		}

		// UserRepo.UpdateReliabilityScore 호출 (실제 테이블 반영)
		return w.UserRepo.UpdateReliabilityScore(
			ctx,
			payload.UserID,
			payload.NewScore,
			payload.NewReviewCount,
			payload.NewBiasCount,
		)

	default:
		return fmt.Errorf("unsupported target table: %s", log.TargetTable)
	}
}
