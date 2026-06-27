import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

http.Client createHttpClient() => _CookieClient();

class _CookieClient extends http.BaseClient {
  static const _storageKey = 'HTTP_AUTH_COOKIES';
  final http.Client _inner = http.Client();
  final Map<String, String> _cookies = {};
  bool _loaded = false;

  Future<void> _loadCookies() async {
    if (_loaded) return;
    _loaded = true;
    final prefs = await SharedPreferences.getInstance();
    final value = prefs.getString(_storageKey);
    if (value == null) return;
    try {
      final decoded = jsonDecode(value);
      if (decoded is Map<String, dynamic>) {
        _cookies.addAll(
          decoded.map((key, value) => MapEntry(key, value.toString())),
        );
      }
    } catch (_) {
      await prefs.remove(_storageKey);
    }
  }

  Future<void> _persistCookies() async {
    final prefs = await SharedPreferences.getInstance();
    if (_cookies.isEmpty) {
      await prefs.remove(_storageKey);
    } else {
      await prefs.setString(_storageKey, jsonEncode(_cookies));
    }
  }

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    await _loadCookies();
    if (_cookies.isNotEmpty) {
      request.headers['Cookie'] = _cookies.entries
          .map((item) => '${item.key}=${item.value}')
          .join('; ');
    }
    final response = await _inner.send(request);
    final setCookie = response.headers['set-cookie'];
    if (setCookie != null) {
      final matches = RegExp(
        r'(refresh_token|device_id)=([^;,\s]*)',
      ).allMatches(setCookie);
      for (final match in matches) {
        final key = match.group(1)!;
        final value = match.group(2) ?? '';
        if (value.isEmpty) {
          _cookies.remove(key);
        } else {
          _cookies[key] = value;
        }
      }
      await _persistCookies();
    }
    return response;
  }

  @override
  void close() => _inner.close();
}
