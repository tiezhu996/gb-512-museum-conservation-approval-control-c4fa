package service

import "errors"

var (
	ErrInvalidTransition = errors.New("requested status transition is not allowed")
	ErrInvalidInput      = errors.New("business input validation failed")
	ErrUnauthorized      = errors.New("invalid username or password")
	ErrInactiveUser      = errors.New("user account is inactive")
	ErrApprovalLocked    = errors.New("approval fields are immutable after review starts")
	ErrReviewerRequired  = errors.New("a reviewer or administrator must approve or reject")
	// Gate errors: entering review requires a resolvable plan and a verified
	// material test; approving requires the frozen snapshot to still hold.
	ErrGatePlanMissing   = errors.New("approval gate: no treatment plan matches the related code")
	ErrGateTestMissing   = errors.New("approval gate: plan has no verified material test")
	ErrGateSnapshotStale = errors.New("approval gate: frozen material test no longer holds")
)
