package database

import (
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB là biến toàn cục (Global Variable) lưu trữ kết nối MySQL.
// Các module khác (user, wallet) sẽ gọi database.DB để thao tác với dữ liệu.
var DB *gorm.DB

// ConnectMySQL khởi tạo kết nối tới MySQL và thiết lập Pool
func ConnectMySQL(dsn string) {
	var err error

	// Không in câu lệnh SQL ra terminal. Các lỗi kết nối/nghiệp vụ quan trọng
	// vẫn được ghi bởi logger của ứng dụng.
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		// Dùng log.Fatalf để dập tắt server ngay lập tức nếu không kết nối được DB
		log.Fatalf("❌ Lỗi nghiêm trọng: Không thể kết nối MySQL: %v\n", err)
	}

	// 2. Lấy đối tượng sql.DB gốc ra để cấu hình Connection Pool
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("❌ Lỗi khi trích xuất sql.DB: %v\n", err)
	}

	// --- CẤU HÌNH CONNECTION POOL (BẮT BUỘC CHO HỆ THỐNG LỚN) ---

	// SetMaxIdleConns: Số lượng kết nối nhàn rỗi tối đa được giữ lại trong Pool.
	// Giảm thiểu thời gian đóng/mở kết nối liên tục khi có request mới.
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns: Số lượng kết nối tối đa được mở cùng lúc.
	// Nếu có 1000 user truy cập, app chỉ mở tối đa 100 kết nối, 900 user kia sẽ phải đợi.
	// Điều này giúp bảo vệ MySQL không bị sập (Out of Memory).
	sqlDB.SetMaxOpenConns(100)

	// SetConnMaxLifetime: Thời gian sống tối đa của một kết nối.
	// Tránh lỗi "MySQL server has gone away" khi kết nối bị hệ điều hành đóng âm thầm.
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("✅ Đã kết nối MySQL (GORM) thành công và khởi tạo Connection Pool!")
}
