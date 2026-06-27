NFBank-Frontend/
├── src/
│   ├── app/                        # 1. Tầng ROUTING (Chỉ chứa cấu hình route, layout)
│   │   ├── (auth)/                 # Route groups (ngoặc đơn để không ảnh hưởng URL)
│   │   │   ├── login/page.tsx      # Điểm entry của trang login
│   │   │   └── register/page.tsx
│   │   ├── dashboard/
│   │   │   ├── layout.tsx          # Layout có Sidebar, Header cho user đã login
│   │   │   └── page.tsx            # Trang tổng quan số dư
│   │   ├── api/                    # (Tùy chọn) Next.js API Routes dùng làm BFF (Backend-for-Frontend) proxy
│   │   ├── layout.tsx              # Root Layout (Nơi bọc Providers: Redux/Query/Theme)
│   │   └── globals.css
│   │
│   ├── features/                   # 2. Tầng NGHIỆP VỤ (Trái tim của FSD)
│   │   ├── auth/                   # Mọi thứ về Auth nằm gọn ở đây
│   │   │   ├── components/         # LoginForm.tsx, OTPInput.tsx (Chỉ dùng cho auth)
│   │   │   ├── hooks/              # useAuth.ts, useLogin.ts
│   │   │   ├── services/           # auth.api.ts (Gọi API login, verify OTP)
│   │   │   └── store/              # auth.slice.ts (Zustand/Redux lưu thông tin user hiện tại)
│   │   │
│   │   ├── wallet/                 # Nghiệp vụ Ví/Số dư
│   │   ├── transaction/            # Nghiệp vụ Chuyển tiền, Lịch sử
│   │   └── user/                   # Nghiệp vụ Profile, Đổi mật khẩu
│   │
│   ├── shared/                     # 3. Tầng DÙNG CHUNG (Dumb Components & Utils)
│   │   ├── ui/                     # Các component UI thuần túy (Button, Input, Modal, Table) -> Thường dùng Shadcn/MUI
│   │   ├── hooks/                  # Các hooks không chứa logic nghiệp vụ (useDebounce, useClickOutside)
│   │   ├── utils/                  # formatCurrency, formatDate, regexValidation
│   │   └── types/                  # Các TypeScript Interfaces dùng toàn cục
│   │
│   └── core/                       # 4. Tầng CỐT LÕI HỆ THỐNG
│       ├── api/                    # Cấu hình Axios instance & Interceptors (Xử lý Refresh Token tự động)
│       ├── store/                  # Main store (Nơi gom các slices từ features lại)
│       └── theme/                  # Cấu hình biến màu sắc, font chữ
│
├── public/                         # Ảnh, logo, favicon
├── next.config.mjs
├── tailwind.config.ts
└── package.json