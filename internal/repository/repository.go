package repository

import (
	"context"
	"errors"

	"org-api/internal/models"
)

type DeleteMode string

const (
	DeleteModeCascade  DeleteMode = "cascade"
	DeleteModeReassign DeleteMode = "reassign"
)

var (
	ErrInvalidDeleteMode        = errors.New("invalid delete mode")
	ErrMissingReassignTarget    = errors.New("missing reassign department")
	ErrReassignToSameDepartment = errors.New("cannot reassign to the same department")
)

type DeleteOptions struct {
	Mode                 DeleteMode
	ReassignToDepartment *int64
}

type DepartmentRepository interface {
	CreateDepartment(ctx context.Context, dept *models.Department) error
	GetDepartmentByID(ctx context.Context, id int64) (*models.Department, error)
	UpdateDepartment(ctx context.Context, dept *models.Department) error
	DeleteDepartment(ctx context.Context, id int64, options DeleteOptions) error
	ListChildDepartments(ctx context.Context, parentID int64) ([]models.Department, error)
	DepartmentExists(ctx context.Context, id int64) (bool, error)
	DepartmentNameExistsUnderParent(ctx context.Context, parentID *int64, name string, excludeID *int64) (bool, error)
	GetParentID(ctx context.Context, id int64) (*int64, error)
	CreateEmployee(ctx context.Context, employee *models.Employee) error
	ListEmployeesByDepartment(ctx context.Context, departmentID int64) ([]models.Employee, error)
}
