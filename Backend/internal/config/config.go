package config

import (
	"fmt"
	"log"
	"os"

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

	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string

	AccessTokenSecret  string
	RefreshTokenSecret string
	CSRFSecret         string
	AppURL             string
	FirebaseCredentials string
}

// LoadConfig đọc file .env và nạp vào Config
func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ Không tìm thấy file .env, hệ thống sẽ dùng biến môi trường hệ thống")
	}

	cfg := &Config{
		ServerPort: getEnv("PORT", "8080"),
		ServerMode: getEnv("SERVER_MODE", "development"),

		MySQLUser:     getEnv("MYSQL_USER", "root"),
		MySQLPassword: getEnv("MYSQL_PASSWORD", ""),
		MySQLHost:     getEnv("MYSQL_HOST", "localhost"),
		MySQLPort:     getEnv("MYSQL_PORT", "3306"),
		MySQLDBName:   getEnv("MYSQL_DB", "nfbank_mysql"),

		MongoURI:    getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDBName: getEnv("MONGO_DB_NAME", "nfbank_mongo"),

		SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUsername: getRequiredEnv("SMTP_USERNAME"),
		SMTPPassword: getRequiredEnv("SMTP_PASSWORD"),
		SMTPFrom:     getRequiredEnv("SMTP_FROM"),

		AccessTokenSecret:   getRequiredEnv("ACCESS_TOKEN_SECRET"),
		RefreshTokenSecret:  getRequiredEnv("REFRESH_TOKEN_SECRET"),
		CSRFSecret:          getRequiredEnv("CSRF_SECRET"),
		AppURL:              getEnv("APP_URL", "http://localhost:8080"),
		FirebaseCredentials: getEnv("FIREBASE_CREDENTIALS", "./config/firebase-adminsdk.json"),
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
