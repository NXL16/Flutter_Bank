🏦 NFBank - Fintech Web ApplicationNFBank là một ứng dụng ngân hàng số hiện đại, được thiết kế theo tư tưởng Modular Monolith kết hợp Domain-Driven Design (DDD) cho Backend và Feature-Sliced Design (FSD) cho Frontend.

Dự án được xây dựng với mục tiêu tối ưu hóa trải nghiệm cho Developer (hiện tại là Solo Dev) nhưng vẫn tuân thủ nghiêm ngặt các tiêu chuẩn bảo mật và hoàn toàn sẵn sàng (Cloud-native ready) để tách thành các Microservices độc lập khi hệ thống cần mở rộng.

🚀 1. Tổng quan Công nghệ (Tech Stack)

Tổng thể: Kiến trúc Monorepo (Quản lý BE và FE chung một kho lưu trữ).

Backend: Golang (Gin/Fiber) xử lý hiệu năng cao.

Frontend: Next.js (App Router), TypeScript, TailwindCSS.

Database Strategy (Polyglot Persistence):

   * MySQL: Lưu trữ dữ liệu quan hệ, bền vững (User, Account, Transaction, Balance).

   * MongoDB: Lưu trữ dữ liệu phi cấu trúc, có tính thời vụ (OTP với TTL Index, System Logs).
   
Bảo mật (Security): * Mật khẩu băm bằng thuật toán Bcrypt.
   * Cơ chế cấp phép kép: Access Token (JWT - TTL 5 phút) và Refresh Token (JWT - TTL 7 ngày lưu qua HttpOnly + Secure Cookie để chống XSS và CSRF).

⚙️ 2. Luồng hoạt động cốt lõi (Core Flows)

   2.1 Luồng Đăng ký / Xác thực (Auth Flow):
   
   * Người dùng đăng ký/đăng nhập. Backend kiểm tra MySQL bằng Bcrypt.
      
   * Nếu yêu cầu xác minh, hệ thống sinh OTP và lưu vào MongoDB. Sau 5 phút, OTP tự động bốc hơi nhờ TTL Index, giúp DB không bị rác.
   
   * Cấp phát Token: Frontend nhận Access Token (lưu memory/local state) và tự động đính kèm Refresh Token vào Cookie (HttpOnly). Khi Access Token hết hạn, Axios Interceptor ở FE tự động gọi API /refresh bằng Cookie để lấy token mới một cách "vô hình" với người dùng.
      
   2.2 Luồng Giao dịch (Transaction Flow):
   * Frontend gọi API qua các Component đã được cô lập.
      
   * Backend tiếp nhận qua Router $\rightarrow$ Middleware (kiểm tra Auth/Rate Limit) $\rightarrow$ Handler $\rightarrow$ Usecase (xử lý logic trừ tiền/cộng tiền trong 1 Transaction của MySQL) $\rightarrow$ Trả về kết quả.
      
📂 3. Cấu trúc Dự án (Project Structure)Toàn bộ dự án được đặt trong một Monorepo để dễ dàng khởi chạy môi trường dev chỉ với 1 lệnh docker-compose up.

   🔹 3.1. Cấu trúc Backend (Golang)Backend tuân thủ Clean Architecture và gom nhóm theo Domain-Driven Design (DDD).


├── cmd/server/main.go            # Entry point: Khởi tạo DB, tiêm Dependency Injection
├── internal/
│   ├── config/                   # Đọc biến môi trường (.env)
│   ├── database/                 # Khởi tạo kết nối MySQL, MongoDB
│   ├── middleware/               # Middleware toàn cục (Logger, CORS, Rate Limit)
│   ├── modules/                  # ✨ LÕI ỨNG DỤNG: Tách biệt theo từng Nghiệp vụ
│   │   ├── auth/                 # (Ví dụ Domain Auth)
│   │   │   ├── handler.go        # Giao tiếp HTTP (Nhận request, parse JSON)
│   │   │   ├── usecase.go        # Business logic (Sinh OTP, Hash Pass, JWT)
│   │   │   ├── repository.go     # Giao tiếp trực tiếp với MySQL/Mongo
│   │   │   └── routes.go         # Router cục bộ của riêng Auth
│   │   ├── user/                 # Domain User
│   │   └── wallet/               # Domain Wallet
│   └── infrastructure/           # Giao tiếp với bên thứ 3 (Email, SMS provider)
└── pkg/                          # Tiện ích dùng chung (Validator, Error wrapper)


💡 Tại sao chọn cấu trúc này?
   * Tránh "Spaghetti Code": Không chia thư mục theo kiểu kỹ thuật (handlers, services dàn hàng ngang) mà chia theo Nghiệp vụ (auth, wallet). Muốn sửa lỗi chuyển tiền, chỉ cần vào folder wallet.

   * Sẵn sàng cho Microservices: Nếu module wallet bị quá tải do nhiều người dùng, chỉ cần copy thư mục internal/modules/wallet ra một repo mới là có ngay một Microservice độc lập.
   
   * Dễ Test (Testability): Tầng usecase không phụ thuộc vào framework HTTP hay DB. Có thể viết Unit Test cực nhanh bằng cách Mock các interfaces.

   🔹 3.2. Cấu trúc Frontend (Next.js)Frontend tuân thủ kiến trúc Feature-Sliced Design (FSD) để chống lại việc lồng ghép Component vô tội vạ.Plaintextfrontend/src/


├── app/                          # 1. Routing & Layouts (App Router)
│   ├── (auth)/login/page.tsx
│   └── dashboard/page.tsx
│
├── features/                     # 2. Ngăn chứa Nghiệp vụ (Logic Components)
│   ├── auth/                     # Tính năng Auth
│   │   ├── components/           # LoginForm, OTPInput
│   │   ├── services/             # API calls (login, refresh)
│   │   └── store/                # Zustand/Redux slice cho User
│   └── wallet/                   # Tính năng Wallet
│
├── shared/                       # 3. UI/UX & Tiện ích dùng chung
│   ├── ui/                       # Button, Input, Modal (Không chứa logic nghiệp vụ)
│   └── utils/                    # formatCurrency, formatDate
│
└── core/                         # 4. Trái tim cấu hình FE
    ├── api/                      # Axios instance & Interceptors (Auto refresh token)
    └── theme/                    # Cấu hình màu sắc, typography
💡 Tại sao chọn cấu trúc này?

   * Nguyên tắc dòng chảy một chiều: Tầng app có thể gọi features, tầng features có thể gọi shared. Tuyệt đối không có chiều ngược lại. Nút Button ở shared/ui không bao giờ được phép biết logic "Đăng nhập" là gì.
   
   * Tính đóng gói cao (Encapsulation): Các Component của tính năng Auth chỉ nằm trong thư mục Auth, không làm "ô nhiễm" mã nguồn của tính năng Wallet.
   
   * Phù hợp với Solo Dev/Small Team: Tránh được rào cản vận hành (Overhead) khổng lồ của Micro-frontend, trong khi vẫn giữ được sự rõ ràng và tách bạch của các tính năng.

🛠 4. Hướng dẫn khởi chạy (Getting Started)