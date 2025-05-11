package domain
 
import (
    "errors"
    "time"
 
    "github.com/google/uuid"
    "gorm.io/gorm"
)
 
// LeaveType represents different types of leave (vacation, sick, etc.)
type LeaveType struct {
    Base
    ID                uuid.UUID `json:"id"`
    OrganizationID    uuid.UUID `json:"organization_id" gorm:"type:uuid;not null" binding:"required"`
    Name              string    `json:"name" gorm:"not null" binding:"required,min=2,max=100"`
    Description       string    `json:"description" binding:"max=500"`
    Color             string    `json:"color" gorm:"type:varchar(7)" binding:"required,hexcolor"`
    DefaultDays       int       `json:"default_days" binding:"required,min=0,max=365"`
    IsPaid            bool      `json:"is_paid" gorm:"default:true"`
    RequiresApproval  bool      `json:"requires_approval" gorm:"default:true"`
    MinDaysNotice     int       `json:"min_days_notice" gorm:"default:0" binding:"min=0"`
    MaxDaysPerRequest int       `json:"max_days_per_request" binding:"required,min=1,max=365"`
}
 
// LeaveBalance tracks employee's leave balance
type LeaveBalance struct {
    Base
    ID             uuid.UUID  `json:"id"`
    OrganizationID uuid.UUID  `json:"organization_id" gorm:"type:uuid;not null"`
    EmployeeID     uuid.UUID  `json:"employee_id" gorm:"type:uuid;not null"`
    LeaveTypeID    uuid.UUID  `json:"leave_type_id" gorm:"type:uuid"`
    Year           int        `json:"year" gorm:"not null"`
    TotalDays      float64    `json:"total_days" gorm:"type:decimal(5,2);not null"`
    UsedDays       float64    `json:"used_days" gorm:"type:decimal(5,2);default:0"`
    PendingDays    float64    `json:"pending_days" gorm:"type:decimal(5,2);default:0"`
    RemainingDays  float64    `json:"remaining_days" gorm:"type:decimal(5,2)"`
    LeaveType      *LeaveType `json:"leave_type,omitempty" gorm:"foreignKey:LeaveTypeID"`
}
 
type LeaveRequest struct {
    Base
    OrganizationID uuid.UUID  `json:"organization_id" gorm:"type:uuid;not null" binding:"required"`
    EmployeeID     uuid.UUID  `json:"employee_id" gorm:"type:uuid;not null" binding:"required"`
    LeaveTypeID    uuid.UUID  `json:"leave_type_id" gorm:"type:uuid" binding:"required"`
    StartDate      time.Time  `json:"start_date" gorm:"not null" binding:"required"`
    EndDate        time.Time  `json:"end_date" gorm:"not null" binding:"required,gtefield=StartDate"`
    Days           float64    `json:"days" gorm:"type:decimal(5,2);not null"`
    Status         string     `json:"status" gorm:"default:'pending'" binding:"required,oneof=pending approved rejected cancelled"`
    Reason         string     `json:"reason" binding:"required,min=5,max=500"`
    Comments       string     `json:"comments" binding:"max=1000"`
    ApprovedBy     *uuid.UUID `json:"approved_by,omitempty" gorm:"type:uuid"`
    ApprovedAt     *time.Time `json:"approved_at,omitempty"`
    LeaveType      *LeaveType `json:"leave_type,omitempty" gorm:"foreignKey:LeaveTypeID"`
}
 
type LeaveRequestHistory struct {
    Base
    LeaveRequestID uuid.UUID `json:"leave_request_id" gorm:"type:uuid"`
    Action         string    `json:"action" gorm:"not null"`
    Status         string    `json:"status" gorm:"not null"`
    Comments       string    `json:"comments"`
    PerformedBy    uuid.UUID `json:"performed_by" gorm:"type:uuid;not null"`
}
 
type Holiday struct {
    Base
    OrganizationID uuid.UUID `json:"organization_id" gorm:"type:uuid;not null"`
    Name           string    `json:"name" gorm:"not null"`
    Date           time.Time `json:"date" gorm:"not null"`
    Type           string    `json:"type" gorm:"not null"` // public, company, optional
}
 
type CreateLeaveTypeRequest struct {
    Name              string `json:"name" binding:"required"`
    Description       string `json:"description"`
    Color             string `json:"color" binding:"required"`
    DefaultDays       int    `json:"default_days" binding:"required"`
    IsPaid            bool   `json:"is_paid"`
    RequiresApproval  bool   `json:"requires_approval"`
    MinDaysNotice     int    `json:"min_days_notice"`
    MaxDaysPerRequest int    `json:"max_days_per_request"`
}
 
type ListLeaveTypesParams struct {
    Page             int
    PageSize         int
    Name             string
    IsPaid           *bool
    RequiresApproval *bool
}
 
type ListLeaveRequestsParams struct {
    Page     int
    PageSize int
    Status   string
}
 
type CreateLeaveRequest struct {
    OrganizationID uuid.UUID `json:"organization_id" gorm:"type:uuid;not null" binding:"required"`
    EmployeeID     uuid.UUID `json:"employee_id" binding:"required"`
    LeaveTypeID    uuid.UUID `json:"leave_type_id" binding:"required"`
    StartDate      time.Time `json:"start_date" binding:"required"`
    EndDate        time.Time `json:"end_date" binding:"required"`
    TotalDays      float64   `json:"total_days" binding:"required"`
    Status         string    `json:"status" binding:"required,oneof=pending approved rejected cancelled"`
    Reason         string    `json:"reason" binding:"required"`
    Comment        string    `json:"comment"`
}
 
type UpdateLeaveRequest struct {
    Status   string `json:"status" binding:"required,oneof=approved rejected cancelled"`
    Comments string `json:"comments"`
}
 
type CreateHolidayRequest struct {
    Name string    `json:"name" binding:"required"`
    Date time.Time `json:"date" binding:"required"`
    Type string    `json:"type" binding:"required,oneof=public company optional"`
}
 
type LeaveBalanceResponse struct {
    LeaveType     string  `json:"leave_type"`
    TotalDays     float64 `json:"total_days"`
    UsedDays      float64 `json:"used_days"`
    PendingDays   float64 `json:"pending_days"`
    RemainingDays float64 `json:"remaining_days"`
}
 
// Constants
const (
    LeaveStatusPending   = "pending"
    LeaveStatusApproved  = "approved"
    LeaveStatusRejected  = "rejected"
    LeaveStatusCancelled = "cancelled"
 
    HolidayTypePublic   = "public"
    HolidayTypeCompany  = "company"
    HolidayTypeOptional = "optional"
)
 
func (l *LeaveRequest) BeforeCreate(tx *gorm.DB) error {
    if l.StartDate.After(l.EndDate) {
        return errors.New("start date must be before end date")
    }
 
    l.Days = calculateWorkingDays(l.StartDate, l.EndDate)
    return nil
}
 
func (l *LeaveRequest) BeforeUpdate(tx *gorm.DB) error {
    if l.Status == LeaveStatusApproved && l.ApprovedBy == nil {
        return errors.New("approved_by is required when status is approved")
    }
    return nil
}
 
func (l *LeaveRequest) CanCancel() bool {
    return l.Status == LeaveStatusPending ||
        (l.Status == LeaveStatusApproved && l.StartDate.After(time.Now()))
}
 
func (l *LeaveRequest) CanApprove() bool {
    return l.Status == LeaveStatusPending
}
 
func calculateWorkingDays(start, end time.Time) float64 {
    var days float64
    current := start
 
    for current.Before(end) || current.Equal(end) {
        if current.Weekday() != time.Saturday && current.Weekday() != time.Sunday {
            days++
        }
        current = current.AddDate(0, 0, 1)
    }
 
    return days
}
 