package domain

import "errors"

// These are the errors of the rules of the domain. A caller tests for them with errors.Is.
var (
	ErrInvalidEmailAddress = errors.New("The email address is not valid.")
	ErrInvalidPassword     = errors.New("The password is not valid.")
	ErrInvalidCredentials  = errors.New("The email address or the password is not correct.")
	ErrNotLoggedIn         = errors.New("There is no session for this request.")
	ErrAdminNotFound       = errors.New("The Admin does not exist.")
	ErrEmailTaken          = errors.New("An Admin with this email address exists.")
)
