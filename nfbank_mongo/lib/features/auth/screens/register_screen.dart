import 'package:flutter/material.dart';

import '../../../core/network/api_service.dart';
import '../../../shared/widgets/common.dart';
import '../services/auth_service.dart';

class RegisterScreen extends StatefulWidget {
  const RegisterScreen({super.key});

  @override
  State<RegisterScreen> createState() => _RegisterScreenState();
}

class _RegisterScreenState extends State<RegisterScreen> {
  final _form = GlobalKey<FormState>();
  final _name = TextEditingController();
  final _email = TextEditingController();
  final _phone = TextEditingController();
  final _password = TextEditingController();
  final _otp = TextEditingController();
  final _service = AuthService();
  String _channel = 'email';
  bool _verifyStep = false;
  bool _loading = false;

  Future<void> _submit() async {
    if (!_form.currentState!.validate()) return;
    setState(() => _loading = true);
    try {
      if (!_verifyStep) {
        await _service.register({
          'full_name': _name.text.trim(),
          'email': _email.text.trim(),
          'phone': _phone.text.trim(),
          'password': _password.text,
          'otp_channel': _channel,
        });
        setState(() => _verifyStep = true);
        if (mounted) showMessage(context, 'OTP đã được gửi');
      } else {
        await _service.verifyRegister(_email.text.trim(), _otp.text.trim());
        if (mounted) {
          showMessage(context, 'Kích hoạt tài khoản thành công');
          Navigator.pop(context);
        }
      }
    } on ApiException catch (error) {
      if (mounted) showMessage(context, error.message, error: true);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(title: const Text('Đăng ký NF Bank')),
    body: Center(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 520),
          child: SurfaceCard(
            child: Form(
              key: _form,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    _verifyStep ? 'Kích hoạt tài khoản' : 'Tạo tài khoản',
                    style: Theme.of(context).textTheme.headlineSmall,
                  ),
                  const SizedBox(height: 24),
                  if (!_verifyStep) ...[
                    TextFormField(
                      controller: _name,
                      decoration: fieldDecoration('Họ và tên'),
                      validator: _required,
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _email,
                      decoration: fieldDecoration('Email'),
                      validator: (value) => value != null && value.contains('@')
                          ? null
                          : 'Email không hợp lệ',
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _phone,
                      decoration: fieldDecoration('Số điện thoại'),
                      validator: _required,
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _password,
                      obscureText: true,
                      decoration: fieldDecoration('Mật khẩu'),
                      validator: (value) => (value?.length ?? 0) >= 8
                          ? null
                          : 'Tối thiểu 8 ký tự',
                    ),
                    const SizedBox(height: 12),
                    DropdownButtonFormField<String>(
                      initialValue: _channel,
                      decoration: fieldDecoration('Kênh nhận OTP'),
                      items: const [
                        DropdownMenuItem(value: 'email', child: Text('Email')),
                        DropdownMenuItem(
                          value: 'sms',
                          child: Text('SMS / Firebase'),
                        ),
                      ],
                      onChanged: (value) =>
                          setState(() => _channel = value ?? 'email'),
                    ),
                  ] else ...[
                    Text(
                      'Nhập mã được gửi tới ${_email.text}',
                      style: const TextStyle(color: Colors.white60),
                    ),
                    const SizedBox(height: 16),
                    TextFormField(
                      controller: _otp,
                      textAlign: TextAlign.center,
                      style: const TextStyle(fontSize: 25, letterSpacing: 8),
                      decoration: fieldDecoration('OTP'),
                      validator: _required,
                    ),
                  ],
                  const SizedBox(height: 22),
                  FilledButton(
                    onPressed: _loading ? null : _submit,
                    child: Text(_verifyStep ? 'Kích hoạt' : 'Tiếp tục'),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    ),
  );

  String? _required(String? value) =>
      value == null || value.trim().isEmpty ? 'Không được để trống' : null;
}
