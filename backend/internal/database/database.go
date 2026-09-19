package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/blueship581/museum-conservation-approval-control/backend/internal/config"
	"github.com/blueship581/museum-conservation-approval-control/backend/internal/model"
	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(ctx context.Context, cfg config.Config, log *slog.Logger) (*gorm.DB, *redis.Client, error) {
	var dialector gorm.Dialector
	switch cfg.DatabaseDriver {
	case "postgres":
		dialector = postgres.Open(cfg.DatabaseDSN)
	case "mysql":
		dialector = mysql.Open(cfg.DatabaseDSN)
	case "sqlite":
		dialector = sqlite.Open(cfg.DatabaseDSN)
	default:
		return nil, nil, fmt.Errorf("unsupported database driver %q", cfg.DatabaseDriver)
	}
	logLevel := logger.Warn
	if cfg.Environment == "development" {
		logLevel = logger.Info
	}
	var db *gorm.DB
	var err error
	for attempt := 1; attempt <= 20; attempt++ {
		db, err = gorm.Open(dialector, &gorm.Config{Logger: logger.Default.LogMode(logLevel)})
		if err == nil {
			sqlDB, dbErr := db.DB()
			if dbErr == nil && sqlDB.PingContext(ctx) == nil {
				break
			}
			if dbErr != nil {
				err = dbErr
			} else {
				err = sqlDB.PingContext(ctx)
			}
		}
		log.Warn("database not ready", "attempt", attempt, "error", err)
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	if err != nil {
		return nil, nil, fmt.Errorf("connect database: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, nil, err
	}
	if err := Seed(ctx, db); err != nil {
		return nil, nil, err
	}
	var redisClient *redis.Client
	if cfg.RedisAddr != "" {
		redisClient = redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword})
		if err := redisClient.Ping(ctx).Err(); err != nil {
			return nil, nil, fmt.Errorf("connect redis: %w", err)
		}
	}
	return db, redisClient, nil
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{}, &model.AuditLog{},
		&model.Artifact{},
		&model.TreatmentPlan{},
		&model.MaterialTest{},
		&model.StageApproval{},
		&model.ApprovalOpinion{},
	)
}

func Seed(ctx context.Context, db *gorm.DB) error {
	var users int64
	if err := db.WithContext(ctx).Model(&model.User{}).Count(&users).Error; err != nil {
		return err
	}
	if users == 0 {
		password, err := bcrypt.GenerateFromPassword([]byte("Admin123!"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		seedUsers := []model.User{
			{Username: "admin", DisplayName: "系统管理员", PasswordHash: string(password), Role: model.RoleAdmin, Active: true},
			{Username: "reviewer", DisplayName: "质量复核员", PasswordHash: string(password), Role: model.RoleReviewer, Active: true},
			{Username: "operator", DisplayName: "现场操作员", PasswordHash: string(password), Role: model.RoleOperator, Active: true},
			{Username: "viewer", DisplayName: "只读观察员", PasswordHash: string(password), Role: model.RoleViewer, Active: true},
		}
		if err := db.WithContext(ctx).Create(&seedUsers).Error; err != nil {
			return err
		}
	}

	if err := seedArtifact(ctx, db); err != nil {
		return err
	}

	if err := seedTreatmentPlan(ctx, db); err != nil {
		return err
	}

	if err := seedMaterialTest(ctx, db); err != nil {
		return err
	}

	if err := seedStageApproval(ctx, db); err != nil {
		return err
	}

	return nil
}

func seedArtifact(ctx context.Context, db *gorm.DB) error {
	var count int64
	if err := db.WithContext(ctx).Model(&model.Artifact{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	now := time.Now().UTC()
	items := []model.Artifact{

		{BaseModel: model.BaseModel{Code: "A-001", Name: "文物示例一", Status: "registered", Version: 1,
			Description: "用于启动验证和主要流程演示的文物记录"}, Facility: "文物保护处理审批区域1", Owner: "运行一组",
			Category: "常规", RiskLevel: "low", MetricValue: 12.5, MetricUnit: "unit",
			EffectiveAt: now.Add(0 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-512-01"},

		{BaseModel: model.BaseModel{Code: "A-002", Name: "文物示例二", Status: "stable", Version: 1,
			Description: "用于启动验证和主要流程演示的文物记录"}, Facility: "文物保护处理审批区域2", Owner: "质量复核组",
			Category: "重点", RiskLevel: "medium", MetricValue: 25.0, MetricUnit: "%",
			EffectiveAt: now.Add(3 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-512-02"},

		{BaseModel: model.BaseModel{Code: "A-003", Name: "文物示例三", Status: "treatment", Version: 1,
			Description: "用于启动验证和主要流程演示的文物记录"}, Facility: "文物保护处理审批区域3", Owner: "安全主管组",
			Category: "复核", RiskLevel: "high", MetricValue: 37.5, MetricUnit: "score",
			EffectiveAt: now.Add(6 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-512-03"},
	}
	return db.WithContext(ctx).Create(&items).Error
}

func seedTreatmentPlan(ctx context.Context, db *gorm.DB) error {
	var count int64
	if err := db.WithContext(ctx).Model(&model.TreatmentPlan{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	now := time.Now().UTC()
	items := []model.TreatmentPlan{

		{BaseModel: model.BaseModel{Code: "TP-001", Name: "处理方案示例一", Status: "draft", Version: 1,
			Description: "用于启动验证和主要流程演示的处理方案记录"}, Facility: "文物保护处理审批区域1", Owner: "运行一组",
			Category: "常规", RiskLevel: "low", MetricValue: 12.5, MetricUnit: "unit",
			EffectiveAt: now.Add(0 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-512-01"},

		{BaseModel: model.BaseModel{Code: "TP-002", Name: "处理方案示例二", Status: "review", Version: 1,
			Description: "用于启动验证和主要流程演示的处理方案记录"}, Facility: "文物保护处理审批区域2", Owner: "质量复核组",
			Category: "重点", RiskLevel: "medium", MetricValue: 25.0, MetricUnit: "%",
			EffectiveAt: now.Add(3 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-512-02"},

		{BaseModel: model.BaseModel{Code: "TP-003", Name: "处理方案示例三", Status: "approved", Version: 1,
			Description: "用于启动验证和主要流程演示的处理方案记录"}, Facility: "文物保护处理审批区域3", Owner: "安全主管组",
			Category: "复核", RiskLevel: "high", MetricValue: 37.5, MetricUnit: "score",
			EffectiveAt: now.Add(6 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "REL-512-03"},
	}
	return db.WithContext(ctx).Create(&items).Error
}

func seedMaterialTest(ctx context.Context, db *gorm.DB) error {
	var count int64
	if err := db.WithContext(ctx).Model(&model.MaterialTest{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	now := time.Now().UTC()
	items := []model.MaterialTest{

		{BaseModel: model.BaseModel{Code: "MT-001", Name: "材料检测示例一", Status: "planned", Version: 1,
			Description: "用于启动验证和主要流程演示的材料检测记录"}, Facility: "文物保护处理审批区域1", Owner: "运行一组",
			Category: "常规", RiskLevel: "low", MetricValue: 12.5, MetricUnit: "unit",
			EffectiveAt: now.Add(0 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "TP-001"},

		{BaseModel: model.BaseModel{Code: "MT-002", Name: "丝绸纤维成分检测", Status: "running", Version: 1,
			Description: "用于演示待复核门禁缺少有效检测：检测尚未核验"}, Facility: "文物保护处理审批区域2", Owner: "质量复核组",
			Category: "重点", RiskLevel: "medium", MetricValue: 25.0, MetricUnit: "%",
			EffectiveAt: now.Add(3 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "TP-002"},

		{BaseModel: model.BaseModel{Code: "MT-003", Name: "漆皮附着力检测", Status: "verified", Version: 1,
			Description: "用于启动验证和门禁放行演示的已核验材料检测"}, Facility: "文物保护处理审批区域3", Owner: "安全主管组",
			Category: "复核", RiskLevel: "high", MetricValue: 37.5, MetricUnit: "score",
			EffectiveAt: now.Add(6 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "TP-003"},

		{BaseModel: model.BaseModel{Code: "MT-004", Name: "漆皮耐老化复测", Status: "verified", Version: 2,
			Description: "同方案下更新时间更新的已核验检测，冻结时应选中本记录而非 MT-003"},
			Facility: "文物保护处理审批区域3", Owner: "安全主管组",
			Category: "复核", RiskLevel: "high", MetricValue: 41.0, MetricUnit: "score",
			EffectiveAt: now.Add(9 * time.Hour), Evidence: "复测通过，结论维持稳定",
			RelatedCode: "TP-003"},
	}
	return db.WithContext(ctx).Create(&items).Error
}

func seedStageApproval(ctx context.Context, db *gorm.DB) error {
	var count int64
	if err := db.WithContext(ctx).Model(&model.StageApproval{}).Count(&count).Error; err != nil || count > 0 {
		return err
	}
	now := time.Now().UTC()
	items := []model.StageApproval{

		{BaseModel: model.BaseModel{Code: "SA-001", Name: "阶段审批示例一", Status: "draft", Version: 1,
			Description: "用于启动验证和主要流程演示的阶段审批记录"}, Facility: "文物保护处理审批区域1", Owner: "运行一组",
			Category: "常规", RiskLevel: "low", MetricValue: 12.5, MetricUnit: "unit",
			EffectiveAt: now.Add(0 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "TP-001"},

		// SA-002 处于待复核但关联方案尚无已核验检测，用于演示“门禁拦截、保持待复核”。
		{BaseModel: model.BaseModel{Code: "SA-002", Name: "阶段审批示例二", Status: "review", Version: 2,
			Description: "用于启动验证和主要流程演示的阶段审批记录"}, Facility: "文物保护处理审批区域2", Owner: "质量复核组",
			Category: "重点", RiskLevel: "medium", MetricValue: 25.0, MetricUnit: "%",
			EffectiveAt: now.Add(3 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "TP-002"},

		// SA-003 已批准并保留当时的检测快照 MT-004 v2，方案页与审批列表展示冻结结论。
		{BaseModel: model.BaseModel{Code: "SA-003", Name: "阶段审批示例三", Status: "approved", Version: 3,
			Description: "用于启动验证和主要流程演示的阶段审批记录"}, Facility: "文物保护处理审批区域3", Owner: "安全主管组",
			Category: "复核", RiskLevel: "high", MetricValue: 37.5, MetricUnit: "score",
			EffectiveAt: now.Add(6 * time.Hour), Evidence: "已完成基础证据核对", RelatedCode: "TP-003"},
	}
	// 已批准示例在门禁上线前完成，补齐其冻结快照字段以保持演示数据一致：
	// 关联方案 TP-003、冻结当时最新的已核验检测 MT-004 v2。
	items[2].GatePlanCode = "TP-003"
	items[2].GateTestCode = "MT-004"
	items[2].GateTestName = "漆皮耐老化复测"
	items[2].GateTestVersion = 2
	items[2].GateTestStatus = "verified"
	items[2].GateTestUpdated = now.Add(9 * time.Hour)
	items[2].GateFrozenAt = now.Add(10 * time.Hour)
	items[2].GateVerdict = "frozen"
	return db.WithContext(ctx).Create(&items).Error
}
