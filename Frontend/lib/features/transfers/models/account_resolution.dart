class AccountResolution {
  const AccountResolution({
    required this.accountNumber,
    required this.accountName,
    required this.bankName,
    required this.currency,
  });

  final String accountNumber;
  final String accountName;
  final String bankName;
  final String currency;

  factory AccountResolution.fromJson(Map<String, dynamic> json) =>
      AccountResolution(
        accountNumber: json['account_number']?.toString() ?? '',
        accountName: json['account_name']?.toString() ?? '',
        bankName: json['bank_name']?.toString() ?? 'NF Bank',
        currency: json['currency']?.toString() ?? 'VND',
      );
}
