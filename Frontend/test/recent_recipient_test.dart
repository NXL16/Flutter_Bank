import 'package:flutter_test/flutter_test.dart';
import 'package:nf_bank/features/transfers/models/recent_recipient.dart';

void main() {
  test('keeps unique successful outgoing transfers in newest order', () {
    final recipients = recentRecipientsFromTransactions([
      _transaction('111111111111', 'Nguyen Van A'),
      _transaction('111111111111', 'Nguyen Van A'),
      _transaction('222222222222', 'Tran Thi B'),
      _transaction('333333333333', 'Giao dich den', direction: 'IN'),
      _transaction('444444444444', 'Giao dich loi', status: 'FAILED'),
      _transaction('555555555555', 'Thanh toan', type: 'PAYMENT_GATEWAY'),
    ]);

    expect(recipients, hasLength(2));
    expect(recipients[0].accountNumber, '111111111111');
    expect(recipients[1].accountName, 'Tran Thi B');
  });

  test('limits recent recipients to ten accounts', () {
    final transactions = List.generate(
      12,
      (index) =>
          _transaction((100000000000 + index).toString(), 'Nguoi nhan $index'),
    );

    final recipients = recentRecipientsFromTransactions(transactions);

    expect(recipients, hasLength(10));
    expect(recipients.first.accountNumber, '100000000000');
    expect(recipients.last.accountNumber, '100000000009');
  });
}

Map<String, dynamic> _transaction(
  String accountNumber,
  String accountName, {
  String direction = 'OUT',
  String status = 'SUCCESS',
  String type = 'TRANSFER',
}) => {
  'counterparty_account_number': accountNumber,
  'counterparty_name': accountName,
  'direction': direction,
  'status': status,
  'type': type,
};
