import 'package:flutter/material.dart';

import '../../../core/storage/token_storage.dart';
import '../../auth/screens/login_screen.dart';
import '../../auth/services/auth_service.dart';
import '../../notifications/services/push_notification_service.dart';
import 'admin_pages.dart';

class AdminShell extends StatefulWidget {
  const AdminShell({super.key, required this.user});

  final SessionUser user;

  @override
  State<AdminShell> createState() => _AdminShellState();
}

class _AdminShellState extends State<AdminShell> {
  int _index = 0;

  Future<void> _logout() async {
    await PushNotificationService.instance.unregister();
    await AuthService().logout();
    if (!mounted) return;
    Navigator.of(context).pushAndRemoveUntil(
      MaterialPageRoute(builder: (_) => const LoginScreen()),
      (_) => false,
    );
  }

  @override
  Widget build(BuildContext context) {
    final pages = [
      AdminOverviewPage(user: widget.user),
      const AdminUsersPage(),
      AdminOperationsPage(user: widget.user),
    ];
    const destinations = [
      NavigationDestination(
        icon: Icon(Icons.dashboard_outlined),
        selectedIcon: Icon(Icons.dashboard_rounded),
        label: 'Tổng quan',
      ),
      NavigationDestination(
        icon: Icon(Icons.people_outline_rounded),
        selectedIcon: Icon(Icons.people_rounded),
        label: 'Người dùng',
      ),
      NavigationDestination(
        icon: Icon(Icons.account_balance_outlined),
        selectedIcon: Icon(Icons.account_balance_rounded),
        label: 'Nghiệp vụ',
      ),
    ];
    final wide = MediaQuery.sizeOf(context).width >= 900;

    return Scaffold(
      appBar: wide
          ? null
          : AppBar(
              title: const Text('NF Bank Admin'),
              actions: [
                IconButton(
                  tooltip: 'Đăng xuất',
                  onPressed: _logout,
                  icon: const Icon(Icons.logout_rounded),
                ),
              ],
            ),
      body: Row(
        children: [
          if (wide)
            NavigationRail(
              extended: MediaQuery.sizeOf(context).width >= 1160,
              minExtendedWidth: 245,
              selectedIndex: _index,
              onDestinationSelected: (value) => setState(() => _index = value),
              leading: Padding(
                padding: const EdgeInsets.fromLTRB(12, 22, 12, 30),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const CircleAvatar(
                      backgroundColor: Color(0xFF6C7CFF),
                      child: Icon(Icons.admin_panel_settings_rounded),
                    ),
                    if (MediaQuery.sizeOf(context).width >= 1160) ...[
                      const SizedBox(width: 12),
                      const Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            'NF BANK',
                            style: TextStyle(
                              fontWeight: FontWeight.w900,
                              letterSpacing: 1,
                            ),
                          ),
                          Text(
                            'ADMIN CONSOLE',
                            style: TextStyle(
                              color: Colors.white54,
                              fontSize: 10,
                              letterSpacing: 1.2,
                            ),
                          ),
                        ],
                      ),
                    ],
                  ],
                ),
              ),
              trailing: Expanded(
                child: Align(
                  alignment: Alignment.bottomCenter,
                  child: Padding(
                    padding: const EdgeInsets.only(bottom: 18),
                    child: IconButton(
                      tooltip: 'Đăng xuất',
                      onPressed: _logout,
                      icon: const Icon(Icons.logout_rounded),
                    ),
                  ),
                ),
              ),
              destinations: const [
                NavigationRailDestination(
                  icon: Icon(Icons.dashboard_outlined),
                  selectedIcon: Icon(Icons.dashboard_rounded),
                  label: Text('Tổng quan'),
                ),
                NavigationRailDestination(
                  icon: Icon(Icons.people_outline_rounded),
                  selectedIcon: Icon(Icons.people_rounded),
                  label: Text('Người dùng'),
                ),
                NavigationRailDestination(
                  icon: Icon(Icons.account_balance_outlined),
                  selectedIcon: Icon(Icons.account_balance_rounded),
                  label: Text('Nghiệp vụ'),
                ),
              ],
            ),
          Expanded(
            child: Container(
              decoration: wide
                  ? BoxDecoration(
                      border: Border(
                        left: BorderSide(
                          color: Colors.white.withValues(alpha: .06),
                        ),
                      ),
                    )
                  : null,
              child: SafeArea(
                child: Padding(
                  padding: EdgeInsets.all(wide ? 28 : 16),
                  child: KeyedSubtree(
                    key: ValueKey(_index),
                    child: pages[_index],
                  ),
                ),
              ),
            ),
          ),
        ],
      ),
      bottomNavigationBar: wide
          ? null
          : NavigationBar(
              selectedIndex: _index,
              onDestinationSelected: (value) => setState(() => _index = value),
              destinations: destinations,
            ),
    );
  }
}
