import '../../../core/constants/api_url.dart';
import '../../../core/network/api_service.dart';
import '../../../core/storage/token_storage.dart';

enum LoginStep {
  authenticated,
  totpRequired,
  otpRequired,
  deviceApprovalRequired,
}

class LoginResult {
  const LoginResult(this.step, {this.pendingId, this.phone});

  final LoginStep step;
  final String? pendingId;
  final String? phone;
}

class AuthService {
  Future<LoginResult> login(
    String email,
    String password, {
    String totpCode = '',
  }) async {
    final result = await ApiService.post(
      ApiUrl.login,
      body: {'email': email, 'password': password, 'totp_code': totpCode},
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

  Future<LoginResult> confirmLogin(
    String email,
    String otp, {
    String idToken = '',
  }) async {
    final result = await ApiService.post(
      ApiUrl.confirmLogin,
      body: {'email': email, 'otp': otp, 'id_token': idToken},
    );
    final data = _map(result.data);
    if (data['pending_verification'] == true) {
      return LoginResult(
        LoginStep.deviceApprovalRequired,
        pendingId: data['pending_id']?.toString(),
      );
    }
    await _persist(data);
    return const LoginResult(LoginStep.authenticated);
  }

  Future<LoginResult> checkLoginStatus(String pendingId) async {
    final uri = Uri.parse(
      ApiUrl.loginStatus,
    ).replace(queryParameters: {'pending_id': pendingId});
    final result = await ApiService.get(uri.toString());
    final raw = _map(result.raw);
    final status = raw['status']?.toString();
    if (status == 'PENDING') {
      return LoginResult(
        LoginStep.deviceApprovalRequired,
        pendingId: pendingId,
      );
    }
    if (status == 'REJECTED' || status == 'EXPIRED') {
      throw ApiException(raw['message']?.toString() ?? 'Yêu cầu đã bị từ chối');
    }
    final data = _map(result.data);
    if (data['access_token'] != null) {
      await _persist(data);
      return const LoginResult(LoginStep.authenticated);
    }
    return LoginResult(LoginStep.deviceApprovalRequired, pendingId: pendingId);
  }

  Future<void> register(Map<String, dynamic> body) async =>
      ApiService.post(ApiUrl.register, body: body);

  Future<void> verifyRegister(
    String email,
    String otp, {
    String idToken = '',
  }) async => ApiService.post(
    ApiUrl.verifyRegister,
    body: {'email': email, 'otp': otp, 'id_token': idToken},
  );

  Future<void> forgotPassword(String email) async =>
      ApiService.post(ApiUrl.forgotPassword, body: {'email': email});

  Future<void> resetPassword(String email, String otp, String password) async =>
      ApiService.post(
        ApiUrl.resetPassword,
        body: {'email': email, 'otp': otp, 'new_password': password},
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
