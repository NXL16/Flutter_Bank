import '../constants/api_url.dart';
import '../network/api_service.dart';

class BankRepository {
  const BankRepository();

  Future<List<Map<String, dynamic>>> accounts() async =>
      _list((await ApiService.get(ApiUrl.accounts, auth: true)).data);

  Future<Map<String, dynamic>> profile() async =>
      _map((await ApiService.get(ApiUrl.profile, auth: true)).data);

  Future<void> updateProfile(Map<String, dynamic> body) async =>
      ApiService.put(ApiUrl.profile, body: body);

  Future<List<Map<String, dynamic>>> transactions() async =>
      _list((await ApiService.get(ApiUrl.transactions, auth: true)).data);

  Future<Map<String, dynamic>> transaction(String reference) async => _map(
    (await ApiService.get(
      '${ApiUrl.transactions}/$reference',
      auth: true,
    )).data,
  );

  Future<Map<String, dynamic>> transfer({
    required String accountNumber,
    required int amount,
    required String description,
    required String idToken,
  }) async => _map(
    (await ApiService.post(
      ApiUrl.transfer,
      auth: true,
      body: {
        'receiver_account_number': accountNumber,
        'amount': amount,
        'description': description,
        'id_token': idToken,
      },
    )).data,
  );

  Future<Map<String, dynamic>> openSavings(int amount) async => _map(
    (await ApiService.post(
      ApiUrl.savings,
      auth: true,
      body: {'amount': amount},
    )).data,
  );

  Future<List<Map<String, dynamic>>> notifications() async =>
      _list((await ApiService.get(ApiUrl.notifications, auth: true)).data);

  Future<void> markNotificationRead(int id) async =>
      ApiService.patch('${ApiUrl.notifications}/$id/read');

  Future<void> markAllNotificationsRead() async =>
      ApiService.patch('${ApiUrl.notifications}/read-all');

  static Map<String, dynamic> _map(dynamic value) =>
      value is Map<String, dynamic> ? value : <String, dynamic>{};

  static List<Map<String, dynamic>> _list(dynamic value) => value is List
      ? value.whereType<Map<String, dynamic>>().toList()
      : <Map<String, dynamic>>[];
}
