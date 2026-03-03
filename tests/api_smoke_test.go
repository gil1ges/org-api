package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"gorm.io/gorm"
	"org-api/internal/handlers"
	"org-api/internal/models"
	"org-api/internal/repository"
	"org-api/internal/service"
)

type memoryRepo struct {
	nextDeptID int64
	nextEmpID  int64
	depts      map[int64]models.Department
	emps       map[int64][]models.Employee
}

func newMemoryRepo() *memoryRepo {
	return &memoryRepo{
		nextDeptID: 1,
		nextEmpID:  1,
		depts:      map[int64]models.Department{},
		emps:       map[int64][]models.Employee{},
	}
}

func (m *memoryRepo) CreateDepartment(_ context.Context, dept *models.Department) error {
	dept.ID = m.nextDeptID
	m.nextDeptID++
	dept.CreatedAt = time.Now()
	m.depts[dept.ID] = *dept
	return nil
}

func (m *memoryRepo) GetDepartmentByID(_ context.Context, id int64) (*models.Department, error) {
	dept, ok := m.depts[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return &dept, nil
}

func (m *memoryRepo) UpdateDepartment(_ context.Context, dept *models.Department) error {
	m.depts[dept.ID] = *dept
	return nil
}

func (m *memoryRepo) DeleteDepartment(_ context.Context, id int64, _ repository.DeleteOptions) error {
	delete(m.depts, id)
	delete(m.emps, id)
	return nil
}

func (m *memoryRepo) ListChildDepartments(_ context.Context, parentID int64) ([]models.Department, error) {
	out := make([]models.Department, 0)
	for _, dept := range m.depts {
		if dept.ParentID != nil && *dept.ParentID == parentID {
			out = append(out, dept)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (m *memoryRepo) DepartmentExists(_ context.Context, id int64) (bool, error) {
	_, ok := m.depts[id]
	return ok, nil
}

func (m *memoryRepo) DepartmentNameExistsUnderParent(_ context.Context, parentID *int64, name string, excludeID *int64) (bool, error) {
	for id, dept := range m.depts {
		if excludeID != nil && *excludeID == id {
			continue
		}
		if dept.Name != name {
			continue
		}
		if dept.ParentID == nil && parentID == nil {
			return true, nil
		}
		if dept.ParentID != nil && parentID != nil && *dept.ParentID == *parentID {
			return true, nil
		}
	}
	return false, nil
}

func (m *memoryRepo) GetParentID(_ context.Context, id int64) (*int64, error) {
	dept, ok := m.depts[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return dept.ParentID, nil
}

func (m *memoryRepo) CreateEmployee(_ context.Context, employee *models.Employee) error {
	employee.ID = m.nextEmpID
	m.nextEmpID++
	employee.CreatedAt = time.Now()
	m.emps[employee.DepartmentID] = append(m.emps[employee.DepartmentID], *employee)
	return nil
}

func (m *memoryRepo) ListEmployeesByDepartment(_ context.Context, departmentID int64) ([]models.Employee, error) {
	out := append([]models.Employee(nil), m.emps[departmentID]...)
	sort.Slice(out, func(i, j int) bool { return out[i].FullName < out[j].FullName })
	return out, nil
}

func TestAPIFlow_CreateDepartmentEmployeeAndGetTree(t *testing.T) {
	repo := newMemoryRepo()
	svc := service.NewOrgService(repo)
	h := handlers.New(svc)
	mux := http.NewServeMux()
	h.Register(mux)

	status, body := doJSONRequest(t, mux, http.MethodPost, "/departments/", `{"name":"Platform"}`)
	if status != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusCreated, status, body)
	}

	status, body = doJSONRequest(t, mux, http.MethodPost, "/departments/1/employees/", `{"full_name":"Ivan Ivanov","position":"Go Developer","hired_at":"2026-03-03"}`)
	if status != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusCreated, status, body)
	}

	req := httptest.NewRequest(http.MethodGet, "/departments/1?depth=1&include_employees=true", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var tree struct {
		Department models.Department `json:"department"`
		Employees  []models.Employee `json:"employees"`
		Children   []json.RawMessage `json:"children"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &tree); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if tree.Department.ID != 1 {
		t.Fatalf("expected department id 1, got %d", tree.Department.ID)
	}
	if len(tree.Employees) != 1 {
		t.Fatalf("expected 1 employee, got %d", len(tree.Employees))
	}
	if tree.Employees[0].FullName != "Ivan Ivanov" {
		t.Fatalf("unexpected employee name: %s", tree.Employees[0].FullName)
	}
}

func doJSONRequest(t *testing.T, handler http.Handler, method, path, body string) (int, string) {
	t.Helper()

	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	return resp.Code, string(data)
}
