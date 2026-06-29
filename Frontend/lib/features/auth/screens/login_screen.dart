import 'package:flutter/material.dart';

import '../../../core/network/api_service.dart';
import '../../../shared/widgets/common.dart';
import '../../shell/screens/role_gate.dart';
import '../services/auth_service.dart';
import '../services/phone_auth_service.dart';
import 'register_screen.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _form = GlobalKey<FormState>();
  final _phone = TextEditingController();
  final _password = TextEditingController();
  final _totp = TextEditingController();
  final _otp = TextEditingController();
  final _auth = AuthService();
  final _phoneAuth = PhoneAuthService();
  LoginStep? _step;
  String? _verificationId;
  bool _loading = false;
  bool _passwordHidden = true;

  @override
  void dispose() {
    _phone.dispose();
    _password.dispose();
    _totp.dispose();
    _otp.dispose();
    super.dispose();
  }

  Future<void> _login({String totp = ''}) async {
    if (_step == null && !_form.currentState!.validate()) return;
    await _run(() async {
      final result = await _auth.login(
        _phone.text.trim(),
        _password.text,
        totpCode: totp,
      );
      if (result.step == LoginStep.authenticated) return _openApp();
      if (result.step == LoginStep.otpRequired) {
        _verificationId = await _phoneAuth.sendCode(
          result.phone ?? _phone.text,
        );
        if (_verificationId!.startsWith('AUTO:')) {
          final token = await _phoneAuth.verifyCode(_verificationId!, '');
          await _auth.confirmLogin(_phone.text.trim(), token);
          return _openApp();
        }
      }
      setState(() => _step = result.step);
    });
  }

  Future<void> _confirmSms() async {
    final id = _verificationId;
    if (id == null) return;
    await _run(() async {
      final token = await _phoneAuth.verifyCode(id, _otp.text);
      await _auth.confirmLogin(_phone.text.trim(), token);
      _openApp();
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

  void _openApp() {
    if (!mounted) return;
    Navigator.of(context).pushAndRemoveUntil(
      MaterialPageRoute(builder: (_) => const RoleGate()),
      (_) => false,
    );
  }

  Future<void> _showResetPassword() async {
    final phone = TextEditingController(text: _phone.text);
    final password = TextEditingController();
    final otp = TextEditingController();
    String? verificationId;
    bool busy = false;
    await showDialog<void>(
      context: context,
      builder: (dialogContext) => StatefulBuilder(
        builder: (context, setDialogState) => AlertDialog(
          title: const Text('Đặt lại mật khẩu'),
          content: SizedBox(
            width: 400,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextField(
                  controller: phone,
                  keyboardType: TextInputType.phone,
                  enabled: verificationId == null,
                  decoration: fieldDecoration('Số điện thoại'),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: password,
                  obscureText: true,
                  decoration: fieldDecoration('Mật khẩu mới'),
                ),
                if (verificationId != null) ...[
                  const SizedBox(height: 12),
                  TextField(
                    controller: otp,
                    keyboardType: TextInputType.number,
                    maxLength: 6,
                    decoration: fieldDecoration('Mã OTP SMS'),
                  ),
                ],
              ],
            ),
          ),
          actions: [
            TextButton(
              onPressed: busy ? null : () => Navigator.pop(context),
              child: const Text('Đóng'),
            ),
            FilledButton(
              onPressed: busy
                  ? null
                  : () async {
                      setDialogState(() => busy = true);
                      try {
                        verificationId ??= await _phoneAuth.sendCode(
                          phone.text,
                        );
                        if (verificationId!.startsWith('AUTO:') ||
                            otp.text.length == 6) {
                          final token = await _phoneAuth.verifyCode(
                            verificationId!,
                            otp.text,
                          );
                          await _auth.resetPassword(
                            phone.text,
                            token,
                            password.text,
                          );
                          if (context.mounted) Navigator.pop(context);
                          if (mounted) {
                            showMessage(
                              this.context,
                              'Đặt lại mật khẩu thành công',
                            );
                          }
                        }
                      } on ApiException catch (error) {
                        if (context.mounted) {
                          showMessage(context, error.message, error: true);
                        }
                      } finally {
                        if (context.mounted) {
                          setDialogState(() => busy = false);
                        }
                      }
                    },
              child: Text(
                verificationId == null ? 'Gửi mã SMS' : 'Đổi mật khẩu',
              ),
            ),
          ],
        ),
      ),
    );
    phone.dispose();
    password.dispose();
    otp.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    body: Center(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(28),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 430),
          child: Form(
            key: _form,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                const Icon(Icons.account_balance_rounded, size: 52),
                const SizedBox(height: 18),
                Text(
                  _step == null
                      ? 'Chào mừng trở lại'
                      : _step == LoginStep.totpRequired
                      ? 'Xác thực quản trị viên'
                      : 'Xác thực số điện thoại',
                  style: Theme.of(context).textTheme.headlineMedium,
                ),
                const SizedBox(height: 8),
                Text(
                  _step == LoginStep.otpRequired
                      ? 'Nhập mã SMS 6 số vừa được gửi tới điện thoại.'
                      : 'Đăng nhập an toàn bằng số điện thoại của bạn.',
                  style: const TextStyle(color: Colors.white60),
                ),
                const SizedBox(height: 28),
                if (_step == null) ...[
                  TextFormField(
                    controller: _phone,
                    keyboardType: TextInputType.phone,
                    decoration: fieldDecoration('Số điện thoại'),
                    validator: _validatePhone,
                  ),
                  const SizedBox(height: 14),
                  TextFormField(
                    controller: _password,
                    obscureText: _passwordHidden,
                    decoration: fieldDecoration('Mật khẩu').copyWith(
                      suffixIcon: IconButton(
                        onPressed: () =>
                            setState(() => _passwordHidden = !_passwordHidden),
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
                      onPressed: _showResetPassword,
                      child: const Text('Quên mật khẩu?'),
                    ),
                  ),
                  FilledButton(
                    onPressed: _loading ? null : _login,
                    child: _progressOr('Đăng nhập'),
                  ),
                  const SizedBox(height: 12),
                  OutlinedButton(
                    onPressed: _loading
                        ? null
                        : () => Navigator.push(
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
                    keyboardType: TextInputType.number,
                    maxLength: 6,
                    textAlign: TextAlign.center,
                    decoration: fieldDecoration('Mã TOTP'),
                  ),
                  FilledButton(
                    onPressed: _loading
                        ? null
                        : () => _login(totp: _totp.text.trim()),
                    child: _progressOr('Xác thực và đăng nhập'),
                  ),
                ] else ...[
                  TextField(
                    controller: _otp,
                    autofocus: true,
                    keyboardType: TextInputType.number,
                    maxLength: 6,
                    textAlign: TextAlign.center,
                    style: const TextStyle(fontSize: 25, letterSpacing: 8),
                    decoration: fieldDecoration('Mã OTP'),
                  ),
                  FilledButton(
                    onPressed: _loading ? null : _confirmSms,
                    child: _progressOr('Xác thực'),
                  ),
                  TextButton(
                    onPressed: _loading
                        ? null
                        : () async {
                            await _run(() async {
                              _verificationId = await _phoneAuth.sendCode(
                                _phone.text,
                              );
                            });
                          },
                    child: const Text('Gửi lại mã'),
                  ),
                ],
                if (_step != null)
                  TextButton(
                    onPressed: _loading
                        ? null
                        : () => setState(() {
                            _step = null;
                            _verificationId = null;
                            _otp.clear();
                            _totp.clear();
                          }),
                    child: const Text('Quay lại'),
                  ),
              ],
            ),
          ),
        ),
      ),
    ),
  );

  Widget _progressOr(String label) => _loading
      ? const SizedBox.square(
          dimension: 20,
          child: CircularProgressIndicator(strokeWidth: 2),
        )
      : Text(label);

  String? _validatePhone(String? value) {
    final digits = (value ?? '').replaceAll(RegExp(r'\D'), '');
    return RegExp(r'^(0|84)?(3|5|7|8|9)\d{8}$').hasMatch(digits)
        ? null
        : 'Số điện thoại Việt Nam không hợp lệ';
  }
}
