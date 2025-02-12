package handler

import (
	"net/http"
	"time"

	"github.com/Axontik/comin-leave-management-service/internal/domain"
	"github.com/Axontik/comin-leave-management-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type LeaveRequestHandler struct {
	leaveService service.LeaveService
}

func NewLeaveRequestHandler(leaveService service.LeaveService) *LeaveRequestHandler {
	return &LeaveRequestHandler{
		leaveService: leaveService,
	}
}

// @Summary Create leave request
// @Tags leave-requests
// @Accept json
// @Produce json
// @Success 201 {object} domain.LeaveRequest
func (h *LeaveRequestHandler) Create(c *gin.Context) {
	orgID, err := uuid.Parse(c.Param("organization_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
		return
	}

	var req domain.CreateLeaveRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert date strings to time.Time format
	req.StartDate, err = time.Parse("2006-01-02", req.StartDate.Format("2006-01-02"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start date format"})
		return
	}

	req.EndDate, err = time.Parse("2006-01-02", req.EndDate.Format("2006-01-02"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end date format"})
		return
	}

	leaveRequest, err := h.leaveService.CreateLeaveRequest(orgID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, leaveRequest)
}

// Add other leave request methods: List, GetByID, Update, Delete, Approve, Reject, Cancel

func (h *LeaveRequestHandler) GetCalendarView(c *gin.Context) {
	// Implementation for calendar view
}

func (h *LeaveRequestHandler) GetEmployeeCalendar(c *gin.Context) {
	// Implementation for employee calendar
}

func (h *LeaveRequestHandler) ListByEmployee(c *gin.Context) {
	// Implementation for listing by employee
}
