import 'package:flutter/material.dart';

import '../../../core/storage/token_storage.dart';
import '../../auth/screens/login_screen.dart';
import '../../auth/services/auth_service.dart';
import '../../banking/screens/user_pages.dart' hide TransferPage;
import '../../notifications/services/push_notification_service.dart';
import '../../transfers/screens/transfer_page.dart';
import '../../../shared/widgets/common.dart';

class AppShell extends StatefulWidget {
  const AppShell({super.key});

  @override
  State<AppShell> createState() => _AppShellState();
}

class _AppShellState extends State<AppShell> {
  SessionUser? _user;
  int _index = 0;

  @override
  void initState() {
    super.initState();
    _loadUser();
    PushNotificationService.instance.initialize(
      onForegroundNotification: (title, body) {
        if (mounted) showMessage(context, '$title\n$body');
      },
    );
  }

  Future<void> _loadUser() async {
    final user = await TokenStorage.getUser();
    if (mounted) setState(() => _user = user);
  }

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
    final user = _user;
    if (user == null) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }
    final items = <_NavItem>[
      const _NavItem('Tổng quan', Icons.grid_view_rounded, DashboardPage()),
      const _NavItem(
        'Tài khoản',
        Icons.account_balance_wallet_outlined,
        AccountsPage(),
      ),
      const _NavItem('Chuyển tiền', Icons.swap_horiz_rounded, TransferPage()),
      const _NavItem(
        'Giao dịch',
        Icons.receipt_long_outlined,
        TransactionsPage(),
      ),
      const _NavItem('Tiết kiệm', Icons.savings_outlined, SavingsPage()),
      const _NavItem(
        'Thông báo',
        Icons.notifications_none_rounded,
        NotificationsPage(),
      ),
      const _NavItem('Hồ sơ', Icons.person_outline_rounded, ProfilePage()),
    ];
    if (_index >= items.length) _index = 0;

    final wide = MediaQuery.sizeOf(context).width >= 900;
    return Scaffold(
      appBar: wide
          ? null
          : AppBar(
              title: Text(items[_index].label),
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
              extended: MediaQuery.sizeOf(context).width >= 1180,
              minExtendedWidth: 235,
              selectedIndex: _index,
              onDestinationSelected: (value) => setState(() => _index = value),
              leading: Padding(
                padding: const EdgeInsets.fromLTRB(12, 20, 12, 28),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const CircleAvatar(
                      child: Icon(Icons.account_balance_rounded),
                    ),
                    if (MediaQuery.sizeOf(context).width >= 1180) ...[
                      const SizedBox(width: 12),
                      const Text(
                        'NF BANK',
                        style: TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.w800,
                        ),
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
              destinations: items
                  .map(
                    (item) => NavigationRailDestination(
                      icon: Icon(item.icon),
                      label: Text(item.label),
                    ),
                  )
                  .toList(),
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
                    child: items[_index].page,
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
              selectedIndex: _index.clamp(0, 4),
              onDestinationSelected: (value) => setState(() => _index = value),
              destinations: items
                  .take(5)
                  .map(
                    (item) => NavigationDestination(
                      icon: Icon(item.icon),
                      label: item.label,
                    ),
                  )
                  .toList(),
            ),
      drawer: !wide && items.length > 5
          ? Drawer(
              child: ListView(
                padding: const EdgeInsets.only(top: 60),
                children: [
                  for (var i = 5; i < items.length; i++)
                    ListTile(
                      leading: Icon(items[i].icon),
                      title: Text(items[i].label),
                      selected: _index == i,
                      onTap: () {
                        Navigator.pop(context);
                        setState(() => _index = i);
                      },
                    ),
                ],
              ),
            )
          : null,
    );
  }
}

class _NavItem {
  const _NavItem(this.label, this.icon, this.page);

  final String label;
  final IconData icon;
  final Widget page;
}
