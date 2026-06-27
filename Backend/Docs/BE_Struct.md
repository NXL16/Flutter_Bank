BANK-SERVICE/
├── cmd/
│   └── server/
│       └── main.go                 # Vẫn giữ nguyên, khởi tạo mọi thứ ở đây
│
├── internal/
│   ├── config/                     # Load các biến từ .env
│   ├── database/                   # Khởi tạo kết nối DB (Chuyển từ utils/database qua đây)
│   │   ├── mongodb.go
│   │   └── mysql.go
│   │
│   ├── core/                       # LÕI CỦA ỨNG DỤNG (Nơi chứa Interfaces)
│   │   └── ports/                  # Định nghĩa các Interface cho Repository, Service, Email, SMS...
│   │
│   ├── modules/                    # GOM CODE THEO TỪNG NGHIỆP VỤ (DOMAIN)
│   │   ├── auth/                   # Toàn bộ những gì liên quan đến Auth nằm hết ở đây
│   │   │   ├── handler.go          # Chứa mọi API của Auth (Login, Register, Refresh...)
│   │   │   ├── service.go          # Logic nghiệp vụ của Auth
│   │   │   ├── repository.go       # Tương tác với DB (MongoDB/MySQL) của riêng Auth
│   │   │   ├── routes.go           # Định nghĩa router riêng cho Auth
│   │   │   └── model.go            # Struct riêng cho Auth (ví dụ LoginRequest, TokenResponse)
│   │   │
│   │   ├── user/                   # Tương tự Auth
│   │   ├── wallet/                 # Tương tự Auth
│   │   └── transaction/            # Tương tự Auth
│   │
│   ├── middleware/                 # Middleware dùng chung
│   │   ├── auth_middleware.go
│   │   └── rate_limiter.go
│   │
│   └── infrastructure/             # Tương tác với bên ngoài (Third-party)
│       ├── notification/
│       │   ├── email_client.go     # Thay vì để utils/email.go
│       │   └── sms_client.go       # Thay vì để utils/sms.go
│       └── jwt/
│           └── token_manager.go    # Chuyển từ internal/auth/jwt.go ra đây
│
├── pkg/                            # Các hàm utils thuần túy có thể dùng lại ở project khác
│   └── utils/
│       ├── validator.go
│       └── crypto.go               # Bcrypt hash...
│
├── .env
├── go.mod
└── go.sum