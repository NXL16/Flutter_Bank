package email

import (
	"fmt"
	"net/smtp"
	"time"

	"bank-service/internal/config"
)

type Sender struct {
	cfg *config.Config
}

func NewSender(cfg *config.Config) *Sender {
	return &Sender{
		cfg: cfg,
	}
}

// SendRegisterOTP gửi OTP đăng ký với tông màu xanh lá (Vibrant Green) chào mừng
func (s *Sender) SendRegisterOTP(toEmail string, otp string) error {
	auth := smtp.PlainAuth(
		"",
		s.cfg.SMTPUsername,
		s.cfg.SMTPPassword,
		s.cfg.SMTPHost,
	)

	subject := "Subject: [NF-Bank] Xac thuc dang ky tai khoan moi\r\nMIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n"

	body := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="utf-8">
		<style>
			body { font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; line-height: 1.6; color: #333333; margin: 0; padding: 20px; background-color: #f4f5f7; }
			.container { max-width: 550px; margin: 0 auto; border: 1px solid #e2e8f0; border-radius: 12px; padding: 30px; background-color: #ffffff; box-shadow: 0 4px 6px rgba(0,0,0,0.05); }
			.logo { font-size: 24px; font-weight: 800; color: #10b981; margin-bottom: 20px; text-transform: uppercase; letter-spacing: 1px; }
			h2 { color: #0f172a; font-size: 20px; font-weight: 700; margin-top: 0; }
			.otp-box { background-color: #f0fdf4; border: 1px dashed #10b981; padding: 20px; border-radius: 8px; text-align: center; margin: 25px 0; }
			.otp-code { font-size: 32px; font-weight: 800; color: #10b981; letter-spacing: 5px; }
			.footer { font-size: 12px; color: #64748b; margin-top: 30px; border-top: 1px solid #e2e8f0; paddingTop: 15px; }
		</style>
	</head>
	<body>
		<div class="container">
			<div class="logo">NF-Bank</div>
			<h2>Chào mừng thành viên mới!</h2>
			<p>Cảm ơn bạn đã lựa chọn dịch vụ ngân hàng số NF-Bank. Để hoàn tất quy trình đăng ký tài khoản của bạn, vui lòng nhập mã xác thực OTP dưới đây:</p>
			
			<div class="otp-box">
				<div class="otp-code">%s</div>
				<p style="margin: 5px 0 0 0; font-size: 13px; color: #10b981; font-weight: 600;">Mã OTP có hiệu lực trong 5 phút</p>
			</div>
			
			<p>Nếu bạn không thực hiện yêu cầu này, vui lòng bỏ qua email này hoặc liên hệ hotline để được hỗ trợ.</p>
			
			<div class="footer">
				<p>Email này được gửi tự động từ hệ thống bảo mật NF-Bank.</p>
				<p>Hotline khẩn cấp: 1900-XXXX | Website: nfbank.com</p>
			</div>
		</div>
	</body>
	</html>
	`, otp)

	message := []byte(subject + "\r\n" + body)
	addr := fmt.Sprintf("%s:%s", s.cfg.SMTPHost, s.cfg.SMTPPort)

	return smtp.SendMail(
		addr,
		auth,
		s.cfg.SMTPUsername,
		[]string{toEmail},
		message,
	)
}

// SendLoginOTP gửi OTP đăng nhập với tông màu xanh dương (Sleek Royal Blue) bảo mật
func (s *Sender) SendLoginOTP(toEmail string, otp string) error {
	auth := smtp.PlainAuth(
		"",
		s.cfg.SMTPUsername,
		s.cfg.SMTPPassword,
		s.cfg.SMTPHost,
	)

	subject := "Subject: [NF-Bank] Ma OTP xac nhan dang nhap\r\nMIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n"

	body := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="utf-8">
		<style>
			body { font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; line-height: 1.6; color: #333333; margin: 0; padding: 20px; background-color: #f4f5f7; }
			.container { max-width: 550px; margin: 0 auto; border: 1px solid #e2e8f0; border-radius: 12px; padding: 30px; background-color: #ffffff; box-shadow: 0 4px 6px rgba(0,0,0,0.05); }
			.logo { font-size: 24px; font-weight: 800; color: #3b82f6; margin-bottom: 20px; text-transform: uppercase; letter-spacing: 1px; }
			h2 { color: #0f172a; font-size: 20px; font-weight: 700; margin-top: 0; }
			.otp-box { background-color: #eff6ff; border: 1px dashed #3b82f6; padding: 20px; border-radius: 8px; text-align: center; margin: 25px 0; }
			.otp-code { font-size: 32px; font-weight: 800; color: #3b82f6; letter-spacing: 5px; }
			.footer { font-size: 12px; color: #64748b; margin-top: 30px; border-top: 1px solid #e2e8f0; paddingTop: 15px; }
		</style>
	</head>
	<body>
		<div class="container">
			<div class="logo">NF-Bank Security</div>
			<h2>Xác thực đăng nhập tài khoản</h2>
			<p>Hệ thống nhận thấy yêu cầu đăng nhập vào tài khoản của bạn. Vui lòng nhập mã OTP dưới đây để xác nhận phiên đăng nhập:</p>
			
			<div class="otp-box">
				<div class="otp-code">%s</div>
				<p style="margin: 5px 0 0 0; font-size: 13px; color: #3b82f6; font-weight: 600;">Mã OTP có hiệu lực trong 5 phút</p>
			</div>
			
			<p style="color: #64748b; font-size: 14px;">Vì an toàn tài khoản, tuyệt đối không cung cấp mã OTP này cho bất kỳ ai, kể cả nhân viên ngân hàng.</p>
			
			<div class="footer">
				<p>Email này được gửi tự động từ hệ thống bảo mật NF-Bank.</p>
				<p>Hotline khẩn cấp: 1900-XXXX | Website: nfbank.com</p>
			</div>
		</div>
	</body>
	</html>
	`, otp)

	message := []byte(subject + "\r\n" + body)
	addr := fmt.Sprintf("%s:%s", s.cfg.SMTPHost, s.cfg.SMTPPort)

	return smtp.SendMail(
		addr,
		auth,
		s.cfg.SMTPUsername,
		[]string{toEmail},
		message,
	)
}

// SendResetPasswordOTP gửi OTP reset password với tông đỏ/cam (Warning Red/Orange) cảnh báo
func (s *Sender) SendResetPasswordOTP(toEmail string, otp string) error {
	auth := smtp.PlainAuth(
		"",
		s.cfg.SMTPUsername,
		s.cfg.SMTPPassword,
		s.cfg.SMTPHost,
	)

	subject := "Subject: [NF-Bank] CANH BAO: Ma OTP dat lai mat khau\r\nMIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n"

	body := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="utf-8">
		<style>
			body { font-family: 'Helvetica Neue', Helvetica, Arial, sans-serif; line-height: 1.6; color: #333333; margin: 0; padding: 20px; background-color: #f4f5f7; }
			.container { max-width: 550px; margin: 0 auto; border: 1px solid #e2e8f0; border-radius: 12px; padding: 30px; background-color: #ffffff; box-shadow: 0 4px 6px rgba(0,0,0,0.05); border-top: 4px solid #ef4444; }
			.logo { font-size: 24px; font-weight: 800; color: #ef4444; margin-bottom: 20px; text-transform: uppercase; letter-spacing: 1px; }
			h2 { color: #ef4444; font-size: 20px; font-weight: 700; margin-top: 0; }
			.otp-box { background-color: #fef2f2; border: 1px dashed #ef4444; padding: 20px; border-radius: 8px; text-align: center; margin: 25px 0; }
			.otp-code { font-size: 32px; font-weight: 800; color: #ef4444; letter-spacing: 5px; }
			.footer { font-size: 12px; color: #64748b; margin-top: 30px; border-top: 1px solid #e2e8f0; paddingTop: 15px; }
		</style>
	</head>
	<body>
		<div class="container">
			<div class="logo">NF-Bank Alert</div>
			<h2>Yêu cầu đặt lại mật khẩu!</h2>
			<p>Hệ thống nhận được yêu cầu đặt lại mật khẩu cho tài khoản của bạn. Vui lòng sử dụng mã OTP dưới đây để hoàn tất thiết lập:</p>
			
			<div class="otp-box">
				<div class="otp-code">%s</div>
				<p style="margin: 5px 0 0 0; font-size: 13px; color: #ef4444; font-weight: 600;">Mã OTP có hiệu lực trong 5 phút</p>
			</div>
			
			<p style="color: #ef4444; font-weight: bold;">CẢNH BÁO BẢO MẬT: Nếu bạn không thực hiện yêu cầu này, tài khoản của bạn có thể đang bị xâm nhập trái phép. Vui lòng đổi mật khẩu ngay hoặc liên hệ quản trị viên.</p>
			
			<div class="footer">
				<p>Email này được gửi tự động từ hệ thống bảo mật NF-Bank.</p>
				<p>Hotline khẩn cấp: 1900-XXXX | Website: nfbank.com</p>
			</div>
		</div>
	</body>
	</html>
	`, otp)

	message := []byte(subject + "\r\n" + body)
	addr := fmt.Sprintf("%s:%s", s.cfg.SMTPHost, s.cfg.SMTPPort)

	return smtp.SendMail(
		addr,
		auth,
		s.cfg.SMTPUsername,
		[]string{toEmail},
		message,
	)
}

func (s *Sender) SendSecurityAlert(toEmail string) error {
	auth := smtp.PlainAuth(
		"",
		s.cfg.SMTPUsername,
		s.cfg.SMTPPassword,
		s.cfg.SMTPHost,
	)

	subject := "Subject: Canh bao bao mat NF-Bank\r\n"

	body := `
Phát hiện đăng nhập từ thiết bị khác.

Toàn bộ phiên đăng nhập đã bị vô hiệu hóa để bảo vệ tài khoản của bạn.

Nếu đây không phải là bạn, vui lòng liên hệ quản trị viên ngay lập tức.
`

	message := []byte(subject + "\r\n" + body)

	addr := fmt.Sprintf("%s:%s", s.cfg.SMTPHost, s.cfg.SMTPPort)

	return smtp.SendMail(
		addr,
		auth,
		s.cfg.SMTPUsername,
		[]string{toEmail},
		message,
	)
}

func (s *Sender) SendNewDeviceAlert(toEmail string, ipAddress string, location string, deviceName string, confirmURL string, rejectURL string) error {
	auth := smtp.PlainAuth(
		"",
		s.cfg.SMTPUsername,
		s.cfg.SMTPPassword,
		s.cfg.SMTPHost,
	)

	// Định nghĩa tiêu đề dạng HTML
	subject := "Subject: [NF-Bank] Canh bao dang nhap tu thiet bi moi\r\nMIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n"

	body := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="utf-8">
		<style>
			body { font-family: Arial, sans-serif; line-height: 1.6; color: #333333; margin: 0; padding: 20px; }
			.container { max-width: 600px; margin: 0 auto; border: 1px solid #dddddd; border-radius: 8px; padding: 20px; box-shadow: 0 4px 8px rgba(0,0,0,0.05); }
			h2 { color: #dc3545; }
			.info { background-color: #f8f9fa; padding: 15px; border-radius: 4px; margin-bottom: 20px; list-style-type: none; }
			.info li { margin-bottom: 10px; }
			.btn-group { margin: 30px 0; }
			.btn { display: inline-block; padding: 12px 24px; text-decoration: none; border-radius: 4px; font-weight: bold; font-size: 14px; text-align: center; }
			.btn-success { background-color: #28a745; color: #ffffff !important; margin-right: 15px; }
			.btn-danger { background-color: #dc3545; color: #ffffff !important; }
			.warning { color: #dc3545; font-weight: bold; margin-top: 25px; font-size: 12px; }
			.hotline { font-size: 16px; font-weight: bold; color: #dc3545; }
		</style>
	</head>
	<body>
		<div class="container">
			<h2>[NF-Bank] CẢNH BÁO BẢO MẬT: Phát hiện đăng nhập từ thiết bị lạ</h2>
			<p>Chào bạn,</p>
			<p>Hệ sinh thái bảo mật NF-Bank phát hiện tài khoản của bạn đang thực hiện đăng nhập từ một thiết bị mới chưa được xác minh.</p>
			
			<ul class="info">
				<li><strong>Thiết bị:</strong> %s</li>
				<li><strong>Địa chỉ IP:</strong> %s</li>
				<li><strong>Khu vực:</strong> %s</li>
				<li><strong>Thời gian:</strong> %s</li>
			</ul>
			
			<p>Vui lòng xác nhận danh tính của bạn để tiếp tục xử lý:</p>
			
			<div class="btn-group">
				<a href="%s" class="btn btn-success">Có, đúng là tôi</a>
				<a href="%s" class="btn btn-danger">Không phải tôi, KHÓA TÀI KHOẢN NGAY</a>
			</div>
			
			<p class="warning">
				*LƯU Ý: Nếu bạn chọn "Không phải tôi", tài khoản của bạn sẽ lập tức bị khóa và toàn bộ các phiên hoạt động trên các thiết bị khác sẽ bị đăng xuất khẩn cấp để bảo vệ tài sản của bạn.
			</p>
			<p>Nếu cần hỗ trợ kỹ thuật hoặc giải đáp thắc mắc, vui lòng gọi Hotline khẩn cấp: <span class="hotline">1900-XXXX</span></p>
		</div>
	</body>
	</html>
	`, deviceName, ipAddress, location, time.Now().Format("02/01/2006 15:04:05"), confirmURL, rejectURL)

	message := []byte(subject + "\r\n" + body)

	addr := fmt.Sprintf("%s:%s", s.cfg.SMTPHost, s.cfg.SMTPPort)

	return smtp.SendMail(
		addr,
		auth,
		s.cfg.SMTPUsername,
		[]string{toEmail},
		message,
	)
}
