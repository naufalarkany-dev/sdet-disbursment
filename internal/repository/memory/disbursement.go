package memory

import (
	"errors"
	"strings"
	"sync"

	"example.com/disbursement/internal/models"
)

// DisbursementRepository adalah implementasi in-memory dari repository.DisbursementRepository.
// Digunakan untuk integration test — tidak perlu koneksi database nyata.
type DisbursementRepository struct {
	mu   sync.RWMutex
	data map[string]*models.Disbursement
}

func NewDisbursementRepository() *DisbursementRepository {
	return &DisbursementRepository{
		data: make(map[string]*models.Disbursement),
	}
}

func (r *DisbursementRepository) Create(d *models.Disbursement) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[d.ID] = d
	return nil
}

func (r *DisbursementRepository) FindByID(id string) (*models.Disbursement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.data[id]
	if !ok {
		return nil, errors.New("not found")
	}
	// Return copy to prevent external mutation of stored data
	copy := *d
	return &copy, nil
}

func (r *DisbursementRepository) FindAll(params models.ListDisbursementParams) ([]*models.Disbursement, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var filtered []*models.Disbursement
	for _, d := range r.data {
		if params.Status != "" && string(d.Status) != params.Status {
			continue
		}
		if params.Search != "" && !strings.Contains(strings.ToLower(d.RecipientName), strings.ToLower(params.Search)) {
			continue
		}
		copy := *d
		filtered = append(filtered, &copy)
	}

	total := int64(len(filtered))

	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 {
		params.Limit = 10
	}

	start := (params.Page - 1) * params.Limit
	if start >= len(filtered) {
		return []*models.Disbursement{}, total, nil
	}
	end := start + params.Limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[start:end], total, nil
}

func (r *DisbursementRepository) Update(d *models.Disbursement) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data[d.ID]; !ok {
		return errors.New("not found")
	}
	copy := *d
	r.data[d.ID] = &copy
	return nil
}

// UpdateIfStatus atomically updates a disbursement only when its stored status
// still matches the status observed by the caller.
func (r *DisbursementRepository) UpdateIfStatus(d *models.Disbursement, expected models.DisbursementStatus) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	current, ok := r.data[d.ID]
	if !ok {
		return false, errors.New("not found")
	}
	if current.Status != expected {
		return false, nil
	}
	copy := *d
	r.data[d.ID] = &copy
	return true, nil
}

// Reset menghapus semua data. Gunakan di test cleanup (t.Cleanup atau defer).
func (r *DisbursementRepository) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data = make(map[string]*models.Disbursement)
}
