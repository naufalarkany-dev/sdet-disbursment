package services

import (
	"sync"
	"sync/atomic"
	"testing"

	"example.com/disbursement/internal/models"
	"example.com/disbursement/internal/repository/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type synchronizedFindRepository struct {
	*memory.DisbursementRepository
	targetID string
	workers  int32
	arrived  atomic.Int32
	release  chan struct{}
}

func (r *synchronizedFindRepository) FindByID(id string) (*models.Disbursement, error) {
	d, err := r.DisbursementRepository.FindByID(id)
	if id != r.targetID || err != nil {
		return d, err
	}

	arrival := r.arrived.Add(1)
	if arrival <= r.workers {
		if arrival == r.workers {
			close(r.release)
		}
		<-r.release
	}
	return d, nil
}

func TestUpdateStatusConcurrentApprovalIsAtomic(t *testing.T) {
	const workerCount = 10

	baseRepo := memory.NewDisbursementRepository()
	d := &models.Disbursement{ID: "DSB-concurrent", Status: models.StatusPending}
	require.NoError(t, baseRepo.Create(d))
	repo := &synchronizedFindRepository{
		DisbursementRepository: baseRepo,
		targetID:               d.ID,
		workers:                workerCount,
		release:                make(chan struct{}),
	}
	svc := NewDisbursementService(repo)

	start := make(chan struct{})
	results := make(chan error, workerCount)
	var ready sync.WaitGroup
	var workers sync.WaitGroup
	ready.Add(workerCount)
	workers.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go func() {
			defer workers.Done()
			ready.Done()
			<-start
			_, err := svc.UpdateStatus(d.ID, models.UpdateStatusRequest{Status: models.StatusApproved}, "admin")
			results <- err
		}()
	}

	ready.Wait()
	close(start)
	workers.Wait()
	close(results)

	successes := 0
	failures := 0
	for err := range results {
		if err == nil {
			successes++
			continue
		}
		failures++
		assert.ErrorIs(t, err, ErrFinalStatus, "a losing approval must clearly report the final state")
	}
	assert.LessOrEqual(t, successes, 1, "only one concurrent approval may succeed")
	assert.Equal(t, workerCount-successes, failures, "all other approvals must be rejected")
}
