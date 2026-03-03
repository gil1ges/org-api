package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodePatchDepartmentRequestParentNull(t *testing.T) {
	req := httptest.NewRequest("PATCH", "/departments/1", strings.NewReader(`{"parent_id":null,"name":" Platform "}`))

	payload, err := decodePatchDepartmentRequest(req)
	require.NoError(t, err)
	require.True(t, payload.ParentSet)
	require.Nil(t, payload.ParentID)
	require.NotNil(t, payload.Name)
	require.Equal(t, " Platform ", *payload.Name)
}

func TestParseDateInvalidFormat(t *testing.T) {
	value := "03-03-2026"

	result, err := parseDate(&value)
	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "YYYY-MM-DD")
}

func TestCreateDepartmentInvalidJSONReturnsBadRequest(t *testing.T) {
	handler := &Handler{}
	req := httptest.NewRequest(http.MethodPost, "/departments/", strings.NewReader(`{"name":`))
	rec := httptest.NewRecorder()

	handler.createDepartment(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "invalid json body")
}

func TestCreateEmployeeInvalidJSONReturnsBadRequest(t *testing.T) {
	handler := &Handler{}
	req := httptest.NewRequest(http.MethodPost, "/departments/1/employees/", strings.NewReader(`{"full_name":`))
	req.SetPathValue("id", "1")
	rec := httptest.NewRecorder()

	handler.createEmployee(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "invalid json body")
}
