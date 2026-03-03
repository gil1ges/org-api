package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"org-api/internal/models"
	"org-api/internal/repository"
)

type OrgService struct {
	repo repository.DepartmentRepository
}

type DepartmentTree struct {
	Department models.Department `json:"department"`
	Employees  []models.Employee `json:"employees,omitempty"`
	Children   []DepartmentTree  `json:"children"`
}

func NewOrgService(repo repository.DepartmentRepository) *OrgService {
	return &OrgService{repo: repo}
}

func (s *OrgService) CreateDepartment(ctx context.Context, name string, parentID *int64) (*models.Department, error) {
	name = strings.TrimSpace(name)
	if err := validateName(name); err != nil {
		return nil, err
	}

	if parentID != nil {
		exists, err := s.repo.DepartmentExists(ctx, *parentID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrNotFound
		}
	}

	exists, err := s.repo.DepartmentNameExistsUnderParent(ctx, parentID, name, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("%w: department name already exists under parent", ErrConflict)
	}

	dept := &models.Department{Name: name, ParentID: parentID}
	if err := s.repo.CreateDepartment(ctx, dept); err != nil {
		return nil, err
	}

	return dept, nil
}

func (s *OrgService) CreateEmployee(
	ctx context.Context,
	departmentID int64,
	fullName string,
	position string,
	hiredAt *time.Time,
) (*models.Employee, error) {
	fullName = strings.TrimSpace(fullName)
	position = strings.TrimSpace(position)

	if err := validateText(fullName, "full_name"); err != nil {
		return nil, err
	}
	if err := validateText(position, "position"); err != nil {
		return nil, err
	}

	exists, err := s.repo.DepartmentExists(ctx, departmentID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}

	employee := &models.Employee{
		DepartmentID: departmentID,
		FullName:     fullName,
		Position:     position,
		HiredAt:      hiredAt,
	}

	if err := s.repo.CreateEmployee(ctx, employee); err != nil {
		return nil, err
	}

	return employee, nil
}

func (s *OrgService) GetDepartmentTree(
	ctx context.Context,
	departmentID int64,
	depth int,
	includeEmployees bool,
) (*DepartmentTree, error) {
	if depth < 1 || depth > 5 {
		return nil, fmt.Errorf("%w: depth must be 1..5", ErrValidation)
	}

	root, err := s.repo.GetDepartmentByID(ctx, departmentID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	tree, err := s.buildTree(ctx, *root, depth, includeEmployees)
	if err != nil {
		return nil, err
	}

	return &tree, nil
}

func (s *OrgService) UpdateDepartment(
	ctx context.Context,
	departmentID int64,
	name *string,
	parentSet bool,
	parentID *int64,
) (*models.Department, error) {
	dept, err := s.repo.GetDepartmentByID(ctx, departmentID)
	if err != nil {
		if repository.IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	newName := dept.Name
	if name != nil {
		newName = strings.TrimSpace(*name)
		if err := validateName(newName); err != nil {
			return nil, err
		}
	}

	newParent := dept.ParentID
	if parentSet {
		newParent = parentID
		if newParent != nil {
			if *newParent == departmentID {
				return nil, ErrSelfParent
			}

			exists, err := s.repo.DepartmentExists(ctx, *newParent)
			if err != nil {
				return nil, err
			}
			if !exists {
				return nil, ErrNotFound
			}

			isCycle, err := s.createsCycle(ctx, departmentID, *newParent)
			if err != nil {
				return nil, err
			}
			if isCycle {
				return nil, ErrCycle
			}
		}
	}

	exists, err := s.repo.DepartmentNameExistsUnderParent(ctx, newParent, newName, &departmentID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("%w: department name already exists under parent", ErrConflict)
	}

	dept.Name = newName
	dept.ParentID = newParent
	if err := s.repo.UpdateDepartment(ctx, dept); err != nil {
		return nil, err
	}

	return dept, nil
}

func (s *OrgService) DeleteDepartment(ctx context.Context, departmentID int64, mode string, reassignTo *int64) error {
	exists, err := s.repo.DepartmentExists(ctx, departmentID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}

	var options repository.DeleteOptions
	switch mode {
	case string(repository.DeleteModeCascade):
		options = repository.DeleteOptions{Mode: repository.DeleteModeCascade}
	case string(repository.DeleteModeReassign):
		if reassignTo == nil {
			return ErrReassignIDRequired
		}
		options = repository.DeleteOptions{Mode: repository.DeleteModeReassign, ReassignToDepartment: reassignTo}
	default:
		return ErrInvalidDeleteMode
	}

	if err := s.repo.DeleteDepartment(ctx, departmentID, options); err != nil {
		if repository.IsNotFound(err) {
			return ErrNotFound
		}
		if errors.Is(err, repository.ErrReassignToSameDepartment) {
			return fmt.Errorf("%w: reassign_to_department_id cannot be deleted department", ErrValidation)
		}
		if errors.Is(err, repository.ErrInvalidDeleteMode) {
			return ErrInvalidDeleteMode
		}
		if errors.Is(err, repository.ErrMissingReassignTarget) {
			return ErrReassignIDRequired
		}
		return err
	}

	return nil
}

func (s *OrgService) buildTree(
	ctx context.Context,
	department models.Department,
	depth int,
	includeEmployees bool,
) (DepartmentTree, error) {
	result := DepartmentTree{Department: department, Children: make([]DepartmentTree, 0)}

	if includeEmployees {
		employees, err := s.repo.ListEmployeesByDepartment(ctx, department.ID)
		if err != nil {
			return DepartmentTree{}, err
		}
		result.Employees = employees
	}

	if depth == 0 {
		return result, nil
	}

	children, err := s.repo.ListChildDepartments(ctx, department.ID)
	if err != nil {
		return DepartmentTree{}, err
	}

	for _, child := range children {
		childTree, err := s.buildTree(ctx, child, depth-1, includeEmployees)
		if err != nil {
			return DepartmentTree{}, err
		}
		result.Children = append(result.Children, childTree)
	}

	return result, nil
}

func (s *OrgService) createsCycle(ctx context.Context, departmentID int64, newParentID int64) (bool, error) {
	current := &newParentID
	for current != nil {
		if *current == departmentID {
			return true, nil
		}
		parentID, err := s.repo.GetParentID(ctx, *current)
		if err != nil {
			if repository.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		current = parentID
	}

	return false, nil
}

func validateName(name string) error {
	return validateText(name, "name")
}

func validateText(text string, field string) error {
	if text == "" {
		return fmt.Errorf("%w: %s must not be empty", ErrValidation, field)
	}

	if len([]rune(text)) > 200 {
		return fmt.Errorf("%w: %s must have length <= 200", ErrValidation, field)
	}

	return nil
}
