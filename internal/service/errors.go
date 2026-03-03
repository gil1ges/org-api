package service

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrValidation         = errors.New("validation error")
	ErrConflict           = errors.New("conflict")
	ErrSelfParent         = errors.New("department cannot be parent of itself")
	ErrCycle              = errors.New("department cycle detected")
	ErrInvalidDeleteMode  = errors.New("invalid delete mode")
	ErrReassignIDRequired = errors.New("reassign_to_department_id is required")
)
