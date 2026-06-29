import 'dart:math';

import '../../../core/constants/api_url.dart';
import '../../../core/network/api_service.dart';
import '../models/account_resolution.dart';
import '../models/transfer_receipt.dart';

class TransferRepository {
  const TransferRepository();

  Future<AccountResolution> resolveAccount(String accountNumber) async {
    final result = await ApiService.get(
      ApiUrl.resolveAccount(accountNumber),
      auth: true,
    );
    return AccountResolution.fromJson(_map(result.data));
  }

  Future<TransferReceipt> transfer({
    required String accountNumber,
    required int amount,
    required String description,
    required String idempotencyKey,
    String idToken = '',
  }) async {
    final result = await ApiService.post(
      ApiUrl.transfer,
      auth: true,
      headers: {'Idempotency-Key': idempotencyKey},
      body: {
        'receiver_account_number': accountNumber,
        'amount': amount,
        'description': description,
        'id_token': idToken,
      },
    );
    return TransferReceipt.fromJson(_map(result.data));
  }

  static String createIdempotencyKey() {
    final random = Random.secure();
    final bytes = List<int>.generate(16, (_) => random.nextInt(256));
    final suffix = bytes
        .map((value) => value.toRadixString(16).padLeft(2, '0'))
        .join();
    return '${DateTime.now().microsecondsSinceEpoch}:$suffix';
  }

  static Map<String, dynamic> _map(dynamic value) =>
      value is Map<String, dynamic> ? value : <String, dynamic>{};
}
