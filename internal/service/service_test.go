package service

import (
	"context"
	"testing"
	"time"

	"org-api/internal/models"
	"org-api/internal/repository"

	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	departments map[int64]models.Department
	employees   map[int64][]models.Employee
	deleteErr   error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		departments: map[int64]models.Department{},
		employees:   map[int64][]models.Employee{},
	}
}

func (f *fakeRepo) CreateDepartment(_ context.Context, dept *models.Department) error {
	dept.ID = int64(len(f.departments) + 1)
	dept.CreatedAt = time.Now()
	f.departments[dept.ID] = *dept
	return nil
}

func (f *fakeRepo) GetDepartmentByID(_ context.Context, id int64) (*models.Department, error) {
	dept, ok := f.departments[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &dept, nil
}

func (f *fakeRepo) UpdateDepartment(_ context.Context, dept *models.Department) error {
	f.departments[dept.ID] = *dept
	return nil
}

func (f *fakeRepo) DeleteDepartment(_ context.Context, _ int64, _ repository.DeleteOptions) error {
	return f.deleteErr
}

func (f *fakeRepo) ListChildDepartments(_ context.Context, parentID int64) ([]models.Department, error) {
	children := make([]models.Department, 0)
	for _, dept := range f.departments {
		if dept.ParentID != nil && *dept.ParentID == parentID {
			children = append(children, dept)
		}
	}
	return children, nil
}

func (f *fakeRepo) DepartmentExists(_ context.Context, id int64) (bool, error) {
	_, ok := f.departments[id]
	return ok, nil
}

func (f *fakeRepo) DepartmentNameExistsUnderParent(_ context.Context, _ *int64, _ string, _ *int64) (bool, error) {
	return false, nil
}

func (f *fakeRepo) GetParentID(_ context.Context, id int64) (*int64, error) {
	dept, ok := f.departments[id]
	if !ok {
		return nil, ErrNotFound
	}
	return dept.ParentID, nil
}

func (f *fakeRepo) CreateEmployee(_ context.Context, employee *models.Employee) error {
	employee.ID = int64(len(f.employees[employee.DepartmentID]) + 1)
	employee.CreatedAt = time.Now()
	f.employees[employee.DepartmentID] = append(f.employees[employee.DepartmentID], *employee)
	return nil
}

func (f *fakeRepo) ListEmployeesByDepartment(_ context.Context, departmentID int64) ([]models.Employee, error) {
	return f.employees[departmentID], nil
}

func TestUpdateDepartmentDetectsCycle(t *testing.T) {
	repo := newFakeRepo()
	rootID := int64(1)
	childID := int64(2)
	grandChildID := int64(3)

	repo.departments[rootID] = models.Department{ID: rootID, Name: "Root"}
	repo.departments[childID] = models.Department{ID: childID, Name: "Child", ParentID: &rootID}
	repo.departments[grandChildID] = models.Department{ID: grandChildID, Name: "GrandChild", ParentID: &childID}

	svc := NewOrgService(repo)
	_, err := svc.UpdateDepartment(context.Background(), rootID, nil, true, &grandChildID)
	require.ErrorIs(t, err, ErrCycle)
}

func TestGetDepartmentTreeRespectsDepth(t *testing.T) {
	repo := newFakeRepo()
	rootID := int64(1)
	childID := int64(2)
	grandChildID := int64(3)

	repo.departments[rootID] = models.Department{ID: rootID, Name: "Root"}
	repo.departments[childID] = models.Department{ID: childID, Name: "Child", ParentID: &rootID}
	repo.departments[grandChildID] = models.Department{ID: grandChildID, Name: "Grand", ParentID: &childID}

	svc := NewOrgService(repo)
	tree, err := svc.GetDepartmentTree(context.Background(), rootID, 1, false)
	require.NoError(t, err)
	require.Len(t, tree.Children, 1)
	require.Empty(t, tree.Children[0].Children)
}

func TestDeleteDepartmentMapsSameReassignDepartmentToValidationError(t *testing.T) {
	repo := newFakeRepo()
	deptID := int64(1)
	repo.departments[deptID] = models.Department{ID: deptID, Name: "Root"}
	repo.deleteErr = repository.ErrReassignToSameDepartment

	svc := NewOrgService(repo)
	err := svc.DeleteDepartment(context.Background(), deptID, string(repository.DeleteModeReassign), &deptID)

	require.ErrorIs(t, err, ErrValidation)
	require.Contains(t, err.Error(), "reassign_to_department_id cannot be deleted department")
}

func TestGetDepartmentTreeRejectsDepthBelowOne(t *testing.T) {
	repo := newFakeRepo()
	rootID := int64(1)
	repo.departments[rootID] = models.Department{ID: rootID, Name: "Root"}

	svc := NewOrgService(repo)
	_, err := svc.GetDepartmentTree(context.Background(), rootID, 0, false)

	require.ErrorIs(t, err, ErrValidation)
	require.Contains(t, err.Error(), "depth must be 1..5")
}
