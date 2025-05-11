package handler
 
import (
    "net/http"
    "strconv"
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
 
    var req domain.CreateLeaveRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
 
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
 
func (h *LeaveRequestHandler) UpdateLeaveRequest(c *gin.Context) {
    orgID, err := uuid.Parse(c.Param("organization_id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
        return
    }
 
    requestID, err := uuid.Parse(c.Param("req_id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid leave request id"})
        return
    }
 
    userIDStr, exists := c.Get("user_id")
    if !exists {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "user ID not found in context"})
        return
    }
 
    userIDString, ok := userIDStr.(string)
    if !ok {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID format in context"})
        return
    }
 
    userID, err := uuid.Parse(userIDString)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse user ID"})
        return
    }
 
    var updateReq domain.UpdateLeaveRequest
    if err := c.BindJSON(&updateReq); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
        return
    }
 
    leaveRequest := domain.LeaveRequest{
        Status:   updateReq.Status,
        Comments: updateReq.Comments,
    }
 
    if err := h.leaveService.UpdateLeaveRequest(orgID, requestID, &leaveRequest, userID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
 
    c.JSON(http.StatusOK, gin.H{"message": "leave request updated successfully"})
}
 
// @Summary List leave requests
// @Tags leave-requests
// @Produce json
// @Param organization_id path string true "Organization ID"
// @Param page query int false "Page number"
// @Param page_size query int false "Page size"
// @Param status query string false "Leave request status"
// @Success 200 {object} domain.LeaveRequest
func (h *LeaveRequestHandler) ListLeaveRequests(c *gin.Context) {
    orgID, err := uuid.Parse(c.Param("organization_id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid organization id"})
        return
    }
 
    params := &domain.ListLeaveRequestsParams{
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
 
    // if status := c.Query("status"); status != "" {
    //  params.Status = status
    // }
 
    leaveRequests, err := h.leaveService.ListLeaveRequests(orgID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
 
    c.JSON(http.StatusOK, leaveRequests)
}
 
func (h *LeaveRequestHandler) GetCalendarView(c *gin.Context) {
    // Implementation for calendar view
}
 
func (h *LeaveRequestHandler) GetEmployeeCalendar(c *gin.Context) {
    // Implementation for employee calendar
}
 
func (h *LeaveRequestHandler) ListByEmployee(c *gin.Context) {
    // Implementation for listing by employee
}
 
 