class TransferReceipt {
  const TransferReceipt({
    required this.referenceCode,
    required this.amount,
    required this.currency,
    required this.status,
    required this.description,
    required this.createdAt,
  });

  final String referenceCode;
  final int amount;
  final String currency;
  final String status;
  final String description;
  final DateTime? createdAt;

  factory TransferReceipt.fromJson(Map<String, dynamic> json) =>
      TransferReceipt(
        referenceCode: json['reference_code']?.toString() ?? '',
        amount: (json['amount'] as num?)?.toInt() ?? 0,
        currency: json['currency']?.toString() ?? 'VND',
        status: json['status']?.toString() ?? '',
        description: json['description']?.toString() ?? '',
        createdAt: DateTime.tryParse(json['created_at']?.toString() ?? ''),
      );
}
