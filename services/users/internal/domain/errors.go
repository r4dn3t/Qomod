package domain

import "errors"

// Common domain errors representing business rule violations.
var (
	ErrUserNotFound        = errors.New("user not found")
	ErrPhoneAlreadyExists  = errors.New("cellphone number is already registered")
	ErrEmailAlreadyExists  = errors.New("email address is already registered")
	ErrInvalidPhoneNumber  = errors.New("invalid cellphone number format (E.164 required, e.g. +12025550123)")
	ErrInvalidStatus       = errors.New("invalid user status")
	ErrUserDeleted         = errors.New("user account has been deleted")
	ErrUserSuspended       = errors.New("user account is suspended")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrWeakPassword        = errors.New("password must be at least 8 characters long")
	ErrCannotDeleteAccount = errors.New("unable to delete account")
)
