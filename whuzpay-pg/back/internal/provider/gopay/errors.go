package gopay

import "errors"

var (
	// ErrCredentialsNotConfigured: merchant belum mengisi API key
	// gopay-notifications di Settings -- lihat respondCreatePaymentError
	// di payment_handler.go untuk pemetaannya ke HTTP 400.
	ErrCredentialsNotConfigured = errors.New("gopay: merchant belum mengatur API key gopay-notifications")
	// ErrQRISNotConfigured: kredensial ada, tapi account gopay-notifications
	// merchant ini belum upload QRIS -- lihat sub-project 1.
	ErrQRISNotConfigured = errors.New("gopay: merchant belum mengatur QRIS di gopay-notifications")
)
