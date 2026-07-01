import 'package:flutter/material.dart';
import 'package:flutter_dotenv/flutter_dotenv.dart';

import 'core/network/api_service.dart';
import 'core/storage/token_storage.dart';
import 'core/theme/app_theme.dart';
import 'features/auth/screens/login_screen.dart';
import 'features/shell/screens/role_gate.dart';

final appNavigatorKey = GlobalKey<NavigatorState>();

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await dotenv.load(fileName: '.env');
  final token = await TokenStorage.getToken();
  final user = await TokenStorage.getUser();
  ApiService.onSessionExpired = () async {
    final navigator = appNavigatorKey.currentState;
    if (navigator == null) return;
    navigator.pushAndRemoveUntil(
      MaterialPageRoute(builder: (_) => const LoginScreen()),
      (_) => false,
    );
  };
  runApp(NfBankApp(isLoggedIn: token != null && user != null));
}

class NfBankApp extends StatelessWidget {
  const NfBankApp({super.key, required this.isLoggedIn});

  final bool isLoggedIn;

  @override
  Widget build(BuildContext context) => MaterialApp(
    navigatorKey: appNavigatorKey,
    title: 'NF Bank',
    debugShowCheckedModeBanner: false,
    theme: AppTheme.dark,
    home: isLoggedIn ? const RoleGate() : const LoginScreen(),
  );
}
