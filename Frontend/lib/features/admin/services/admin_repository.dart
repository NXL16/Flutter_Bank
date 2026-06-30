import '../../../core/constants/api_url.dart';
import '../../../core/network/api_service.dart';

class AdminRepository {
  const AdminRepository();

  Future<List<Map<String, dynamic>>> users() async =>
      _list((await ApiService.get('${ApiUrl.admin}/users', auth: true)).data);

  Future<Map<String, dynamic>> user(int id) async => _map(
    (await ApiService.get('${ApiUrl.admin}/users/$id', auth: true)).data,
  );

  Future<void> setLocked(int id, {required bool locked}) async =>
      ApiService.patch(
        '${ApiUrl.admin}/users/$id/${locked ? 'lock' : 'unlock'}',
      );

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
  }) async => _map(
    (await ApiService.post(
      '${ApiUrl.admin}/deposit',
      auth: true,
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
  }) async => _map(
    (await ApiService.post(
      '${ApiUrl.admin}/create-admin',
      auth: true,
      body: {'full_name': fullName, 'phone': phone, 'password': password},
    )).data,
  );

  static Map<String, dynamic> _map(dynamic value) =>
      value is Map<String, dynamic> ? value : <String, dynamic>{};

  static List<Map<String, dynamic>> _list(dynamic value) => value is List
      ? value.whereType<Map<String, dynamic>>().toList()
      : <Map<String, dynamic>>[];
}
