import 'package:flutter_test/flutter_test.dart';
import 'package:nf_bank/features/transfers/models/account_resolution.dart';
import 'package:nf_bank/features/transfers/models/transfer_receipt.dart';
import 'package:nf_bank/features/transfers/services/transfer_repository.dart';
import 'package:nf_bank/shared/utils/formatters.dart';

void main() {
  test('idempotency keys are valid and unique', () {
    final first = TransferRepository.createIdempotencyKey();
    final second = TransferRepository.createIdempotencyKey();

    expect(first, isNot(second));
    expect(RegExp(r'^[A-Za-z0-9._:-]{16,64}$').hasMatch(first), isTrue);
  });

  test('transfer contract models parse backend payload', () {
    final account = AccountResolution.fromJson({
      'account_number': '970412345678',
      'account_name': 'Nguyen Van A',
      'avatar_url': 'https://res.cloudinary.com/nfbank/avatar.jpg',
      'bank_name': 'NF Bank',
      'currency': 'VND',
    });
    final receipt = TransferReceipt.fromJson({
      'reference_code': 'TRX123',
      'amount': 150000,
      'currency': 'VND',
      'status': 'SUCCESS',
      'description': 'Thanh toán',
      'created_at': '2026-06-30T00:00:00Z',
    });

    expect(account.accountName, 'Nguyen Van A');
    expect(account.avatarUrl, startsWith('https://'));
    expect(receipt.amount, 150000);
    expect(receipt.createdAt, isNotNull);
  });

  test('default transfer description removes Vietnamese diacritics', () {
    expect(
      '${removeVietnameseDiacritics('Nguyễn Xuân Linh')} chuyen khoan',
      'Nguyen Xuan Linh chuyen khoan',
    );
    expect(removeVietnameseDiacritics('Đào Thành Nhân'), 'Dao Thanh Nhan');
  });
}
