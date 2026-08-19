package repository

import "example.com/disbursement/internal/models"

// DisbursementRepository adalah interface yang harus diimplementasikan
// oleh layer database. Interface ini yang kamu gunakan untuk membuat mock di unit test.
type DisbursementRepository interface {
	Create(d *models.Disbursement) error
	FindByID(id string) (*models.Disbursement, error)
	FindAll(params models.ListDisbursementParams) ([]*models.Disbursement, int64, error)
	Update(d *models.Disbursement) error
}
