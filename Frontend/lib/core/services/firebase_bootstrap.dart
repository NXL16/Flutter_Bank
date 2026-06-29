import 'package:firebase_core/firebase_core.dart';
import 'package:flutter_dotenv/flutter_dotenv.dart';

class FirebaseBootstrap {
  FirebaseBootstrap._();

  static bool get enabled =>
      dotenv.env['FIREBASE_ENABLED']?.toLowerCase() == 'true';

  static Future<void> initialize() async {
    if (Firebase.apps.isNotEmpty) return;

    final apiKey = _required('FIREBASE_API_KEY');
    final appId = _required('FIREBASE_APP_ID');
    final senderId = _required('FIREBASE_MESSAGING_SENDER_ID');
    final projectId = _required('FIREBASE_PROJECT_ID');

    await Firebase.initializeApp(
      options: FirebaseOptions(
        apiKey: apiKey,
        appId: appId,
        messagingSenderId: senderId,
        projectId: projectId,
        authDomain: _optional('FIREBASE_AUTH_DOMAIN'),
        storageBucket: _optional('FIREBASE_STORAGE_BUCKET'),
      ),
    );
  }

  static String _required(String key) {
    final value = dotenv.env[key]?.trim();
    if (value == null || value.isEmpty) {
      throw StateError('Thiếu $key trong file .env');
    }
    return value;
  }

  static String? _optional(String key) {
    final value = dotenv.env[key]?.trim();
    return value == null || value.isEmpty ? null : value;
  }
}
