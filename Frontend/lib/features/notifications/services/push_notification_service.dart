import 'dart:async';

import 'package:firebase_messaging/firebase_messaging.dart';

import '../../../core/constants/api_url.dart';
import '../../../core/network/api_service.dart';
import '../../../core/services/firebase_bootstrap.dart';

typedef ForegroundNotification = void Function(String title, String body);

class PushNotificationService {
  PushNotificationService._();

  static final instance = PushNotificationService._();

  StreamSubscription<String>? _refreshSubscription;
  StreamSubscription<RemoteMessage>? _messageSubscription;
  String? _token;
  bool _initialized = false;

  Future<void> initialize({
    ForegroundNotification? onForegroundNotification,
  }) async {
    if (_initialized || !FirebaseBootstrap.enabled) return;

    try {
      await FirebaseBootstrap.initialize();
      final messaging = FirebaseMessaging.instance;
      final permission = await messaging.requestPermission(
        alert: true,
        badge: true,
        sound: true,
      );
      if (permission.authorizationStatus == AuthorizationStatus.denied) return;

      _token = await messaging.getToken();
      if (_token != null) await _register(_token!);

      _refreshSubscription = messaging.onTokenRefresh.listen((token) async {
        _token = token;
        await _register(token);
      });
      _messageSubscription = FirebaseMessaging.onMessage.listen((message) {
        final notification = message.notification;
        if (notification != null) {
          onForegroundNotification?.call(
            notification.title ?? 'NF Bank',
            notification.body ?? '',
          );
        }
      });
      _initialized = true;
    } catch (_) {
      // Firebase chưa được cấu hình trên flavor hiện tại. App vẫn hoạt động
      // với thông báo trong hộp thư, nhưng không đăng ký push.
    }
  }

  Future<void> unregister() async {
    final token = _token;
    if (token != null) {
      try {
        await ApiService.delete(
          '${ApiUrl.notifications}/devices',
          auth: true,
          body: {'token': token, 'platform': 'android'},
        );
      } catch (_) {
        // Logout không được mắc kẹt chỉ vì endpoint push tạm thời lỗi.
      }
    }
    await _refreshSubscription?.cancel();
    await _messageSubscription?.cancel();
    _initialized = false;
    _token = null;
  }

  Future<void> _register(String token) => ApiService.post(
    '${ApiUrl.notifications}/devices',
    auth: true,
    body: {'token': token, 'platform': 'android'},
  );
}
