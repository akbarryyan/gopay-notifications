package gopayonboard

import "errors"

// ErrEmailTaken/ErrUsernameTaken -- sentinel yang dipetakan dari respons
// 409 gopay-notifications ("error":"email_taken"/"username_taken", lihat
// backend/internal/httpapi/signup.go). AuthService.RegisterMerchant
// memakai ini untuk memutuskan apakah retry username ada gunanya
// (ErrUsernameTaken -- ya) atau tidak (ErrEmailTaken -- tidak, email tidak
// berubah antar percobaan).
var (
	ErrEmailTaken    = errors.New("gopayonboard: email sudah terdaftar di gopay-notifications")
	ErrUsernameTaken = errors.New("gopayonboard: username sudah dipakai di gopay-notifications")
)
