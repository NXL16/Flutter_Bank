class AccountResolution {
  const AccountResolution({
    required this.accountNumber,
    required this.accountName,
    required this.avatarUrl,
    required this.bankName,
    required this.currency,
  });

  final String accountNumber;
  final String accountName;
  final String avatarUrl;
  final String bankName;
  final String currency;

  factory AccountResolution.fromJson(Map<String, dynamic> json) =>
      AccountResolution(
        accountNumber: json['account_number']?.toString() ?? '',
        accountName: json['account_name']?.toString() ?? '',
        avatarUrl: json['avatar_url']?.toString().trim() ?? '',
        bankName: json['bank_name']?.toString() ?? 'NF Bank',
        currency: json['currency']?.toString() ?? 'VND',
      );
}
