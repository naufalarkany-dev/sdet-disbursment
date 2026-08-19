package services

import (
	"errors"
	"fmt"
	"time"

	"example.com/disbursement/internal/models"
	"example.com/disbursement/internal/repository"
	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("disbursement not found")
	ErrInvalidStatus = errors.New("invalid status transition")
	ErrFinalStatus   = errors.New("disbursement already in final state")
	ErrInvalidAmount = errors.New("amount must be a positive integer >= 10000")
	ErrMissingField  = errors.New("required field is missing")
)

type DisbursementService struct {
	repo repository.DisbursementRepository
}

type atomicStatusUpdater interface {
	UpdateIfStatus(d *models.Disbursement, expected models.DisbursementStatus) (bool, error)
}

func NewDisbursementService(repo repository.DisbursementRepository) *DisbursementService {
	return &DisbursementService{repo: repo}
}

// CalculateAdminFee menghitung biaya admin berdasarkan jumlah transaksi.
// Fungsi ini adalah pure function — tidak bergantung pada state eksternal.
func CalculateAdminFee(amount int64) int64 {
	if amount < 5_000_000 {
		return 2500
	}
	return 5000
}

// ValidateStatusTransition memvalidasi apakah perubahan status diperbolehkan.
// Disbursement yang sudah APPROVED atau REJECTED tidak bisa diubah.
func ValidateStatusTransition(current, next models.DisbursementStatus) error {
	if current == models.StatusApproved || current == models.StatusRejected {
		return ErrFinalStatus
	}
	if next != models.StatusApproved && next != models.StatusRejected {
		return ErrInvalidStatus
	}
	return nil
}

func (s *DisbursementService) Create(req models.CreateDisbursementRequest, createdBy string) (*models.Disbursement, error) {
	if req.RecipientName == "" {
		return nil, fmt.Errorf("%w: recipient_name", ErrMissingField)
	}
	if req.AccountNumber == "" {
		return nil, fmt.Errorf("%w: account_number", ErrMissingField)
	}
	if req.BankCode == "" {
		return nil, fmt.Errorf("%w: bank_code", ErrMissingField)
	}
	if req.Amount < 10_000 {
		return nil, ErrInvalidAmount
	}

	d := &models.Disbursement{
		ID:            fmt.Sprintf("DSB-%s", uuid.New().String()[:8]),
		RecipientName: req.RecipientName,
		AccountNumber: req.AccountNumber,
		BankCode:      req.BankCode,
		Amount:        req.Amount,
		AdminFee:      CalculateAdminFee(req.Amount),
		Status:        models.StatusPending,
		Note:          req.Note,
		CreatedBy:     createdBy,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.repo.Create(d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *DisbursementService) UpdateStatus(id string, req models.UpdateStatusRequest, approvedBy string) (*models.Disbursement, error) {
	d, err := s.repo.FindByID(id)
	if err != nil {
		return nil, ErrNotFound
	}

	if err := ValidateStatusTransition(d.Status, req.Status); err != nil {
		return nil, err
	}

	previousStatus := d.Status
	d.Status = req.Status
	d.ApprovedBy = approvedBy
	d.Note = req.Note
	d.UpdatedAt = time.Now()

	if updater, ok := s.repo.(atomicStatusUpdater); ok {
		updated, err := updater.UpdateIfStatus(d, previousStatus)
		if err != nil {
			return nil, err
		}
		if !updated {
			return nil, ErrFinalStatus
		}
		return d, nil
	}

	if err := s.repo.Update(d); err != nil {
		return nil, err
	}
	return d, nil
}

func (s *DisbursementService) GetByID(id string) (*models.Disbursement, error) {
	d, err := s.repo.FindByID(id)
	if err != nil {
		return nil, ErrNotFound
	}
	return d, nil
}

func (s *DisbursementService) List(params models.ListDisbursementParams) ([]*models.Disbursement, int64, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.Limit < 1 || params.Limit > 100 {
		params.Limit = 10
	}
	return s.repo.FindAll(params)
}
