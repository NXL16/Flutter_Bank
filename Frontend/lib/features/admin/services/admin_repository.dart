import '../../../core/constants/api_url.dart';
import '../../../core/network/api_service.dart';
import '../models/admin_models.dart';

class AdminRepository {
  const AdminRepository();

  Future<String> createStepUp({
    required String action,
    required String totpCode,
    required String binding,
  }) async {
    final result = await ApiService.post(
      '${ApiUrl.admin}/step-up',
      auth: true,
      body: {
        'action': action,
        'totp_code': totpCode,
        'binding': binding,
      },
    );
    final token = _map(result.data)['token']?.toString() ?? '';
    if (token.isEmpty) {
      throw const ApiException('Server không trả về xác thực nâng cao hợp lệ');
    }
    return token;
  }

  Future<AdminDashboard> dashboard() async => AdminDashboard.fromJson(
    _map((await ApiService.get('${ApiUrl.admin}/dashboard', auth: true)).data),
  );

  Future<List<AdminUserSummary>> users() async => _list(
    (await ApiService.get('${ApiUrl.admin}/users', auth: true)).data,
  ).map(AdminUserSummary.fromJson).toList();

  Future<AdminUserSummary> user(int id) async => AdminUserSummary.fromJson(
    _map((await ApiService.get('${ApiUrl.admin}/users/$id', auth: true)).data),
  );

  Future<void> setLocked(
    int id, {
    required bool locked,
    required String stepUpToken,
  }) async => ApiService.patch(
    '${ApiUrl.admin}/users/$id/${locked ? 'lock' : 'unlock'}',
    headers: {'X-Admin-Step-Up': stepUpToken},
  );

  Future<List<AdminTransactionSummary>> transactions({int limit = 100}) async =>
      _list(
        (await ApiService.get(
          '${ApiUrl.admin}/transactions?limit=$limit',
          auth: true,
        )).data,
      ).map(AdminTransactionSummary.fromJson).toList();

  Future<List<AdminAuditLog>> auditLogs({int limit = 100}) async => _list(
    (await ApiService.get(
      '${ApiUrl.admin}/audit-logs?limit=$limit',
      auth: true,
    )).data,
  ).map(AdminAuditLog.fromJson).toList();

  Future<List<Map<String, dynamic>>> userAccounts(int id) async => _list(
    (await ApiService.get(
      '${ApiUrl.admin}/users/$id/accounts',
      auth: true,
    )).data,
  );

  Future<List<Map<String, dynamic>>> accountTransactions(int accountId) async =>
      _list(
        (await ApiService.get(
          '${ApiUrl.admin}/accounts/$accountId/transactions',
          auth: true,
        )).data,
      );

  Future<Map<String, dynamic>> deposit({
    required String accountNumber,
    required int amount,
    required String description,
    required String idempotencyKey,
    required String stepUpToken,
  }) async => _map(
    (await ApiService.post(
      '${ApiUrl.admin}/deposit',
      auth: true,
      headers: {
        'Idempotency-Key': idempotencyKey,
        'X-Admin-Step-Up': stepUpToken,
      },
      body: {
        'receiver_account_number': accountNumber,
        'amount': amount,
        'description': description,
      },
    )).data,
  );

  Future<Map<String, dynamic>> createAdmin({
    required String fullName,
    required String phone,
    required String password,
    required String stepUpToken,
  }) async => _map(
    (await ApiService.post(
      '${ApiUrl.admin}/create-admin',
      auth: true,
      headers: {'X-Admin-Step-Up': stepUpToken},
      body: {'full_name': fullName, 'phone': phone, 'password': password},
    )).data,
  );

  static Map<String, dynamic> _map(dynamic value) =>
      value is Map<String, dynamic> ? value : <String, dynamic>{};

  static List<Map<String, dynamic>> _list(dynamic value) => value is List
      ? value.whereType<Map<String, dynamic>>().toList()
      : <Map<String, dynamic>>[];
}
