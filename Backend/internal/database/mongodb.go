package database

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Khai báo 2 biến toàn cục:
// MongoClient: Dùng để quản lý kết nối (ví dụ: ngắt kết nối khi tắt server)
// Mongo: Dùng để thao tác trực tiếp với các collection (bảng) trong DB
var MongoClient *mongo.Client
var Mongo *mongo.Database

// ConnectMongoDB khởi tạo kết nối tới MongoDB với Timeout và Connection Pool
func ConnectMongoDB(uri string, dbName string) {
	// 1. Khởi tạo một Context với thời gian timeout là 10 giây.
	// Nếu sau 10 giây mà Mongo chưa phản hồi, tự động hủy kết nối để server không bị treo.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel() // Hàm này sẽ chạy khi ConnectMongoDB kết thúc để giải phóng bộ nhớ của ctx

	// 2. Cấu hình Connection Pool cho MongoDB
	clientOptions := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(100). // Số lượng kết nối tối đa được phép mở cùng lúc
		SetMinPoolSize(10)   // Luôn giữ ít nhất 10 kết nối chạy ngầm để phục vụ nhanh khi có request

	// 3. Thực hiện kết nối
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("❌ Lỗi cấu hình MongoDB: %v\n", err)
	}

	// 4. BẮT BUỘC: Gọi hàm Ping để xác nhận kết nối mạng thực sự thành công
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatalf("❌ Không thể kết nối tới server MongoDB (Ping failed): %v\n", err)
	}

	// 5. Gán vào biến toàn cục để các module khác sử dụng
	MongoClient = client
	Mongo = client.Database(dbName)

	log.Println("✅ Đã kết nối MongoDB thành công và Ping xác nhận!")
}