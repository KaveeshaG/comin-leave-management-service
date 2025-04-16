package handler

import (
	"net/http"
	"strconv"

	"github.com/Axontik/comin-leave-management-service/internal/domain"
	"github.com/Axontik/comin-leave-management-service/internal/errors"
	"github.com/Axontik/comin-leave-management-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type LeaveTypeHandler struct {
	leaveService service.LeaveService
}

func NewLeaveTypeHandler(leaveService service.LeaveService) *LeaveTypeHandler {
	return &LeaveTypeHandler{
		leaveService: leaveService,
	}
}

// @Summary Create a new leave type
// @Description Create a new leave type for an organization
// @Tags leave-types
// @Accept json
// @Produce json
// @Param organization_id path string true "Organization ID"
// @Param leave_type body domain.CreateLeaveTypeRequest true "Leave Type Details"
// @Success 201 {object} domain.LeaveType
// @Failure 400 {object} ErrorResponse
// @Router /organizations/{organization_id}/leave-types [post]
func (h *LeaveTypeHandler) Create(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("organization_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	var req domain.CreateLeaveTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	leaveType := &domain.LeaveType{
		OrganizationID:    orgID,
		Name:              req.Name,
		Description:       req.Description,
		Color:             req.Color,
		DefaultDays:       req.DefaultDays,
		IsPaid:            req.IsPaid,
		RequiresApproval:  req.RequiresApproval,
		MinDaysNotice:     req.MinDaysNotice,
		MaxDaysPerRequest: req.MaxDaysPerRequest,
	}

	if err := h.leaveService.CreateLeaveType(leaveType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, leaveType)
}

// @Summary List leave types
// @Description Get all leave types for an organization
// @Tags leave-types
// @Produce json
// @Param organization_id path string true "Organization ID"
// @Param page query integer false "Page number"
// @Param page_size query integer false "Page size"
// @Param name query string false "Filter by name"
// @Param is_paid query boolean false "Filter by paid status"
// @Success 200 {array} domain.LeaveType
// @Router /organizations/{organization_id}/leave-types [get]
func (h *LeaveTypeHandler) List(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("organization_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	// Parse query parameters
	params := &domain.ListLeaveTypesParams{
		Page:     1,
		PageSize: 10,
	}

	if page := c.Query("page"); page != "" {
		if pageNum, err := strconv.Atoi(page); err == nil {
			params.Page = pageNum
		}
	}

	if pageSize := c.Query("page_size"); pageSize != "" {
		if size, err := strconv.Atoi(pageSize); err == nil {
			params.PageSize = size
		}
	}

	params.Name = c.Query("name")

	if isPaid := c.Query("is_paid"); isPaid != "" {
		paid, err := strconv.ParseBool(isPaid)
		if err == nil {
			params.IsPaid = &paid
		}
	}

	leaveTypes, _, err := h.leaveService.ListLeaveTypes(orgID, params)
	if err != nil {
		c.Error(errors.NewInternalServerError("failed to fetch leave types"))
		return
	}

	c.JSON(http.StatusOK, leaveTypes)
}

// @Summary Get leave type by ID
// @Tags leave-types
// @Produce json
// @Param organization_id path string true "Organization ID"
// @Param id path string true "Leave Type ID"
// @Success 200 {object} domain.LeaveType
// @Router /organizations/{organization_id}/leave-types/{id} [get]
func (h *LeaveTypeHandler) GetByID(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("organization_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid leave type id"})
		return
	}

	leaveType, err := h.leaveService.GetLeaveType(orgID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "leave type not found"})
		return
	}

	c.JSON(http.StatusOK, leaveType)
}

// @Summary Update leave type
// @Tags leave-types
// @Accept json
// @Produce json
// @Param organization_id path string true "Organization ID"
// @Param id path string true "Leave Type ID"
// @Param leave_type body domain.CreateLeaveTypeRequest true "Leave Type Details"
// @Success 200 {object} domain.LeaveType
// @Router /organizations/{organization_id}/leave-types/{id} [put]
func (h *LeaveTypeHandler) Update(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("organization_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid leave type id"})
		return
	}

	var req domain.CreateLeaveTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	leaveType := &domain.LeaveType{
		ID:                id,
		OrganizationID:    orgID,
		Name:              req.Name,
		Description:       req.Description,
		Color:             req.Color,
		DefaultDays:       req.DefaultDays,
		IsPaid:            req.IsPaid,
		RequiresApproval:  req.RequiresApproval,
		MinDaysNotice:     req.MinDaysNotice,
		MaxDaysPerRequest: req.MaxDaysPerRequest,
	}

	if err := h.leaveService.UpdateLeaveType(leaveType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, leaveType)
}

// @Summary Delete leave type
// @Tags leave-types
// @Param organization_id path string true "Organization ID"
// @Param id path string true "Leave Type ID"
// @Success 204 "No Content"
// @Router /organizations/{organization_id}/leave-types/{id} [delete]
func (h *LeaveTypeHandler) Delete(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("organization_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid leave type id"})
		return
	}

	if err := h.leaveService.DeleteLeaveType(orgID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
