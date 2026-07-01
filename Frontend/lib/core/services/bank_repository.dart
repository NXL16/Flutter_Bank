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

  Future<String> uploadAvatar({
    required List<int> bytes,
    required String filename,
  }) async {
    final result = await ApiService.uploadFile(
      ApiUrl.profileAvatar,
      fieldName: 'file',
      bytes: bytes,
      filename: filename,
    );
    final data = _map(result.data);
    final avatarURL = data['avatar_url']?.toString() ?? '';
    if (!avatarURL.startsWith('https://')) {
      throw const ApiException('Server không trả về URL ảnh đại diện hợp lệ');
    }
    return avatarURL;
  }

  Future<List<Map<String, dynamic>>> transactions() async =>
      _list((await ApiService.get(ApiUrl.transactions, auth: true)).data);

  Future<Map<String, dynamic>> transaction(String reference) async => _map(
    (await ApiService.get(
      '${ApiUrl.transactions}/$reference',
      auth: true,
    )).data,
  );

  Future<Map<String, dynamic>> openSavings({
    required int amount,
    required int termMonths,
    required String maturityInstruction,
    required String transactionPin,
    required String idempotencyKey,
  }) async => _map(
    (await ApiService.post(
      ApiUrl.savings,
      auth: true,
      headers: {'Idempotency-Key': idempotencyKey},
      body: {
        'amount': amount,
        'term_months': termMonths,
        'maturity_instruction': maturityInstruction,
        'transaction_pin': transactionPin,
      },
    )).data,
  );

  Future<List<Map<String, dynamic>>> savingsProducts() async => _list(
    (await ApiService.get('${ApiUrl.savings}/products', auth: true)).data,
  );

  Future<List<Map<String, dynamic>>> savingsAccounts() async =>
      _list((await ApiService.get(ApiUrl.savings, auth: true)).data);

  Future<Map<String, dynamic>> withdrawSavingsEarly({
    required String accountNumber,
    required int amount,
    required String transactionPin,
    required String idempotencyKey,
  }) async => _map(
    (await ApiService.post(
      '${ApiUrl.savings}/$accountNumber/withdraw',
      auth: true,
      headers: {'Idempotency-Key': idempotencyKey},
      body: {'amount': amount, 'transaction_pin': transactionPin},
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
