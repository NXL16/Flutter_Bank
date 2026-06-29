import '../../../core/constants/api_url.dart';
import '../../../core/network/api_service.dart';
import '../../../core/storage/token_storage.dart';

enum LoginStep { authenticated, totpRequired, otpRequired }

class LoginResult {
  const LoginResult(this.step, {this.phone});

  final LoginStep step;
  final String? phone;
}

class AuthService {
  Future<LoginResult> login(
    String phone,
    String password, {
    String totpCode = '',
  }) async {
    final result = await ApiService.post(
      ApiUrl.login,
      body: {'phone': phone, 'password': password, 'totp_code': totpCode},
    );
    final data = _map(result.data);
    if (data['totp_required'] == true) {
      return const LoginResult(LoginStep.totpRequired);
    }
    if (data['access_token'] != null) {
      await _persist(data);
      return const LoginResult(LoginStep.authenticated);
    }
    return LoginResult(LoginStep.otpRequired, phone: data['phone']?.toString());
  }

  Future<void> confirmLogin(String phone, String idToken) async {
    final result = await ApiService.post(
      ApiUrl.confirmLogin,
      body: {'phone': phone, 'id_token': idToken},
    );
    await _persist(_map(result.data));
  }

  Future<void> register({
    required String fullName,
    required String phone,
    required String password,
    required String idToken,
  }) async => ApiService.post(
    ApiUrl.register,
    body: {
      'full_name': fullName,
      'phone': phone,
      'password': password,
      'id_token': idToken,
    },
  );

  Future<void> resetPassword(
    String phone,
    String idToken,
    String password,
  ) async => ApiService.post(
    ApiUrl.resetPassword,
    body: {'phone': phone, 'id_token': idToken, 'new_password': password},
  );

  Future<void> changePassword(String oldPassword, String newPassword) async =>
      ApiService.put(
        ApiUrl.changePassword,
        body: {'old_password': oldPassword, 'new_password': newPassword},
      );

  Future<void> logout() async {
    try {
      await ApiService.post(ApiUrl.logout);
    } finally {
      await TokenStorage.clearAuth();
    }
  }

  Future<void> _persist(Map<String, dynamic> data) async {
    final token = data['access_token']?.toString();
    final user = data['user'];
    if (token == null || user is! Map<String, dynamic>) {
      throw const ApiException('Phản hồi đăng nhập không hợp lệ');
    }
    await TokenStorage.saveAuthData(token, user);
  }

  static Map<String, dynamic> _map(dynamic value) =>
      value is Map<String, dynamic> ? value : <String, dynamic>{};
}
