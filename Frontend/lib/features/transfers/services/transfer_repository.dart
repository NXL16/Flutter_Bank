import '../../../core/constants/api_url.dart';
import '../../../core/network/api_service.dart';
import '../../../shared/utils/idempotency.dart' as idempotency;
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
    required String transactionPin,
  }) async {
    final result = await ApiService.post(
      ApiUrl.transfer,
      auth: true,
      headers: {'Idempotency-Key': idempotencyKey},
      body: {
        'receiver_account_number': accountNumber,
        'amount': amount,
        'description': description,
        'transaction_pin': transactionPin,
      },
    );
    return TransferReceipt.fromJson(_map(result.data));
  }

  Future<bool> hasTransactionPin() async {
    final result = await ApiService.get(
      ApiUrl.transactionPinStatus,
      auth: true,
    );
    return _map(result.data)['has_pin'] == true;
  }

  Future<void> setupTransactionPin(String pin, String confirmPin) async {
    await ApiService.post(
      ApiUrl.setupTransactionPin,
      auth: true,
      body: {'pin': pin, 'confirm_pin': confirmPin},
    );
  }

  static String createIdempotencyKey() {
    return idempotency.createIdempotencyKey();
  }

  static Map<String, dynamic> _map(dynamic value) =>
      value is Map<String, dynamic> ? value : <String, dynamic>{};
}
