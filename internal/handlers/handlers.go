package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"org-api/internal/httpx"
	"org-api/internal/service"
)

type Handler struct {
	svc *service.OrgService
}

type createDepartmentRequest struct {
	Name     string `json:"name"`
	ParentID *int64 `json:"parent_id"`
}

type createEmployeeRequest struct {
	FullName string  `json:"full_name"`
	Position string  `json:"position"`
	HiredAt  *string `json:"hired_at"`
}

type patchDepartmentRequest struct {
	Name      *string
	ParentSet bool
	ParentID  *int64
}

func New(svc *service.OrgService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /departments/", h.createDepartment)
	mux.HandleFunc("POST /departments/{id}/employees/", h.createEmployee)
	mux.HandleFunc("GET /departments/{id}", h.getDepartment)
	mux.HandleFunc("PATCH /departments/{id}", h.patchDepartment)
	mux.HandleFunc("DELETE /departments/{id}", h.deleteDepartment)
}

func (h *Handler) createDepartment(w http.ResponseWriter, r *http.Request) {
	var req createDepartmentRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	dept, err := h.svc.CreateDepartment(r.Context(), req.Name, req.ParentID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, dept)
}

func (h *Handler) createEmployee(w http.ResponseWriter, r *http.Request) {
	departmentID, err := parsePathID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	var req createEmployeeRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	hiredAt, err := parseDate(req.HiredAt)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	employee, err := h.svc.CreateEmployee(r.Context(), departmentID, req.FullName, req.Position, hiredAt)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, employee)
}

func (h *Handler) getDepartment(w http.ResponseWriter, r *http.Request) {
	departmentID, err := parsePathID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	depth := 1
	if value := r.URL.Query().Get("depth"); value != "" {
		parsedDepth, err := strconv.Atoi(value)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "depth must be integer")
			return
		}
		depth = parsedDepth
	}

	includeEmployees := true
	if value := r.URL.Query().Get("include_employees"); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "include_employees must be bool")
			return
		}
		includeEmployees = parsed
	}

	tree, err := h.svc.GetDepartmentTree(r.Context(), departmentID, depth, includeEmployees)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, tree)
}

func (h *Handler) patchDepartment(w http.ResponseWriter, r *http.Request) {
	departmentID, err := parsePathID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	req, err := decodePatchDepartmentRequest(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	department, err := h.svc.UpdateDepartment(r.Context(), departmentID, req.Name, req.ParentSet, req.ParentID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, department)
}

func (h *Handler) deleteDepartment(w http.ResponseWriter, r *http.Request) {
	departmentID, err := parsePathID(r, "id")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	mode := r.URL.Query().Get("mode")
	reassignToRaw := r.URL.Query().Get("reassign_to_department_id")

	var reassignTo *int64
	if reassignToRaw != "" {
		parsed, err := strconv.ParseInt(reassignToRaw, 10, 64)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid reassign_to_department_id")
			return
		}
		reassignTo = &parsed
	}

	if err := h.svc.DeleteDepartment(r.Context(), departmentID, mode, reassignTo); err != nil {
		h.writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	switch {
	case errors.Is(err, service.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrConflict), errors.Is(err, service.ErrCycle):
		httpx.WriteError(w, http.StatusConflict, err.Error())
	case errors.Is(err, service.ErrValidation),
		errors.Is(err, service.ErrSelfParent),
		errors.Is(err, service.ErrInvalidDeleteMode),
		errors.Is(err, service.ErrReassignIDRequired):
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}

func parsePathID(r *http.Request, key string) (int64, error) {
	idRaw := r.PathValue(key)
	id, err := strconv.ParseInt(idRaw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid %s", key)
	}

	return id, nil
}

func parseDate(value *string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}

	parsed, err := time.Parse("2006-01-02", *value)
	if err != nil {
		return nil, fmt.Errorf("hired_at must be in YYYY-MM-DD format")
	}

	return &parsed, nil
}

func decodePatchDepartmentRequest(r *http.Request) (patchDepartmentRequest, error) {
	var raw map[string]json.RawMessage
	if err := httpx.DecodeJSON(r, &raw); err != nil {
		return patchDepartmentRequest{}, err
	}

	request := patchDepartmentRequest{}
	for key, value := range raw {
		switch key {
		case "name":
			if string(value) == "null" {
				return patchDepartmentRequest{}, fmt.Errorf("name cannot be null")
			}
			var name string
			if err := json.Unmarshal(value, &name); err != nil {
				return patchDepartmentRequest{}, fmt.Errorf("name must be string")
			}
			request.Name = &name
		case "parent_id":
			request.ParentSet = true
			if string(value) == "null" {
				request.ParentID = nil
				continue
			}

			var parentID int64
			if err := json.Unmarshal(value, &parentID); err != nil || parentID <= 0 {
				return patchDepartmentRequest{}, fmt.Errorf("parent_id must be positive integer or null")
			}
			request.ParentID = &parentID
		default:
			return patchDepartmentRequest{}, fmt.Errorf("unknown field: %s", key)
		}
	}

	return request, nil
}
