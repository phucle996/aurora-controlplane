package iam_errorx

import "errors"

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive      = errors.New("user is inactive")
)
