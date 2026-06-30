class RecentRecipient {
  const RecentRecipient({
    required this.accountNumber,
    required this.accountName,
  });

  final String accountNumber;
  final String accountName;
}

List<RecentRecipient> recentRecipientsFromTransactions(
  List<Map<String, dynamic>> transactions, {
  int limit = 10,
}) {
  if (limit <= 0) return const [];

  final seenAccounts = <String>{};
  final recipients = <RecentRecipient>[];

  for (final transaction in transactions) {
    if (transaction['direction']?.toString().toUpperCase() != 'OUT' ||
        transaction['status']?.toString().toUpperCase() != 'SUCCESS' ||
        transaction['type']?.toString().toUpperCase() != 'TRANSFER') {
      continue;
    }

    final accountNumber =
        transaction['counterparty_account_number']?.toString().trim() ?? '';
    final accountName =
        transaction['counterparty_name']?.toString().trim() ?? '';
    if (!RegExp(r'^[0-9]{12}$').hasMatch(accountNumber) ||
        accountName.isEmpty ||
        !seenAccounts.add(accountNumber)) {
      continue;
    }

    recipients.add(
      RecentRecipient(accountNumber: accountNumber, accountName: accountName),
    );
    if (recipients.length == limit) break;
  }
  return recipients;
}
