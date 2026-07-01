class AdminUserSummary {
  const AdminUserSummary({
    required this.id,
    required this.fullName,
    required this.phone,
    required this.role,
    required this.isVerified,
    required this.isLocked,
    this.avatarUrl,
    required this.createdAt,
  });

  final int id;
  final String fullName;
  final String phone;
  final String role;
  final bool isVerified;
  final bool isLocked;
  final String? avatarUrl;
  final DateTime? createdAt;

  factory AdminUserSummary.fromJson(Map<String, dynamic> json) =>
      AdminUserSummary(
        id: (json['id'] as num?)?.toInt() ?? 0,
        fullName: json['full_name']?.toString() ?? 'Người dùng',
        phone: json['phone']?.toString() ?? '',
        role: json['role']?.toString() ?? 'user',
        isVerified: json['is_verified'] == true,
        isLocked: json['is_locked'] == true,
        avatarUrl: json['avatar_url']?.toString(),
        createdAt: DateTime.tryParse(json['created_at']?.toString() ?? ''),
      );
}

class AdminTransactionSummary {
  const AdminTransactionSummary({
    required this.referenceCode,
    required this.type,
    required this.amount,
    required this.currency,
    required this.status,
    required this.description,
    required this.createdAt,
  });

  final String referenceCode;
  final String type;
  final int amount;
  final String currency;
  final String status;
  final String description;
  final DateTime? createdAt;

  factory AdminTransactionSummary.fromJson(Map<String, dynamic> json) =>
      AdminTransactionSummary(
        referenceCode: json['reference_code']?.toString() ?? '—',
        type: json['type']?.toString() ?? 'UNKNOWN',
        amount: (json['amount'] as num?)?.toInt() ?? 0,
        currency: json['currency']?.toString() ?? 'VND',
        status: json['status']?.toString() ?? 'UNKNOWN',
        description: json['description']?.toString() ?? '',
        createdAt: DateTime.tryParse(json['created_at']?.toString() ?? ''),
      );
}

class AdminAuditLog {
  const AdminAuditLog({
    required this.actorName,
    required this.action,
    required this.targetType,
    required this.targetId,
    required this.summary,
    required this.ipAddress,
    required this.createdAt,
  });

  final String actorName;
  final String action;
  final String targetType;
  final String targetId;
  final String summary;
  final String ipAddress;
  final DateTime? createdAt;

  factory AdminAuditLog.fromJson(Map<String, dynamic> json) => AdminAuditLog(
    actorName:
        json['actor_name']?.toString() ??
        'Admin #${json['actor_user_id'] ?? '—'}',
    action: json['action']?.toString() ?? 'UNKNOWN',
    targetType: json['target_type']?.toString() ?? 'UNKNOWN',
    targetId: json['target_id']?.toString() ?? '—',
    summary: json['summary']?.toString() ?? '',
    ipAddress: json['ip_address']?.toString() ?? '',
    createdAt: DateTime.tryParse(json['created_at']?.toString() ?? ''),
  );
}

class AdminDashboard {
  const AdminDashboard({
    required this.customerCount,
    required this.adminCount,
    required this.lockedCustomerCount,
    required this.paymentBalance,
    required this.activeSavingsCount,
    required this.activeSavingsBalance,
    required this.todayTransactionCount,
    required this.todayTransactionValue,
    required this.recentUsers,
    required this.recentTransactions,
    required this.recentAuditLogs,
  });

  final int customerCount;
  final int adminCount;
  final int lockedCustomerCount;
  final int paymentBalance;
  final int activeSavingsCount;
  final int activeSavingsBalance;
  final int todayTransactionCount;
  final int todayTransactionValue;
  final List<AdminUserSummary> recentUsers;
  final List<AdminTransactionSummary> recentTransactions;
  final List<AdminAuditLog> recentAuditLogs;

  factory AdminDashboard.fromJson(Map<String, dynamic> json) => AdminDashboard(
    customerCount: (json['customer_count'] as num?)?.toInt() ?? 0,
    adminCount: (json['admin_count'] as num?)?.toInt() ?? 0,
    lockedCustomerCount: (json['locked_customer_count'] as num?)?.toInt() ?? 0,
    paymentBalance: (json['payment_balance'] as num?)?.toInt() ?? 0,
    activeSavingsCount: (json['active_savings_count'] as num?)?.toInt() ?? 0,
    activeSavingsBalance:
        (json['active_savings_balance'] as num?)?.toInt() ?? 0,
    todayTransactionCount:
        (json['today_transaction_count'] as num?)?.toInt() ?? 0,
    todayTransactionValue:
        (json['today_transaction_value'] as num?)?.toInt() ?? 0,
    recentUsers: _items(json['recent_users'], AdminUserSummary.fromJson),
    recentTransactions: _items(
      json['recent_transactions'],
      AdminTransactionSummary.fromJson,
    ),
    recentAuditLogs: _items(json['recent_audit_logs'], AdminAuditLog.fromJson),
  );

  static List<T> _items<T>(
    dynamic value,
    T Function(Map<String, dynamic>) fromJson,
  ) => value is List
      ? value.whereType<Map<String, dynamic>>().map(fromJson).toList()
      : <T>[];
}
