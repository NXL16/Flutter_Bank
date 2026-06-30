package main

import (
	"fmt"
	"log"
	"net/http"

	"bank-service/internal/config"
	"bank-service/internal/database"
	"bank-service/internal/infrastructure/cloudinary"
	"bank-service/internal/infrastructure/firebase"
	"bank-service/internal/modules/account"
	"bank-service/internal/modules/admin"
	"bank-service/internal/modules/auth"
	"bank-service/internal/modules/notification"
	"bank-service/internal/modules/payment"
	"bank-service/internal/modules/savings"
	"bank-service/internal/modules/transaction"
	"bank-service/internal/modules/user"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg := config.LoadConfig()

	dsn := cfg.GetMySQLDSN()
	database.ConnectMySQL(dsn)

	// Phiên bản cũ giới hạn một tài khoản SAVINGS cho mỗi người. Bỏ unique
	// index này trước khi migrate để hỗ trợ nhiều sổ tiết kiệm độc lập.
	if database.DB.Migrator().HasTable(&account.Account{}) &&
		database.DB.Migrator().HasIndex(&account.Account{}, "idx_user_account_type") {
		if !database.DB.Migrator().HasIndex(&account.Account{}, "idx_accounts_user_type") {
			if err := database.DB.Migrator().CreateIndex(&account.Account{}, "idx_accounts_user_type"); err != nil {
				log.Fatalf("Không thể tạo index tài khoản thay thế: %v", err)
			}
		}
		if err := database.DB.Migrator().DropIndex(&account.Account{}, "idx_user_account_type"); err != nil {
			log.Fatalf("Không thể bỏ giới hạn một sổ tiết kiệm: %v", err)
		}
	}

	if err := database.DB.AutoMigrate(
		&auth.User{},
		&auth.RefreshToken{},
		&auth.UserDevice{},
		&notification.Notification{},
		&notification.PushToken{},
		&account.Account{},
		&savings.SavingsDetail{},
		&user.UserProfile{},
		&transaction.Transaction{},
		&transaction.LedgerEntry{},
		&transaction.TransactionPIN{},
		&payment.Merchant{},
		&payment.PaymentSession{},
	); err != nil {
		log.Fatalf("MySQL Auto Migration thất bại: %v", err)
	}
	log.Println("MySQL Auto Migration hoàn tất!")

	// Khởi tạo Firebase Admin Client
	firebaseClient, err := firebase.InitFirebase(cfg.FirebaseCredentials)
	if err != nil {
		log.Fatalf("Khởi tạo Firebase Admin SDK thất bại: %v", err)
	}

	if cfg.ServerMode == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.Use(corsMiddleware(cfg))

	authRepo := auth.NewRepository(database.DB)
	accountRepo := account.NewRepository(database.DB)
	accountService := account.NewService(accountRepo)

	if cfg.EnableDevSeed {
		log.Println("Development seed đang được bật")

		var count int64
		if err := database.DB.Model(&auth.User{}).Where("role = ?", "super_admin").Count(&count).Error; err != nil {
			log.Printf("Lỗi kiểm tra tài khoản Super Admin: %v", err)
		} else if count == 0 {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte("SuperAdmin123!"), bcrypt.DefaultCost)
			if err != nil {
				log.Fatalf("Lỗi mã hóa mật khẩu Super Admin: %v", err)
			}
			superAdmin := auth.User{
				FullName:     "Super Admin",
				PasswordHash: string(hashedPassword),
				Phone:        "+84999999999",
				Role:         "super_admin",
				IsVerified:   true,
				IsLocked:     false,
				TOTPSecret:   "KGF2MOLIONATKJ5IWJW4FJYUVFS7KHPT",
			}
			if err := database.DB.Create(&superAdmin).Error; err != nil {
				log.Printf("Lỗi tạo tài khoản Super Admin phát triển: %v", err)
			} else {
				_ = accountService.CreateDefaultPaymentAccount(superAdmin.ID)
			}
		}
	}

	userRepo := user.NewRepository(database.DB)
	cloudinaryClient := cloudinary.NewClient(
		cfg.CloudinaryCloudName,
		cfg.CloudinaryAPIKey,
		cfg.CloudinaryAPISecret,
	)
	userService := user.NewService(userRepo, cloudinaryClient)

	authService := auth.NewService(
		authRepo,
		accountService,
		userService,
		cfg,
		firebaseClient,
	)
	authHandler := auth.NewHandler(authService)

	accountHandler := account.NewHandler(accountService)
	userHandler := user.NewHandler(userService)

	// Khởi tạo notification module
	notificationRepo := notification.NewRepository(database.DB)
	notificationService := notification.NewService(notificationRepo, firebaseClient)
	notificationHandler := notification.NewHandler(notificationService)

	transactionRepo := transaction.NewRepository(database.DB)
	transactionService := transaction.NewService(transactionRepo, notificationService, cfg)
	transactionHandler := transaction.NewHandler(transactionService)

	adminRepo := admin.NewRepository(database.DB)
	adminService := admin.NewService(adminRepo, accountService, transactionService)
	adminHandler := admin.NewHandler(adminService)

	// Khởi tạo payment module
	paymentRepo := payment.NewRepository(database.DB)
	paymentService := payment.NewService(paymentRepo, firebaseClient, cfg, notificationService)
	paymentHandler := payment.NewHandler(paymentService)

	// Khởi tạo savings module
	savingsRepo := savings.NewRepository(database.DB)
	savingsService := savings.NewService(savingsRepo, notificationService, transactionService)
	savingsHandler := savings.NewHandler(savingsService)

	api := r.Group("/api/v1")
	auth.RegisterRoutes(api, authHandler)
	account.RegisterRoutes(api, accountHandler, cfg)
	user.RegisterRoutes(api, userHandler, cfg)
	transaction.RegisterRoutes(api, transactionHandler, cfg)
	admin.RegisterRoutes(api, adminHandler, cfg)
	payment.RegisterRoutes(api, paymentHandler, cfg)
	notification.RegisterRoutes(api, notificationHandler, cfg)
	savings.RegisterRoutes(api, savingsHandler, cfg)

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "NF-Bank Server is running!",
			"mode":    cfg.ServerMode,
		})
	})

	port := fmt.Sprintf(":%s", cfg.ServerPort)
	fmt.Printf("Server đang chạy tại cổng %s\n", cfg.ServerPort)

	if err := r.Run(port); err != nil {
		log.Fatalf("Lỗi nghiêm trọng khi khởi chạy server: %v", err)
	}
}

func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowedOrigin := cfg.AppURL

		if cfg.ServerMode != "production" && origin != "" {
			allowedOrigin = origin
		}

		if origin != "" && origin == allowedOrigin {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
