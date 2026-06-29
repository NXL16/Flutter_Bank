import 'dart:convert';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';

class SessionUser {
  const SessionUser({
    required this.id,
    required this.fullName,
    required this.phone,
    required this.role,
    required this.isVerified,
  });

  final int id;
  final String fullName;
  final String phone;
  final String role;
  final bool isVerified;

  factory SessionUser.fromJson(Map<String, dynamic> json) => SessionUser(
    id: (json['id'] as num?)?.toInt() ?? 0,
    fullName: json['full_name']?.toString() ?? 'Người dùng',
    phone: json['phone']?.toString() ?? '',
    role: json['role']?.toString() ?? 'user',
    isVerified: json['is_verified'] == true,
  );

  Map<String, dynamic> toJson() => {
    'id': id,
    'full_name': fullName,
    'phone': phone,
    'role': role,
    'is_verified': isVerified,
  };
}

class TokenStorage {
  TokenStorage._();

  static const _accessTokenKey = 'ACCESS_TOKEN';
  static const _userKey = 'SESSION_USER';
  static const _cookieKey = 'HTTP_AUTH_COOKIES';
  static const _secureStorage = FlutterSecureStorage();

  static Future<void> saveAuthData(
    String token,
    Map<String, dynamic> user,
  ) async {
    await _secureStorage.write(key: _accessTokenKey, value: token);
    await _secureStorage.write(key: _userKey, value: jsonEncode(user));
  }

  static Future<String?> getToken() =>
      _secureStorage.read(key: _accessTokenKey);

  static Future<SessionUser?> getUser() async {
    final value = await _secureStorage.read(key: _userKey);
    if (value == null) return null;
    try {
      return SessionUser.fromJson(jsonDecode(value) as Map<String, dynamic>);
    } catch (_) {
      return null;
    }
  }

  static Future<String> getUserName() async =>
      (await getUser())?.fullName ?? 'Người dùng';

  static Future<void> clearAuth() async {
    await _secureStorage.delete(key: _accessTokenKey);
    await _secureStorage.delete(key: _userKey);
    await _secureStorage.delete(key: _cookieKey);

    // Xóa dữ liệu từ phiên bản cũ từng lưu token dạng rõ.
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_accessTokenKey);
    await prefs.remove(_userKey);
    await prefs.remove(_cookieKey);
  }
}
