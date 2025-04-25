package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/Axontik/comin-leave-management-service/internal/handler"
	"github.com/Axontik/comin-leave-management-service/internal/middleware"
	"github.com/Axontik/comin-leave-management-service/internal/repository"
	"github.com/Axontik/comin-leave-management-service/internal/service"
	"github.com/Axontik/comin-leave-management-service/pkg/auth"
	"github.com/Axontik/comin-leave-management-service/pkg/organization"
)

type Application struct {
	db                  *gorm.DB
	leaveTypeHandler    *handler.LeaveTypeHandler
	leaveRequestHandler *handler.LeaveRequestHandler
	leaveBalanceHandler *handler.LeaveBalanceHandler
	holidayHandler      *handler.HolidayHandler
	reportHandler       *handler.ReportHandler
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	app := &Application{}

	// Initialize database
	db, err := initDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	app.db = db

	// Initialize dependencies
	app.initializeDependencies()

	// Setup router
	router := setupRouter(app)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func initDB() (*gorm.DB, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://comin_owner:Ye5rfjcIB7FX@ep-flat-shadow-a8onelva.eastus2.azure.neon.tech/comin?sslmode=require"
	}

	// Run migrations
	m, err := migrate.New(
		"file://migrations",
		dbURL,
	)
	if err != nil {
		log.Printf("Warning: Failed to initialize migrations: %v", err)
	} else {
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Printf("Warning: Failed to run migrations: %v", err)
		}
	}

	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	return gorm.Open(postgres.Open(dbURL), config)
}

func (app *Application) initializeDependencies() {
	// Initialize repositories
	leaveRepo := repository.NewLeaveRepository(app.db)

	// Initialize services
	leaveService := service.NewLeaveService(leaveRepo)

	// Initialize handlers
	app.leaveTypeHandler = handler.NewLeaveTypeHandler(leaveService)
	app.leaveRequestHandler = handler.NewLeaveRequestHandler(leaveService)
	app.leaveBalanceHandler = handler.NewLeaveBalanceHandler(leaveService)
	app.holidayHandler = handler.NewHolidayHandler(leaveService)
	app.reportHandler = handler.NewReportHandler(leaveService)
}

func (app *Application) healthHandler(c *gin.Context) {
	// Check DB connection
	sqlDB, err := app.db.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "unhealthy", "reason": "database connection error"})
		return
	}
	err = sqlDB.Ping()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "unhealthy", "reason": "database ping failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   time.Now().UTC(),
	})
}

func (app *Application) metricsHandler(c *gin.Context) {
	// Implement your metrics logic here
	c.JSON(http.StatusOK, gin.H{
		"total_requests": 0,
		"active_users":   0,
		"response_time":  0,
	})
}

func setupRouter(app *Application) *gin.Engine {
	authServiceURL := os.Getenv("AUTH_SERVICE_URL")
	if authServiceURL == "" {
		authServiceURL = "https://comin.kaveeshagimhana.com/api/v1/auth"
	}
	authClient := auth.NewAuthClient(authServiceURL)

	orgServiceURL := os.Getenv("ORGANIZATION_SERVICE_URL")
	if orgServiceURL == "" {
		orgServiceURL = "https://comin.kaveeshagimhana.com/api/v1"
	}
	orgClient := organization.NewOrganizationClient(orgServiceURL)

	router := gin.New()

	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.ErrorHandler())
	// router.Use(middleware.RequestID())
	// router.Use(middleware.Timeout(10 * time.Second))
	// router.Use(middleware.CORS())
	// router.Use(middleware.RateLimiter(100, 1*time.Minute))

	// Health and metrics
	router.GET("/health", app.healthHandler)
	router.GET("/metrics", app.metricsHandler)

	// API routes
	api := router.Group("/api/v1/leaves")
	// api.Use(middleware.APIVersionCheck("1.0"))
	{
		// Organization-specific routes
		orgs := api.Group("/organizations/:organization_id")
		orgs.Use(organization.ValidateOrganizationAccess(authClient, orgClient))
		{
			// Leave Types
			leaveTypes := orgs.Group("/leave-types")
			{
				leaveTypes.POST("/", app.leaveTypeHandler.Create)
				leaveTypes.GET("/", app.leaveTypeHandler.List)
				leaveTypes.GET("/:id", app.leaveTypeHandler.GetByID)
				leaveTypes.PUT("/:id", app.leaveTypeHandler.Update)
				leaveTypes.DELETE("/:id", app.leaveTypeHandler.Delete)
				// leaveTypes.POST("/bulk", app.leaveTypeHandler.BulkCreate)
				// leaveTypes.GET("/stats", app.leaveTypeHandler.GetStats)
			}

			// Leave Requests
			leaveRequests := orgs.Group("/leave-requests")
			{
				leaveRequests.POST("/", app.leaveRequestHandler.Create)
				leaveRequests.GET("/", app.leaveRequestHandler.ListLeaveRequests)
				// leaveRequests.GET("/:id", app.leaveRequestHandler.GetByID)
				// leaveRequests.PUT("/:id", app.leaveRequestHandler.Update)
				// leaveRequests.DELETE("/:id", app.leaveRequestHandler.Delete)
				// leaveRequests.PUT("/:id/approve", app.leaveRequestHandler.Approve)
				// leaveRequests.PUT("/:id/reject", app.leaveRequestHandler.Reject)
				// leaveRequests.PUT("/:id/cancel", app.leaveRequestHandler.Cancel)
				leaveRequests.GET("/calendar", app.leaveRequestHandler.GetCalendarView)
				// leaveRequests.GET("/stats", app.leaveRequestHandler.GetStats)
			}

			// Leave Balances
			leaveBalances := orgs.Group("/leave-balances")
			{
				leaveBalances.GET("/", app.leaveBalanceHandler.List)
				leaveBalances.GET("/:employee_id", app.leaveBalanceHandler.GetByEmployee)
				leaveBalances.POST("/adjust", app.leaveBalanceHandler.AdjustBalance)
				leaveBalances.GET("/history/:employee_id", app.leaveBalanceHandler.GetBalanceHistory)
				leaveBalances.POST("/yearly-reset", app.leaveBalanceHandler.YearlyReset)
			}

			// Holidays
			holidays := orgs.Group("/holidays")
			{
				holidays.POST("/", app.holidayHandler.Create)
				holidays.GET("/", app.holidayHandler.List)
				holidays.PUT("/:id", app.holidayHandler.Update)
				holidays.DELETE("/:id", app.holidayHandler.Delete)
				holidays.GET("/calendar", app.holidayHandler.GetCalendarView)
			}

			// Reports
			reports := orgs.Group("/reports")
			// reports.Use(middleware.CachingMiddleware(10 * time.Minute))
			{
				reports.GET("/leave-summary", app.reportHandler.LeaveSummary)
				reports.GET("/department-analysis", app.reportHandler.DepartmentAnalysis)
				reports.GET("/monthly-trends", app.reportHandler.MonthlyTrends)
			}
		}

		// Employee-specific routes
		employees := api.Group("/employees")
		// employees.Use(auth.ValidateOrganizationAccess(authClient))
		{
			employees.GET("/:employee_id/leave-requests", app.leaveRequestHandler.ListByEmployee)
			employees.GET("/:employee_id/leave-balance", app.leaveBalanceHandler.GetByEmployee)
			employees.GET("/:employee_id/calendar", app.leaveRequestHandler.GetEmployeeCalendar)
		}
	}

	return router
}
