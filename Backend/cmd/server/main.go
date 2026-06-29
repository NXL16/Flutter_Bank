package main

import (
	"fmt"
	"log"
	"net/http"

	"bank-service/internal/config"
	"bank-service/internal/database"
	"bank-service/internal/infrastructure/firebase"
	"bank-service/internal/modules/account"
	"bank-service/internal/modules/admin"
	"bank-service/internal/modules/auth"
	"bank-service/internal/modules/credit"
	"bank-service/internal/modules/notification"
	"bank-service/internal/modules/payment"
	"bank-service/internal/modules/savings"
	"bank-service/internal/modules/transaction"
	"bank-service/internal/modules/user"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("🚀 Đang khởi động hệ thống NF-Bank...")

	cfg := config.LoadConfig()

	database.ConnectMongoDB(cfg.MongoURI, cfg.MongoDBName)

	dsn := cfg.GetMySQLDSN()
	database.ConnectMySQL(dsn)

	if err := database.DB.AutoMigrate(
		&auth.User{},
		&auth.RefreshToken{},
		&auth.UserDevice{},
		&auth.PendingLogin{},
		&notification.Notification{},
		&notification.PushToken{},
		&account.Account{},
		&savings.SavingsDetail{},
		&credit.CreditDetail{},
		&user.UserProfile{},
		&transaction.Transaction{},
		&transaction.LedgerEntry{},
		&payment.Merchant{},
		&payment.PaymentSession{},
	); err != nil {
		log.Fatalf("❌ MySQL Auto Migration thất bại: %v", err)
	}
	log.Println("✅ MySQL Auto Migration hoàn tất!")

	// Khởi tạo Firebase Admin Client
	firebaseClient, err := firebase.InitFirebase(cfg.FirebaseCredentials)
	if err != nil {
		log.Fatalf("❌ Khởi tạo Firebase Admin SDK thất bại: %v", err)
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
		log.Println("⚠️ Development seed đang được bật")

		var count int64
		if err := database.DB.Model(&auth.User{}).Where("role = ?", "super_admin").Count(&count).Error; err != nil {
			log.Printf("⚠️ Lỗi kiểm tra tài khoản Super Admin: %v", err)
		} else if count == 0 {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte("SuperAdmin123!"), bcrypt.DefaultCost)
			if err != nil {
				log.Fatalf("❌ Lỗi mã hóa mật khẩu Super Admin: %v", err)
			}
			superAdmin := auth.User{
				FullName:     "Super Admin",
				Email:        "84999999999@phone.identity",
				PasswordHash: string(hashedPassword),
				Phone:        "+84999999999",
				Role:         "super_admin",
				IsVerified:   true,
				IsLocked:     false,
				TOTPSecret:   "KGF2MOLIONATKJ5IWJW4FJYUVFS7KHPT",
			}
			if err := database.DB.Create(&superAdmin).Error; err != nil {
				log.Printf("⚠️ Lỗi tạo tài khoản Super Admin phát triển: %v", err)
			} else {
				_ = accountService.CreateDefaultPaymentAccount(superAdmin.ID)
			}
		}

		seedTestUsers(database.DB, accountService)
		seedMerchant(database.DB, accountService)
	}

	userRepo := user.NewRepository(database.DB)
	userService := user.NewService(userRepo)

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
	transactionService := transaction.NewService(transactionRepo, firebaseClient, notificationService, cfg)
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
	savingsService := savings.NewService(savingsRepo, notificationService)
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
	fmt.Printf("✅ Server đang chạy tại cổng %s\n", cfg.ServerPort)

	if err := r.Run(port); err != nil {
		log.Fatalf("❌ Lỗi nghiêm trọng khi khởi chạy server: %v", err)
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

func seedTestUsers(db *gorm.DB, accountService *account.Service) {
	usersToSeed := []struct {
		FullName string
		Phone    string
		Password string
	}{
		{"Nguyen Van A", "+84999000001", "Testuser123!"},
		{"Tran Thi B", "+84999000002", "Testuser123!"},
	}

	for _, u := range usersToSeed {
		var count int64
		if err := db.Model(&auth.User{}).Where("phone = ?", u.Phone).Count(&count).Error; err == nil && count == 0 {
			hashed, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
			if err != nil {
				log.Printf("⚠️ Lỗi mã hóa mật khẩu test user %s: %v", u.Phone, err)
				continue
			}
			user := auth.User{
				FullName:     u.FullName,
				Email:        u.Phone[1:] + "@phone.identity",
				PasswordHash: string(hashed),
				Phone:        u.Phone,
				Role:         "user",
				IsVerified:   true,
				IsLocked:     false,
			}
			if err := db.Create(&user).Error; err != nil {
				log.Printf("⚠️ Lỗi tạo test user %s: %v", u.Phone, err)
			} else {
				log.Printf("✅ Đã tạo test user %s (Testuser123!)", u.Phone)
				if err := accountService.CreateDefaultPaymentAccount(user.ID); err != nil {
					log.Printf("⚠️ Lỗi cấp tài khoản ví cho test user %s: %v", u.Phone, err)
				}
			}
		}
	}
}

func seedMerchant(db *gorm.DB, accountService *account.Service) {
	// 1. Seed Merchant User
	merchantPhone := "+84888000001"
	var userCount int64
	var merchantUser auth.User

	err := db.Model(&auth.User{}).Where("phone = ?", merchantPhone).Count(&userCount).Error
	if err == nil && userCount == 0 {
		hashed, err := bcrypt.GenerateFromPassword([]byte("Merchant123!"), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("❌ Lỗi mã hóa mật khẩu merchant user: %v", err)
		}
		merchantUser = auth.User{
			FullName:     "Music App Merchant",
			Email:        "84888000001@phone.identity",
			PasswordHash: string(hashed),
			Phone:        merchantPhone,
			Role:         "user",
			IsVerified:   true,
			IsLocked:     false,
		}
		if err := db.Create(&merchantUser).Error; err != nil {
			log.Printf("⚠️ Lỗi tạo tài khoản merchant user: %v", err)
			return
		}
		log.Printf("✅ Đã tạo tài khoản Merchant User mặc định (%s)", merchantPhone)
		if err := accountService.CreateDefaultPaymentAccount(merchantUser.ID); err != nil {
			log.Printf("⚠️ Lỗi tạo tài khoản ví cho Merchant User: %v", err)
			return
		}
	} else {
		db.Where("phone = ?", merchantPhone).First(&merchantUser)
	}

	// 2. Lấy ID tài khoản ví thanh toán PAYMENT của Merchant User
	var merchantAccount account.Account
	err = db.Where("user_id = ? AND account_type = ?", merchantUser.ID, "PAYMENT").First(&merchantAccount).Error
	if err != nil {
		log.Printf("⚠️ Không tìm thấy ví thanh toán cho Merchant User: %v", err)
		return
	}

	// 3. Seed Merchant configuration
	partnerCode := "NFBANK_PROD_OR_TEST_ID"
	var merchantCount int64
	err = db.Table("merchants").Where("partner_code = ?", partnerCode).Count(&merchantCount).Error
	if err == nil && merchantCount == 0 {
		m := payment.Merchant{
			PartnerCode:      partnerCode,
			AccessKey:        "your_nfbank_access_key_here",
			SecretKey:        "your_nfbank_secret_key_here",
			MerchantName:     "App Âm Nhạc (Music App)",
			PaymentAccountID: merchantAccount.ID,
		}
		if err := db.Create(&m).Error; err != nil {
			log.Printf("⚠️ Lỗi tạo cấu hình đối tác Merchant: %v", err)
		} else {
			log.Println("✅ Đã tạo cấu hình đối tác Merchant (App Âm Nhạc) thành công!")
		}
	}
}
