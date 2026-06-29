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

	MongoURI    string
	MongoDBName string

	AccessTokenSecret   string
	RefreshTokenSecret  string
	CSRFSecret          string
	AppURL              string
	FirebaseCredentials string

	EnableDevSeed       bool
	EnableOTPDebug      bool
	AllowTestPaymentOTP bool
	TransferMinAmount   int64
	TransferMaxAmount   int64
	DailyTransferLimit  int64
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
		MySQLDBName:   getEnv("MYSQL_DB", "nfbank_mysql"),

		MongoURI:    getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDBName: getEnv("MONGO_DB_NAME", "nfbank_mongo"),

		AccessTokenSecret:   getRequiredEnv("ACCESS_TOKEN_SECRET"),
		RefreshTokenSecret:  getRequiredEnv("REFRESH_TOKEN_SECRET"),
		CSRFSecret:          getRequiredEnv("CSRF_SECRET"),
		AppURL:              getEnv("APP_URL", "http://localhost:8080"),
		FirebaseCredentials: getEnv("FIREBASE_CREDENTIALS", "./config/firebase-adminsdk.json"),
		EnableDevSeed:       getEnvBool("ENABLE_DEV_SEED", serverMode != "production"),
		EnableOTPDebug:      getEnvBool("ENABLE_OTP_DEBUG", false),
		AllowTestPaymentOTP: getEnvBool("ALLOW_TEST_PAYMENT_OTP", false),
		TransferMinAmount:   getEnvInt64("TRANSFER_MIN_AMOUNT", 10000),
		TransferMaxAmount:   getEnvInt64("TRANSFER_MAX_AMOUNT", 500000000),
		DailyTransferLimit:  getEnvInt64("DAILY_TRANSFER_LIMIT", 1000000000),
	}

	if cfg.ServerMode == "production" {
		if cfg.EnableOTPDebug || cfg.AllowTestPaymentOTP {
			log.Fatal("❌ Không được bật OTP debug hoặc OTP kiểm thử trong production")
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
