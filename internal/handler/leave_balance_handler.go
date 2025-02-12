package handler

import (
	"net/http"
	"time"

	"github.com/Axontik/comin-leave-management-service/internal/domain"
	"github.com/Axontik/comin-leave-management-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type LeaveBalanceHandler struct {
	leaveService service.LeaveService
}

func NewLeaveBalanceHandler(leaveService service.LeaveService) *LeaveBalanceHandler {
	return &LeaveBalanceHandler{
		leaveService: leaveService,
	}
}

func (h *LeaveBalanceHandler) List(c *gin.Context) {
	// Implementation
}

func (h *LeaveBalanceHandler) GetByEmployee(c *gin.Context) {
	// Implementation
}

func (h *LeaveBalanceHandler) AdjustBalance(c *gin.Context) {
	var adjustmentRequest struct {
		EmployeeID  uuid.UUID `json:"employee_id" binding:"required"`
		LeaveTypeID uuid.UUID `json:"leave_type_id"`
		Adjustment  float64   `json:"adjustment"`
		Reason      string    `json:"reason"`
	}

	if err := c.ShouldBindJSON(&adjustmentRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	balance, err := h.leaveService.GetLeaveBalance(adjustmentRequest.EmployeeID, adjustmentRequest.LeaveTypeID, time.Now().UTC().Year())
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Leave balance not found"})
		return
	}

	newTotalDays := balance.TotalDays + adjustmentRequest.Adjustment

	if err := h.leaveService.UpdateLeaveBalance(&domain.LeaveBalance{
		ID:        balance.ID,
		TotalDays: newTotalDays,
		UsedDays:    balance.UsedDays,
		PendingDays: balance.PendingDays,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update leave balance"})
		return
	}

	if err := h.leaveService.CreateBalanceAdjustment(&domain.LeaveBalanceAdjustment{
		LeaveBalanceID: balance.ID,
		Adjustment:     adjustmentRequest.Adjustment,
		Reason:         adjustmentRequest.Reason,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create balance adjustment record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Leave balance adjusted successfully"})
}

func (h *LeaveBalanceHandler) GetBalanceHistory(c *gin.Context) {
	// Implementation
}

func (h *LeaveBalanceHandler) YearlyReset(c *gin.Context) {
	// Implementation
}
