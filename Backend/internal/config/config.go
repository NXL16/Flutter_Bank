package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config chứa toàn bộ cấu hình hệ thống
type Config struct {
	ServerPort string
	ServerMode string

	MySQLUser     string
	MySQLPassword string
	MySQLHost     string
	MySQLPort     string
	MySQLDBName   string

	AccessTokenSecret   string
	RefreshTokenSecret  string
	AppURL              string
	FirebaseCredentials string
	CloudinaryCloudName string
	CloudinaryAPIKey    string
	CloudinaryAPISecret string

	EnableDevSeed         bool
	AllowTestPaymentOTP   bool
	TransferMinAmount     int64
	TransferMaxAmount     int64
	DailyTransferLimit    int64
	AdminDepositMaxAmount int64

	SavingsMaturityIntervalSeconds int64
}

// LoadConfig đọc file .env và nạp vào Config
func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ Không tìm thấy file .env, hệ thống sẽ dùng biến môi trường hệ thống")
	}

	serverMode := strings.ToLower(getEnv("SERVER_MODE", "development"))

	cfg := &Config{
		ServerPort: getEnv("PORT", "8080"),
		ServerMode: serverMode,

		MySQLUser:     getEnv("MYSQL_USER", "root"),
		MySQLPassword: getEnv("MYSQL_PASSWORD", ""),
		MySQLHost:     getEnv("MYSQL_HOST", "localhost"),
		MySQLPort:     getEnv("MYSQL_PORT", "3306"),
		MySQLDBName:   getEnv("MYSQL_DB", "nfbank"),

		AccessTokenSecret:   getRequiredEnv("ACCESS_TOKEN_SECRET"),
		RefreshTokenSecret:  getRequiredEnv("REFRESH_TOKEN_SECRET"),
		AppURL:              getEnv("APP_URL", "http://localhost:8080"),
		FirebaseCredentials: getEnv("FIREBASE_CREDENTIALS", "./internal/config/firebase-adminsdk.json"),
		CloudinaryCloudName: getEnv("CLOUDINARY_CLOUD_NAME", ""),
		CloudinaryAPIKey:    getEnv("CLOUDINARY_API_KEY", ""),
		CloudinaryAPISecret: getEnv("CLOUDINARY_API_SECRET", ""),
		EnableDevSeed:       getEnvBool("ENABLE_DEV_SEED", serverMode != "production"),
		AllowTestPaymentOTP: getEnvBool("ALLOW_TEST_PAYMENT_OTP", false),
		TransferMinAmount:   getEnvInt64("TRANSFER_MIN_AMOUNT", 10000),
		TransferMaxAmount:   getEnvInt64("TRANSFER_MAX_AMOUNT", 500000000),
		DailyTransferLimit:  getEnvInt64("DAILY_TRANSFER_LIMIT", 1000000000),
		AdminDepositMaxAmount: getEnvInt64(
			"ADMIN_DEPOSIT_MAX_AMOUNT",
			5000000000,
		),

		SavingsMaturityIntervalSeconds: getEnvInt64(
			"SAVINGS_MATURITY_INTERVAL_SECONDS",
			60,
		),
	}

	if cfg.ServerMode == "production" {
		if cfg.AllowTestPaymentOTP {
			log.Fatal("❌ Không được bật OTP kiểm thử trong production")
		}
		if !strings.HasPrefix(cfg.AppURL, "https://") {
			log.Fatal("❌ APP_URL phải sử dụng HTTPS trong production")
		}
	}
	if cfg.TransferMinAmount <= 0 ||
		cfg.TransferMaxAmount < cfg.TransferMinAmount ||
		cfg.DailyTransferLimit < cfg.TransferMaxAmount {
		log.Fatal("❌ Cấu hình hạn mức chuyển tiền không hợp lệ")
	}
	if cfg.AdminDepositMaxAmount < cfg.TransferMinAmount {
		log.Fatal("❌ ADMIN_DEPOSIT_MAX_AMOUNT không hợp lệ")
	}
	if cfg.SavingsMaturityIntervalSeconds <= 0 {
		log.Fatal("❌ SAVINGS_MATURITY_INTERVAL_SECONDS phải lớn hơn 0")
	}
	cloudinaryValues := []string{
		cfg.CloudinaryCloudName,
		cfg.CloudinaryAPIKey,
		cfg.CloudinaryAPISecret,
	}
	configuredCloudinaryValues := 0
	for _, value := range cloudinaryValues {
		if strings.TrimSpace(value) != "" {
			configuredCloudinaryValues++
		}
	}
	if configuredCloudinaryValues != 0 &&
		configuredCloudinaryValues != len(cloudinaryValues) {
		log.Fatal("❌ Phải cấu hình đầy đủ CLOUDINARY_CLOUD_NAME, CLOUDINARY_API_KEY và CLOUDINARY_API_SECRET")
	}

	return cfg
}

// GetMySQLDSN tự động ghép chuỗi DSN chuẩn cho GORM
func (c *Config) GetMySQLDSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.MySQLUser,
		c.MySQLPassword,
		c.MySQLHost,
		c.MySQLPort,
		c.MySQLDBName,
	)
}

func getEnv(key string, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		return defaultValue
	}
	return value
}

func getRequiredEnv(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		log.Fatalf("❌ Thiếu biến môi trường bắt buộc: %s", key)
	}

	return value
}

func getEnvBool(key string, defaultValue bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists || strings.TrimSpace(value) == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		log.Fatalf("❌ Biến môi trường %s phải là true hoặc false", key)
	}
	return parsed
}

func getEnvInt64(key string, defaultValue int64) int64 {
	value, exists := os.LookupEnv(key)
	if !exists || strings.TrimSpace(value) == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		log.Fatalf("❌ Biến môi trường %s phải là số nguyên", key)
	}
	return parsed
}
