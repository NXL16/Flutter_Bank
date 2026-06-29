import 'dart:async';

import 'package:firebase_auth/firebase_auth.dart';

import '../../../core/network/api_service.dart';
import '../../../core/services/firebase_bootstrap.dart';

class PhoneAuthService {
  Future<String> sendCode(String phone) async {
    await _ensureFirebase();
    final completer = Completer<String>();

    await FirebaseAuth.instance.verifyPhoneNumber(
      phoneNumber: normalizeVietnamPhone(phone),
      timeout: const Duration(seconds: 60),
      verificationCompleted: (credential) async {
        try {
          final result = await FirebaseAuth.instance.signInWithCredential(
            credential,
          );
          final token = await result.user?.getIdToken(true);
          if (token != null && !completer.isCompleted) {
            completer.complete('AUTO:$token');
          }
        } catch (error) {
          if (!completer.isCompleted) completer.completeError(_mapError(error));
        }
      },
      verificationFailed: (error) {
        if (!completer.isCompleted) completer.completeError(_mapError(error));
      },
      codeSent: (verificationId, _) {
        if (!completer.isCompleted) completer.complete(verificationId);
      },
      codeAutoRetrievalTimeout: (verificationId) {
        if (!completer.isCompleted) completer.complete(verificationId);
      },
    );

    return completer.future;
  }

  Future<String> verifyCode(String verificationId, String smsCode) async {
    if (verificationId.startsWith('AUTO:')) {
      final token = verificationId.substring(5);
      await FirebaseAuth.instance.signOut();
      return token;
    }
    if (smsCode.trim().length != 6) {
      throw const ApiException('Mã OTP phải gồm 6 chữ số');
    }

    try {
      final credential = PhoneAuthProvider.credential(
        verificationId: verificationId,
        smsCode: smsCode.trim(),
      );
      final result = await FirebaseAuth.instance.signInWithCredential(
        credential,
      );
      final token = await result.user?.getIdToken(true);
      if (token == null || token.isEmpty) {
        throw const ApiException('Không lấy được mã xác thực Firebase');
      }
      await FirebaseAuth.instance.signOut();
      return token;
    } catch (error) {
      throw _mapError(error);
    }
  }

  Future<void> _ensureFirebase() async {
    if (!FirebaseBootstrap.enabled) {
      throw const ApiException(
        'Firebase chưa được bật. Hãy điền cấu hình Firebase trong .env và '
        'đặt FIREBASE_ENABLED=true.',
      );
    }
    try {
      await FirebaseBootstrap.initialize();
    } catch (_) {
      throw const ApiException(
        'Không khởi tạo được Firebase. Kiểm tra các biến FIREBASE_* và '
        'Android app com.nfbank.mobile.',
      );
    }
  }

  ApiException _mapError(Object error) {
    if (error is ApiException) return error;
    if (error is FirebaseAuthException) {
      const messages = {
        'invalid-phone-number': 'Số điện thoại không hợp lệ',
        'invalid-verification-code': 'Mã OTP không đúng',
        'session-expired': 'Mã OTP đã hết hạn, vui lòng gửi lại',
        'too-many-requests': 'Bạn thử quá nhiều lần, vui lòng chờ một lúc',
        'quota-exceeded': 'Firebase đã hết hạn mức gửi SMS',
      };
      return ApiException(
        messages[error.code] ?? error.message ?? 'Xác thực SMS thất bại',
      );
    }
    return const ApiException('Xác thực SMS thất bại');
  }

  static String normalizeVietnamPhone(String value) {
    final digits = value.replaceAll(RegExp(r'\D'), '');
    if (digits.startsWith('84')) return '+$digits';
    if (digits.startsWith('0')) return '+84${digits.substring(1)}';
    return '+84$digits';
  }
}
