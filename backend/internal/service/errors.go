package service

import "errors"

var (
	ErrInvalidTransition = errors.New("requested status transition is not allowed")
	ErrInvalidInput      = errors.New("business input validation failed")
	ErrUnauthorized      = errors.New("invalid username or password")
	ErrInactiveUser      = errors.New("user account is inactive")
	ErrApprovalLocked    = errors.New("approval fields are immutable after review starts")
	ErrReviewerRequired  = errors.New("a reviewer or administrator must approve or reject")
)
