package service

import "errors"

var (
	ErrInvalidTransition = errors.New("requested status transition is not allowed")
	ErrInvalidInput      = errors.New("business input validation failed")
	ErrUnauthorized      = errors.New("invalid username or password")
	ErrInactiveUser      = errors.New("user account is inactive")
	ErrApprovalLocked    = errors.New("approval fields are immutable after review starts")
	ErrReviewerRequired  = errors.New("a reviewer or administrator must approve or reject")

	// 检测快照门禁的三类拒绝：提交待复核缺方案/缺有效检测；批准时快照与现状漂移。
	// 错误文本由审批门禁服务填充具体原因，处理器统一映射为 422 business_rule。
	ErrGatePlanMissing      = errors.New("gate blocked: related treatment plan was not found")
	ErrGateTestMissing      = errors.New("gate blocked: no verified material test for the related treatment plan")
	ErrGateSnapshotMismatch = errors.New("gate blocked: frozen material test snapshot no longer matches")
)
