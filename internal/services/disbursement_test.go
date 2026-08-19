package services

import (
	"errors"
	"math"
	"testing"
	"time"

	"example.com/disbursement/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockDisbursementRepository struct {
	mock.Mock
}

func (m *mockDisbursementRepository) Create(d *models.Disbursement) error {
	args := m.Called(d)
	return args.Error(0)
}

func (m *mockDisbursementRepository) FindByID(id string) (*models.Disbursement, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Disbursement), args.Error(1)
}

func (m *mockDisbursementRepository) FindAll(params models.ListDisbursementParams) ([]*models.Disbursement, int64, error) {
	args := m.Called(params)
	return args.Get(0).([]*models.Disbursement), args.Get(1).(int64), args.Error(2)
}

func (m *mockDisbursementRepository) Update(d *models.Disbursement) error {
	args := m.Called(d)
	return args.Error(0)
}

func validCreateRequest() models.CreateDisbursementRequest {
	return models.CreateDisbursementRequest{
		RecipientName: "Budi Santoso",
		AccountNumber: "1234567890",
		BankCode:      "BCA",
		Amount:        5_000_000,
		Note:          "invoice-42",
	}
}

func TestCalculateAdminFee(t *testing.T) {
	tests := []struct {
		name   string
		amount int64
		want   int64
	}{
		{name: "zero", amount: 0, want: 2500},
		{name: "one_below_minimum", amount: 9_999, want: 2500},
		{name: "one_below_fee_boundary", amount: 4_999_999, want: 2500},
		{name: "at_fee_boundary", amount: 5_000_000, want: 5000},
		{name: "above_fee_boundary", amount: 10_000_000, want: 5000},
		{name: "maximum_int64", amount: math.MaxInt64, want: 5000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CalculateAdminFee(tt.amount))
		})
	}
}

func TestValidateStatusTransition(t *testing.T) {
	tests := []struct {
		name    string
		current models.DisbursementStatus
		next    models.DisbursementStatus
		wantErr error
	}{
		{name: "pending_to_approved", current: models.StatusPending, next: models.StatusApproved},
		{name: "pending_to_rejected", current: models.StatusPending, next: models.StatusRejected},
		{name: "approved_to_approved", current: models.StatusApproved, next: models.StatusApproved, wantErr: ErrFinalStatus},
		{name: "rejected_to_approved", current: models.StatusRejected, next: models.StatusApproved, wantErr: ErrFinalStatus},
		{name: "pending_to_pending", current: models.StatusPending, next: models.StatusPending, wantErr: ErrInvalidStatus},
		{name: "pending_to_empty", current: models.StatusPending, next: "", wantErr: ErrInvalidStatus},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStatusTransition(tt.current, tt.next)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestDisbursementServiceCreate(t *testing.T) {
	t.Run("creates_pending_disbursement_with_correct_fee", func(t *testing.T) {
		repo := new(mockDisbursementRepository)
		repo.On("Create", mock.MatchedBy(func(d *models.Disbursement) bool {
			return d.RecipientName == "Budi Santoso" && d.AdminFee == 5000
		})).Return(nil).Once()
		svc := NewDisbursementService(repo)

		before := time.Now()
		got, err := svc.Create(validCreateRequest(), "operator-1")

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Regexp(t, `^DSB-[0-9a-f]{8}$`, got.ID)
		assert.Equal(t, int64(5000), got.AdminFee)
		assert.Equal(t, models.StatusPending, got.Status)
		assert.Equal(t, "operator-1", got.CreatedBy)
		assert.False(t, got.CreatedAt.Before(before))
		assert.False(t, got.UpdatedAt.Before(before))
		repo.AssertExpectations(t)
	})

	tests := []struct {
		name    string
		mutate  func(*models.CreateDisbursementRequest)
		wantErr error
	}{
		{name: "missing_recipient", mutate: func(req *models.CreateDisbursementRequest) { req.RecipientName = "" }, wantErr: ErrMissingField},
		{name: "missing_account_number", mutate: func(req *models.CreateDisbursementRequest) { req.AccountNumber = "" }, wantErr: ErrMissingField},
		{name: "amount_below_minimum", mutate: func(req *models.CreateDisbursementRequest) { req.Amount = 9_999 }, wantErr: ErrInvalidAmount},
		{name: "negative_amount", mutate: func(req *models.CreateDisbursementRequest) { req.Amount = -1 }, wantErr: ErrInvalidAmount},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockDisbursementRepository)
			svc := NewDisbursementService(repo)
			req := validCreateRequest()
			tt.mutate(&req)

			got, err := svc.Create(req, "operator-1")

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, got)
			if tt.name == "negative_amount" {
				assert.EqualError(t, err, "amount must be a positive integer >= 10000")
			}
			repo.AssertNotCalled(t, "Create", mock.Anything)
		})
	}

	t.Run("repository_failure_returns_no_disbursement", func(t *testing.T) {
		repoErr := errors.New("database unavailable")
		repo := new(mockDisbursementRepository)
		repo.On("Create", mock.AnythingOfType("*models.Disbursement")).Return(repoErr).Once()
		svc := NewDisbursementService(repo)

		got, err := svc.Create(validCreateRequest(), "operator-1")

		assert.ErrorIs(t, err, repoErr)
		assert.Nil(t, got, "a failed persistence operation must not leak partial state to the caller")
		repo.AssertExpectations(t)
	})
}

func TestDisbursementServiceUpdateStatus(t *testing.T) {
	pending := func() *models.Disbursement {
		return &models.Disbursement{ID: "DSB-12345678", Status: models.StatusPending}
	}
	request := models.UpdateStatusRequest{Status: models.StatusApproved, Note: "verified"}

	t.Run("updates_pending_disbursement", func(t *testing.T) {
		repo := new(mockDisbursementRepository)
		repo.On("FindByID", "DSB-12345678").Return(pending(), nil).Once()
		repo.On("Update", mock.MatchedBy(func(d *models.Disbursement) bool {
			return d.Status == models.StatusApproved && d.ApprovedBy == "admin-1" && d.Note == "verified"
		})).Return(nil).Once()
		svc := NewDisbursementService(repo)

		got, err := svc.UpdateStatus("DSB-12345678", request, "admin-1")

		require.NoError(t, err)
		assert.Equal(t, models.StatusApproved, got.Status)
		assert.Equal(t, "admin-1", got.ApprovedBy)
		repo.AssertExpectations(t)
	})

	t.Run("not_found", func(t *testing.T) {
		repo := new(mockDisbursementRepository)
		repo.On("FindByID", "missing").Return(nil, errors.New("not found")).Once()
		svc := NewDisbursementService(repo)

		got, err := svc.UpdateStatus("missing", request, "admin-1")

		assert.ErrorIs(t, err, ErrNotFound)
		assert.Nil(t, got)
		repo.AssertNotCalled(t, "Update", mock.Anything)
	})

	t.Run("already_in_final_state", func(t *testing.T) {
		repo := new(mockDisbursementRepository)
		final := pending()
		final.Status = models.StatusApproved
		repo.On("FindByID", final.ID).Return(final, nil).Once()
		svc := NewDisbursementService(repo)

		got, err := svc.UpdateStatus(final.ID, request, "admin-1")

		assert.ErrorIs(t, err, ErrFinalStatus)
		assert.Nil(t, got)
		repo.AssertNotCalled(t, "Update", mock.Anything)
	})

	t.Run("repository_update_failure", func(t *testing.T) {
		repoErr := errors.New("write failed")
		repo := new(mockDisbursementRepository)
		repo.On("FindByID", "DSB-12345678").Return(pending(), nil).Once()
		repo.On("Update", mock.AnythingOfType("*models.Disbursement")).Return(repoErr).Once()
		svc := NewDisbursementService(repo)

		got, err := svc.UpdateStatus("DSB-12345678", request, "admin-1")

		assert.ErrorIs(t, err, repoErr)
		assert.Nil(t, got)
		repo.AssertExpectations(t)
	})
}
