import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter_local_notifications/flutter_local_notifications.dart';
import 'package:firebase_messaging/firebase_messaging.dart';

import '../../../core/constants/api_url.dart';
import '../../../core/network/api_service.dart';
import '../../../core/services/firebase_bootstrap.dart';

typedef ForegroundNotification =
    void Function(String title, String body, String? type);

class PushNotificationService {
  PushNotificationService._();

  static final instance = PushNotificationService._();
  static const channelID = 'nfbank_transactions_v2';
  static const _channel = AndroidNotificationChannel(
    channelID,
    'Giao dịch và biến động số dư',
    description:
        'Thông báo tức thời cho chuyển khoản, thanh toán và biến động số dư.',
    importance: Importance.max,
    playSound: true,
    enableVibration: true,
    showBadge: true,
  );

  final _localNotifications = FlutterLocalNotificationsPlugin();
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
      await _initializeLocalNotifications();
      final messaging = FirebaseMessaging.instance;
      final permission = await messaging.requestPermission(
        alert: true,
        badge: true,
        sound: true,
      );
      if (permission.authorizationStatus == AuthorizationStatus.denied) return;
      await messaging.setForegroundNotificationPresentationOptions(
        alert: true,
        badge: true,
        sound: true,
      );

      _token = await messaging.getToken();
      if (_token != null) await _register(_token!);

      _refreshSubscription = messaging.onTokenRefresh.listen((token) async {
        _token = token;
        await _register(token);
      });
      _messageSubscription = FirebaseMessaging.onMessage.listen((
        message,
      ) async {
        final notification = message.notification;
        if (notification != null) {
          if (defaultTargetPlatform == TargetPlatform.android) {
            await _showSystemNotification(message);
          }
          onForegroundNotification?.call(
            notification.title ?? 'NF Bank',
            notification.body ?? '',
            message.data['type'],
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

  Future<void> _initializeLocalNotifications() async {
    const settings = InitializationSettings(
      android: AndroidInitializationSettings('ic_stat_nfbank'),
      iOS: DarwinInitializationSettings(
        requestAlertPermission: false,
        requestBadgePermission: false,
        requestSoundPermission: false,
      ),
    );
    await _localNotifications.initialize(settings: settings);
    await _localNotifications
        .resolvePlatformSpecificImplementation<
          AndroidFlutterLocalNotificationsPlugin
        >()
        ?.createNotificationChannel(_channel);
  }

  Future<void> _showSystemNotification(RemoteMessage message) async {
    final notification = message.notification;
    if (notification == null) return;
    const details = NotificationDetails(
      android: AndroidNotificationDetails(
        channelID,
        'Giao dịch và biến động số dư',
        channelDescription:
            'Thông báo tức thời cho chuyển khoản, thanh toán và biến động số dư.',
        importance: Importance.max,
        priority: Priority.high,
        category: AndroidNotificationCategory.status,
        visibility: NotificationVisibility.public,
        playSound: true,
        enableVibration: true,
        icon: 'ic_stat_nfbank',
      ),
    );
    await _localNotifications.show(
      id:
          message.messageId?.hashCode.abs() ??
          DateTime.now().millisecondsSinceEpoch.remainder(1 << 31),
      title: notification.title ?? 'NF Bank',
      body: notification.body ?? '',
      notificationDetails: details,
      payload: message.data['type'],
    );
  }
}
