import 'package:flutter_test/flutter_test.dart';
import 'package:nf_bank/features/admin/models/admin_models.dart';

void main() {
  test('parses the admin dashboard contract', () {
    final dashboard = AdminDashboard.fromJson({
      'customer_count': 12,
      'admin_count': 2,
      'locked_customer_count': 1,
      'payment_balance': 45000000,
      'active_savings_count': 3,
      'active_savings_balance': 25000000,
      'today_transaction_count': 7,
      'today_transaction_value': 18000000,
      'recent_users': [
        {
          'id': 8,
          'full_name': 'Nguyen Van A',
          'phone': '+84901234567',
          'role': 'user',
          'is_verified': true,
          'is_locked': false,
          'created_at': '2026-07-01T08:00:00Z',
        },
      ],
      'recent_transactions': [
        {
          'reference_code': 'TRX123',
          'type': 'TRANSFER',
          'amount': 10000,
          'currency': 'VND',
          'status': 'SUCCESS',
          'description': 'Chuyen khoan',
          'created_at': '2026-07-01T08:01:00Z',
        },
      ],
      'recent_audit_logs': [
        {
          'action': 'LOCK_USER',
          'summary': 'Khóa tài khoản',
          'ip_address': '127.0.0.1',
          'created_at': '2026-07-01T08:02:00Z',
        },
      ],
    });

    expect(dashboard.customerCount, 12);
    expect(dashboard.paymentBalance, 45000000);
    expect(dashboard.recentUsers.single.fullName, 'Nguyen Van A');
    expect(dashboard.recentTransactions.single.referenceCode, 'TRX123');
    expect(dashboard.recentAuditLogs.single.action, 'LOCK_USER');
  });

  test('uses safe defaults for a partial dashboard response', () {
    final dashboard = AdminDashboard.fromJson(const {});

    expect(dashboard.customerCount, 0);
    expect(dashboard.recentUsers, isEmpty);
    expect(dashboard.recentTransactions, isEmpty);
    expect(dashboard.recentAuditLogs, isEmpty);
  });
}
