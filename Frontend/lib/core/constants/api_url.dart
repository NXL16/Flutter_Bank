import 'package:flutter_dotenv/flutter_dotenv.dart';

class ApiUrl {
  ApiUrl._();

  static String get baseUrl {
    final value = dotenv.env['API_BASE_URL']?.trim();
    if (value == null || value.isEmpty) {
      throw StateError('Thiếu API_BASE_URL trong file .env');
    }
    return value.replaceFirst(RegExp(r'/+$'), '');
  }

  static String get auth => '$baseUrl/auth';
  static String get login => '$auth/login';
  static String get confirmLogin => '$auth/confirm-login';
  static String get register => '$auth/register';
  static String get logout => '$auth/logout';
  static String get refresh => '$auth/refresh';
  static String get changePassword => '$auth/change-password';
  static String get resetPassword => '$auth/password-reset/confirm';

  static String get accounts => '$baseUrl/accounts';
  static String get profile => '$baseUrl/users/me';
  static String get transactions => '$baseUrl/transactions';
  static String get transfer => '$transactions/transfer';
  static String get transactionPinStatus => '$transactions/pin/status';
  static String get setupTransactionPin => '$transactions/pin/setup';
  static String resolveAccount(String accountNumber) =>
      '$transactions/resolve/$accountNumber';
  static String get savings => '$baseUrl/savings';
  static String get notifications => '$baseUrl/notifications';
  static String get admin => '$baseUrl/admin';
}
