import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

class SessionUser {
  const SessionUser({
    required this.id,
    required this.fullName,
    required this.email,
    required this.phone,
    required this.role,
    required this.isVerified,
  });

  final int id;
  final String fullName;
  final String email;
  final String phone;
  final String role;
  final bool isVerified;

  factory SessionUser.fromJson(Map<String, dynamic> json) => SessionUser(
    id: (json['id'] as num?)?.toInt() ?? 0,
    fullName: json['full_name']?.toString() ?? 'Người dùng',
    email: json['email']?.toString() ?? '',
    phone: json['phone']?.toString() ?? '',
    role: json['role']?.toString() ?? 'user',
    isVerified: json['is_verified'] == true,
  );

  Map<String, dynamic> toJson() => {
    'id': id,
    'full_name': fullName,
    'email': email,
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

  static Future<void> saveAuthData(
    String token,
    Map<String, dynamic> user,
  ) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_accessTokenKey, token);
    await prefs.setString(_userKey, jsonEncode(user));
  }

  static Future<String?> getToken() async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getString(_accessTokenKey);
  }

  static Future<SessionUser?> getUser() async {
    final prefs = await SharedPreferences.getInstance();
    final value = prefs.getString(_userKey);
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
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_accessTokenKey);
    await prefs.remove(_userKey);
    await prefs.remove(_cookieKey);
  }
}
