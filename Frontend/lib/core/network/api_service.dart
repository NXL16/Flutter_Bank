import 'dart:convert';

import 'package:http/http.dart' as http;

import '../constants/api_url.dart';
import '../storage/token_storage.dart';
import 'client_factory.dart';

class ApiException implements Exception {
  const ApiException(this.message, {this.statusCode});

  final String message;
  final int? statusCode;

  @override
  String toString() => message;
}

class ApiResult {
  const ApiResult({
    required this.statusCode,
    required this.message,
    this.data,
    this.raw,
  });

  final int statusCode;
  final String message;
  final dynamic data;
  final dynamic raw;

  bool get isSuccess => statusCode >= 200 && statusCode < 300;
}

class ApiService {
  ApiService._();

  static final http.Client _client = createHttpClient();
  static Future<bool>? _refreshInFlight;
  static Future<void>? _sessionExpirationInFlight;
  static Future<void> Function()? onSessionExpired;

  static Future<ApiResult> get(String url, {bool auth = false}) =>
      _request('GET', url, auth: auth);

  static Future<ApiResult> post(
    String url, {
    Map<String, dynamic>? body,
    bool auth = false,
    Map<String, String>? headers,
  }) => _request('POST', url, body: body, auth: auth, extraHeaders: headers);

  static Future<ApiResult> put(
    String url, {
    Map<String, dynamic>? body,
    bool auth = true,
  }) => _request('PUT', url, body: body, auth: auth);

  static Future<ApiResult> patch(
    String url, {
    Map<String, dynamic>? body,
    bool auth = true,
    Map<String, String>? headers,
  }) => _request('PATCH', url, body: body, auth: auth, extraHeaders: headers);

  static Future<ApiResult> delete(
    String url, {
    Map<String, dynamic>? body,
    bool auth = true,
  }) => _request('DELETE', url, body: body, auth: auth);

  static Future<ApiResult> uploadFile(
    String url, {
    required String fieldName,
    required List<int> bytes,
    required String filename,
    bool auth = true,
    bool retry = true,
  }) async {
    final request = http.MultipartRequest('POST', Uri.parse(url));
    if (auth) {
      final token = await TokenStorage.getToken();
      if (token != null) request.headers['Authorization'] = 'Bearer $token';
    }
    request.files.add(
      http.MultipartFile.fromBytes(fieldName, bytes, filename: filename),
    );

    http.Response response;
    try {
      response = await http.Response.fromStream(await _client.send(request));
    } catch (_) {
      throw ApiException('Không thể kết nối ${ApiUrl.baseUrl}.');
    }
    if (response.statusCode == 401 && auth && retry && await _tryRefresh()) {
      return uploadFile(
        url,
        fieldName: fieldName,
        bytes: bytes,
        filename: filename,
        auth: auth,
        retry: false,
      );
    }
    if (response.statusCode == 401 && auth) {
      await _expireSession();
    }

    dynamic decoded;
    try {
      decoded = response.body.isEmpty ? null : jsonDecode(response.body);
    } catch (_) {
      decoded = response.body;
    }
    final map = decoded is Map<String, dynamic> ? decoded : null;
    final message =
        map?['message']?.toString() ??
        (response.statusCode >= 400
            ? 'Tải tệp không thành công'
            : 'Thành công');
    final result = ApiResult(
      statusCode: response.statusCode,
      message: message,
      data: map != null && map.containsKey('data') ? map['data'] : decoded,
      raw: decoded,
    );
    if (!result.isSuccess) {
      throw ApiException(message, statusCode: response.statusCode);
    }
    return result;
  }

  static Future<ApiResult> _request(
    String method,
    String url, {
    Map<String, dynamic>? body,
    required bool auth,
    bool retry = true,
    Map<String, String>? extraHeaders,
  }) async {
    final headers = <String, String>{'Content-Type': 'application/json'};
    if (extraHeaders != null) headers.addAll(extraHeaders);
    if (auth) {
      final token = await TokenStorage.getToken();
      if (token != null) headers['Authorization'] = 'Bearer $token';
    }

    final request = http.Request(method, Uri.parse(url))
      ..headers.addAll(headers);
    if (body != null) request.body = jsonEncode(body);

    http.Response response;
    try {
      response = await http.Response.fromStream(await _client.send(request));
    } catch (_) {
      throw ApiException('Máy chủ chưa sẵn sàng. Vui lòng thử lại');
    }

    if (response.statusCode == 401 && auth && retry) {
      final refreshed = await _tryRefresh();
      if (refreshed) {
        return _request(
          method,
          url,
          body: body,
          auth: auth,
          retry: false,
          extraHeaders: extraHeaders,
        );
      }
    }
    if (response.statusCode == 401 && auth) {
      await _expireSession();
    }

    dynamic decoded;
    try {
      decoded = response.body.isEmpty ? null : jsonDecode(response.body);
    } catch (_) {
      decoded = response.body;
    }

    final map = decoded is Map<String, dynamic> ? decoded : null;
    final message =
        map?['message']?.toString() ??
        (response.statusCode >= 400
            ? 'Yêu cầu không thành công'
            : 'Thành công');
    final result = ApiResult(
      statusCode: response.statusCode,
      message: message,
      data: map != null && map.containsKey('data') ? map['data'] : decoded,
      raw: decoded,
    );
    if (!result.isSuccess) {
      throw ApiException(message, statusCode: response.statusCode);
    }
    return result;
  }

  static Future<bool> _tryRefresh() {
    final inFlight = _refreshInFlight;
    if (inFlight != null) return inFlight;
    final refresh = _performRefresh();
    _refreshInFlight = refresh;
    return refresh.whenComplete(() {
      if (identical(_refreshInFlight, refresh)) _refreshInFlight = null;
    });
  }

  static Future<bool> _performRefresh() async {
    try {
      final request = http.Request('POST', Uri.parse(ApiUrl.refresh))
        ..headers['Content-Type'] = 'application/json';
      final response = await http.Response.fromStream(
        await _client.send(request),
      );
      if (response.statusCode != 200) return false;
      final json = jsonDecode(response.body) as Map<String, dynamic>;
      final data = json['data'] as Map<String, dynamic>?;
      final token = data?['access_token']?.toString();
      final oldUser = await TokenStorage.getUser();
      if (token == null || oldUser == null) return false;
      await TokenStorage.saveAuthData(token, oldUser.toJson());
      return true;
    } catch (_) {
      return false;
    }
  }

  static Future<void> _expireSession() {
    final inFlight = _sessionExpirationInFlight;
    if (inFlight != null) return inFlight;
    final expiration = () async {
      await TokenStorage.clearAuth();
      await onSessionExpired?.call();
    }();
    _sessionExpirationInFlight = expiration;
    return expiration.whenComplete(() {
      if (identical(_sessionExpirationInFlight, expiration)) {
        _sessionExpirationInFlight = null;
      }
    });
  }
}
