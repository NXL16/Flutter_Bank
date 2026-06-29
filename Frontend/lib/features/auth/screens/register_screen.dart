import 'package:flutter/material.dart';

import '../../../core/network/api_service.dart';
import '../../../shared/widgets/common.dart';
import '../services/auth_service.dart';
import '../services/phone_auth_service.dart';

class RegisterScreen extends StatefulWidget {
  const RegisterScreen({super.key});

  @override
  State<RegisterScreen> createState() => _RegisterScreenState();
}

class _RegisterScreenState extends State<RegisterScreen> {
  final _form = GlobalKey<FormState>();
  final _name = TextEditingController();
  final _phone = TextEditingController();
  final _password = TextEditingController();
  final _otp = TextEditingController();
  final _auth = AuthService();
  final _phoneAuth = PhoneAuthService();
  String? _verificationId;
  bool _loading = false;

  @override
  void dispose() {
    _name.dispose();
    _phone.dispose();
    _password.dispose();
    _otp.dispose();
    super.dispose();
  }

  Future<void> _sendCode() async {
    if (!_form.currentState!.validate()) return;
    await _run(() async {
      final id = await _phoneAuth.sendCode(_phone.text);
      setState(() => _verificationId = id);
      if (id.startsWith('AUTO:')) {
        final token = await _phoneAuth.verifyCode(id, '');
        await _auth.register(
          fullName: _name.text.trim(),
          phone: _phone.text.trim(),
          password: _password.text,
          idToken: token,
        );
        if (!mounted) return;
        showMessage(context, 'Đăng ký thành công, bạn có thể đăng nhập');
        Navigator.pop(context);
      }
    });
  }

  Future<void> _completeRegistration() async {
    final id = _verificationId;
    if (id == null) return;
    await _run(() async {
      final token = await _phoneAuth.verifyCode(id, _otp.text);
      await _auth.register(
        fullName: _name.text.trim(),
        phone: _phone.text.trim(),
        password: _password.text,
        idToken: token,
      );
      if (!mounted) return;
      showMessage(context, 'Đăng ký thành công, bạn có thể đăng nhập');
      Navigator.pop(context);
    });
  }

  Future<void> _run(Future<void> Function() action) async {
    if (_loading) return;
    setState(() => _loading = true);
    try {
      await action();
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
                    _verificationId == null
                        ? 'Tạo tài khoản'
                        : 'Xác thực số điện thoại',
                    style: Theme.of(context).textTheme.headlineSmall,
                  ),
                  const SizedBox(height: 8),
                  Text(
                    _verificationId == null
                        ? 'Số điện thoại sẽ là định danh đăng nhập của bạn.'
                        : 'Nhập mã SMS 6 số được gửi tới ${_phone.text}.',
                    style: const TextStyle(color: Colors.white60),
                  ),
                  const SizedBox(height: 24),
                  if (_verificationId == null) ...[
                    TextFormField(
                      controller: _name,
                      decoration: fieldDecoration('Họ và tên'),
                      validator: _required,
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _phone,
                      keyboardType: TextInputType.phone,
                      decoration: fieldDecoration('Số điện thoại'),
                      validator: _validatePhone,
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _password,
                      obscureText: true,
                      decoration: fieldDecoration('Mật khẩu'),
                      validator: (value) =>
                          RegExp(
                            r'^(?=.*[A-Z])(?=.*[!@#$%^&*(),.?":{}|<>]).{8,}$',
                          ).hasMatch(value ?? '')
                          ? null
                          : 'Tối thiểu 8 ký tự, có chữ hoa và ký tự đặc biệt',
                    ),
                  ] else
                    TextFormField(
                      controller: _otp,
                      autofocus: true,
                      keyboardType: TextInputType.number,
                      maxLength: 6,
                      textAlign: TextAlign.center,
                      style: const TextStyle(fontSize: 25, letterSpacing: 8),
                      decoration: fieldDecoration('Mã OTP'),
                    ),
                  const SizedBox(height: 22),
                  FilledButton(
                    onPressed: _loading
                        ? null
                        : _verificationId == null
                        ? _sendCode
                        : _completeRegistration,
                    child: _loading
                        ? const SizedBox.square(
                            dimension: 20,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : Text(
                            _verificationId == null
                                ? 'Gửi mã SMS'
                                : 'Xác thực và đăng ký',
                          ),
                  ),
                  if (_verificationId != null)
                    TextButton(
                      onPressed: _loading ? null : _sendCode,
                      child: const Text('Gửi lại mã'),
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

  String? _validatePhone(String? value) {
    final digits = (value ?? '').replaceAll(RegExp(r'\D'), '');
    return RegExp(r'^(0|84)?(3|5|7|8|9)\d{8}$').hasMatch(digits)
        ? null
        : 'Số điện thoại Việt Nam không hợp lệ';
  }
}
