import 'package:flutter/material.dart';

import '../../../core/storage/token_storage.dart';
import '../../admin/screens/admin_shell.dart';
import '../../auth/screens/login_screen.dart';
import 'app_shell.dart';

class RoleGate extends StatefulWidget {
  const RoleGate({super.key});

  @override
  State<RoleGate> createState() => _RoleGateState();
}

class _RoleGateState extends State<RoleGate> {
  late final Future<SessionUser?> _user = TokenStorage.getUser();

  @override
  Widget build(BuildContext context) => FutureBuilder<SessionUser?>(
    future: _user,
    builder: (context, snapshot) {
      if (snapshot.connectionState != ConnectionState.done) {
        return const Scaffold(body: Center(child: CircularProgressIndicator()));
      }
      final user = snapshot.data;
      if (user == null) return const LoginScreen();
      if (user.role == 'admin' || user.role == 'super_admin') {
        return AdminShell(user: user);
      }
      return const AppShell();
    },
  );
}
