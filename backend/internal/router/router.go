package router

import (
	"log/slog"
	"net/http"

	"github.com/blueship581/museum-conservation-approval-control/backend/internal/config"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/handler"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/middleware"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/repository"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func New(cfg config.Config, db *gorm.DB, redisClient *redis.Client, logger *slog.Logger) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(middleware.RequestContext(logger))
	if len(cfg.TrustedProxies) > 0 {
		_ = engine.SetTrustedProxies(cfg.TrustedProxies)
	} else {
		_ = engine.SetTrustedProxies(nil)
	}

	securityRepository := repository.NewSecurityRepository(db)
	securityService := service.NewSecurityService(securityRepository, cfg)
	artifactRepository := repository.NewArtifactRepository(db)
	treatmentPlanRepository := repository.NewTreatmentPlanRepository(db)
	materialTestRepository := repository.NewMaterialTestRepository(db)
	stageApprovalRepository := repository.NewStageApprovalRepository(db)
	artifactService := service.NewArtifactService(artifactRepository, securityService)
	treatmentPlanService := service.NewTreatmentPlanService(treatmentPlanRepository, securityService)
	materialTestService := service.NewMaterialTestService(materialTestRepository, securityService)
	stageApprovalService := service.NewStageApprovalService(stageApprovalRepository, treatmentPlanRepository, materialTestRepository, securityService)
	artifactHandler := handler.NewArtifactHandler(artifactService)
	treatmentPlanHandler := handler.NewTreatmentPlanHandler(treatmentPlanService)
	materialTestHandler := handler.NewMaterialTestHandler(materialTestService)
	stageApprovalHandler := handler.NewStageApprovalHandler(stageApprovalService)
	systemHandler := handler.NewSystemHandler(securityService, artifactService, treatmentPlanService, materialTestService, stageApprovalService, db, redisClient)

	engine.GET("/healthz", systemHandler.Health)
	engine.POST("/api/auth/login", systemHandler.Login)

	limiter := middleware.NewLimiter(redisClient, cfg.RequestLimit)
	api := engine.Group("/api")
	api.Use(limiter.Middleware(), middleware.Authenticate(cfg))
	api.GET("/overview", systemHandler.Overview)
	api.GET("/audits", middleware.RequireMinimumRole("reviewer"), systemHandler.Audits)
	api.GET("/session", systemHandler.Session)
	api.GET("/runtime", middleware.RequireMinimumRole("reviewer"), systemHandler.Runtime)
	api.GET("/audit-summary", middleware.RequireMinimumRole("reviewer"), systemHandler.AuditSummary)
	api.GET("/audits/:entityType/:id", middleware.RequireMinimumRole("reviewer"), systemHandler.EntityHistory)
	artifactHandler.Register(api)
	treatmentPlanHandler.Register(api)
	materialTestHandler.Register(api)
	stageApprovalHandler.Register(api)

	engine.NoRoute(func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "route_not_found"})
	})
	return engine
}
