package models

import "time"

type DisbursementStatus string

const (
	StatusPending  DisbursementStatus = "PENDING"
	StatusApproved DisbursementStatus = "APPROVED"
	StatusRejected DisbursementStatus = "REJECTED"
)

type Disbursement struct {
	ID            string             `json:"id"`
	RecipientName string             `json:"recipient_name"`
	AccountNumber string             `json:"account_number"`
	BankCode      string             `json:"bank_code"`
	Amount        int64              `json:"amount"`
	AdminFee      int64              `json:"admin_fee"`
	Status        DisbursementStatus `json:"status"`
	Note          string             `json:"note"`
	CreatedBy     string             `json:"created_by"`
	ApprovedBy    string             `json:"approved_by,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

type CreateDisbursementRequest struct {
	RecipientName string `json:"recipient_name"`
	AccountNumber string `json:"account_number"`
	BankCode      string `json:"bank_code"`
	Amount        int64  `json:"amount"`
	Note          string `json:"note"`
}

type UpdateStatusRequest struct {
	Status DisbursementStatus `json:"status"`
	Note   string             `json:"note"`
}

type ListDisbursementParams struct {
	Page   int
	Limit  int
	Search string
	Status string
}
