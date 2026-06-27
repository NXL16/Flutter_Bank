import 'dart:async';

import 'package:flutter/material.dart';

import '../../../core/network/api_service.dart';
import '../../../shared/widgets/common.dart';
import '../../shell/screens/role_gate.dart';
import '../services/auth_service.dart';
import 'register_screen.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _formKey = GlobalKey<FormState>();
  final _email = TextEditingController();
  final _password = TextEditingController();
  final _totp = TextEditingController();
  final _otp = TextEditingController();
  final _service = AuthService();
  bool _loading = false;
  bool _passwordHidden = true;
  LoginStep? _step;
  String? _pendingId;
  Timer? _pollTimer;

  @override
  void dispose() {
    _pollTimer?.cancel();
    _email.dispose();
    _password.dispose();
    _totp.dispose();
    _otp.dispose();
    super.dispose();
  }

  Future<void> _login() async {
    if (!_formKey.currentState!.validate()) return;
    await _run(() async {
      final result = await _service.login(_email.text.trim(), _password.text);
      if (result.step == LoginStep.authenticated) return _openApp();
      setState(() => _step = result.step);
    });
  }

  Future<void> _confirmTotp() async {
    if (_totp.text.trim().length != 6) {
      showMessage(context, 'Vui lòng nhập mã TOTP gồm 6 số', error: true);
      return;
    }
    await _run(() async {
      final result = await _service.login(
        _email.text.trim(),
        _password.text,
        totpCode: _totp.text.trim(),
      );
      if (result.step == LoginStep.authenticated) return _openApp();
      setState(() => _step = result.step);
    });
  }

  Future<void> _confirmOtp() async {
    if (_otp.text.trim().isEmpty) {
      showMessage(context, 'Vui lòng nhập mã OTP', error: true);
      return;
    }
    await _run(() async {
      final result = await _service.confirmLogin(
        _email.text.trim(),
        _otp.text.trim(),
      );
      if (result.step == LoginStep.authenticated) return _openApp();
      setState(() {
        _step = LoginStep.deviceApprovalRequired;
        _pendingId = result.pendingId;
      });
      _startPolling();
    });
  }

  void _startPolling() {
    _pollTimer?.cancel();
    _pollTimer = Timer.periodic(const Duration(seconds: 3), (_) async {
      if (_pendingId == null || !mounted) return;
      try {
        final result = await _service.checkLoginStatus(_pendingId!);
        if (result.step == LoginStep.authenticated) {
          _pollTimer?.cancel();
          _openApp();
        }
      } catch (error) {
        _pollTimer?.cancel();
        if (mounted) showMessage(context, '$error', error: true);
      }
    });
  }

  Future<void> _run(Future<void> Function() action) async {
    setState(() => _loading = true);
    try {
      await action();
    } on ApiException catch (error) {
      if (mounted) showMessage(context, error.message, error: true);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  void _openApp() {
    if (!mounted) return;
    Navigator.of(context).pushAndRemoveUntil(
      MaterialPageRoute(builder: (_) => const RoleGate()),
      (_) => false,
    );
  }

  void _showForgotPassword() {
    final email = TextEditingController(text: _email.text);
    final otp = TextEditingController();
    final password = TextEditingController();
    var requested = false;
    showDialog<void>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setDialogState) => AlertDialog(
          title: Text(requested ? 'Đặt lại mật khẩu' : 'Quên mật khẩu'),
          content: SizedBox(
            width: 400,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextField(
                  controller: email,
                  decoration: fieldDecoration('Email'),
                ),
                if (requested) ...[
                  const SizedBox(height: 12),
                  TextField(
                    controller: otp,
                    decoration: fieldDecoration('OTP 6 số'),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: password,
                    obscureText: true,
                    decoration: fieldDecoration('Mật khẩu mới'),
                  ),
                ],
              ],
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('Đóng'),
            ),
            FilledButton(
              onPressed: () async {
                try {
                  if (!requested) {
                    await _service.forgotPassword(email.text.trim());
                    setDialogState(() => requested = true);
                  } else {
                    await _service.resetPassword(
                      email.text.trim(),
                      otp.text.trim(),
                      password.text,
                    );
                    if (context.mounted) Navigator.pop(context);
                    if (mounted) showMessage(this.context, 'Đã đổi mật khẩu');
                  }
                } on ApiException catch (error) {
                  if (context.mounted) {
                    showMessage(context, error.message, error: true);
                  }
                }
              },
              child: Text(requested ? 'Xác nhận' : 'Gửi OTP'),
            ),
          ],
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    body: Row(
      children: [
        if (MediaQuery.sizeOf(context).width >= 900)
          Expanded(
            child: Container(
              padding: const EdgeInsets.all(64),
              decoration: const BoxDecoration(
                gradient: LinearGradient(
                  begin: Alignment.topLeft,
                  end: Alignment.bottomRight,
                  colors: [Color(0xFF202A78), Color(0xFF121527)],
                ),
              ),
              child: const Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisAlignment: MainAxisAlignment.end,
                children: [
                  Icon(Icons.account_balance_rounded, size: 56),
                  SizedBox(height: 24),
                  Text(
                    'Ngân hàng số,\ngọn gàng và an toàn.',
                    style: TextStyle(
                      fontSize: 42,
                      height: 1.15,
                      fontWeight: FontWeight.w800,
                    ),
                  ),
                  SizedBox(height: 16),
                  Text(
                    'Quản lý tài khoản, chuyển tiền, tiết kiệm và thanh toán trong một nơi.',
                    style: TextStyle(color: Colors.white70, fontSize: 17),
                  ),
                ],
              ),
            ),
          ),
        Expanded(
          child: Center(
            child: SingleChildScrollView(
              padding: const EdgeInsets.all(28),
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 430),
                child: Form(
                  key: _formKey,
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      const Text(
                        'NF Bank',
                        style: TextStyle(
                          color: Color(0xFF8E9BFF),
                          fontWeight: FontWeight.w800,
                          fontSize: 18,
                        ),
                      ),
                      const SizedBox(height: 14),
                      Text(
                        _step == null
                            ? 'Chào mừng trở lại'
                            : _step == LoginStep.totpRequired
                            ? 'Xác thực quản trị viên'
                            : _step == LoginStep.otpRequired
                            ? 'Xác thực đăng nhập'
                            : 'Xác nhận thiết bị mới',
                        style: Theme.of(context).textTheme.headlineMedium,
                      ),
                      const SizedBox(height: 8),
                      Text(
                        _step == LoginStep.deviceApprovalRequired
                            ? 'Mở email và chấp thuận thiết bị. Màn hình sẽ tự cập nhật.'
                            : _step == LoginStep.totpRequired
                            ? 'Nhập mã 6 số hiện tại từ ứng dụng xác thực của bạn.'
                            : 'Đăng nhập để tiếp tục với tài khoản của bạn.',
                        style: const TextStyle(color: Colors.white60),
                      ),
                      const SizedBox(height: 28),
                      if (_step == null) ...[
                        TextFormField(
                          controller: _email,
                          decoration: fieldDecoration('Email'),
                          validator: (value) =>
                              value == null || !value.contains('@')
                              ? 'Email không hợp lệ'
                              : null,
                        ),
                        const SizedBox(height: 14),
                        TextFormField(
                          controller: _password,
                          obscureText: _passwordHidden,
                          decoration: fieldDecoration('Mật khẩu').copyWith(
                            suffixIcon: IconButton(
                              onPressed: () => setState(
                                () => _passwordHidden = !_passwordHidden,
                              ),
                              icon: Icon(
                                _passwordHidden
                                    ? Icons.visibility_outlined
                                    : Icons.visibility_off_outlined,
                              ),
                            ),
                          ),
                          validator: (value) => (value?.length ?? 0) < 8
                              ? 'Mật khẩu tối thiểu 8 ký tự'
                              : null,
                        ),
                        Align(
                          alignment: Alignment.centerRight,
                          child: TextButton(
                            onPressed: _showForgotPassword,
                            child: const Text('Quên mật khẩu?'),
                          ),
                        ),
                        FilledButton(
                          onPressed: _loading ? null : _login,
                          child: _loading
                              ? const SizedBox.square(
                                  dimension: 20,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                  ),
                                )
                              : const Text('Đăng nhập'),
                        ),
                        const SizedBox(height: 12),
                        OutlinedButton(
                          onPressed: () => Navigator.push(
                            context,
                            MaterialPageRoute(
                              builder: (_) => const RegisterScreen(),
                            ),
                          ),
                          child: const Text('Tạo tài khoản mới'),
                        ),
                      ] else if (_step == LoginStep.totpRequired) ...[
                        TextField(
                          controller: _totp,
                          autofocus: true,
                          textAlign: TextAlign.center,
                          keyboardType: TextInputType.number,
                          maxLength: 6,
                          style: const TextStyle(
                            fontSize: 26,
                            letterSpacing: 8,
                          ),
                          decoration: fieldDecoration('Mã TOTP'),
                        ),
                        const SizedBox(height: 10),
                        FilledButton.icon(
                          onPressed: _loading ? null : _confirmTotp,
                          icon: const Icon(Icons.verified_user_outlined),
                          label: const Text('Xác thực và đăng nhập'),
                        ),
                        TextButton(
                          onPressed: () {
                            _totp.clear();
                            setState(() => _step = null);
                          },
                          child: const Text('Quay lại'),
                        ),
                      ] else if (_step == LoginStep.otpRequired) ...[
                        TextField(
                          controller: _otp,
                          autofocus: true,
                          textAlign: TextAlign.center,
                          keyboardType: TextInputType.number,
                          style: const TextStyle(
                            fontSize: 26,
                            letterSpacing: 8,
                          ),
                          decoration: fieldDecoration('Mã OTP'),
                        ),
                        const SizedBox(height: 18),
                        FilledButton(
                          onPressed: _loading ? null : _confirmOtp,
                          child: const Text('Xác thực'),
                        ),
                        TextButton(
                          onPressed: () => setState(() => _step = null),
                          child: const Text('Quay lại'),
                        ),
                      ] else ...[
                        const Center(
                          child: Padding(
                            padding: EdgeInsets.all(28),
                            child: CircularProgressIndicator(),
                          ),
                        ),
                        OutlinedButton.icon(
                          onPressed: _pendingId == null
                              ? null
                              : () async {
                                  try {
                                    final result = await _service
                                        .checkLoginStatus(_pendingId!);
                                    if (result.step ==
                                        LoginStep.authenticated) {
                                      _openApp();
                                    }
                                  } on ApiException catch (error) {
                                    if (!context.mounted) return;
                                    showMessage(
                                      context,
                                      error.message,
                                      error: true,
                                    );
                                  }
                                },
                          icon: const Icon(Icons.refresh),
                          label: const Text('Kiểm tra lại'),
                        ),
                      ],
                    ],
                  ),
                ),
              ),
            ),
          ),
        ),
      ],
    ),
  );
}
