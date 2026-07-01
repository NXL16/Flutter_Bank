import 'package:flutter/material.dart';

import '../../../core/services/bank_repository.dart';
import '../../../core/storage/token_storage.dart';
import '../../../shared/widgets/common.dart';
import '../../auth/screens/login_screen.dart';
import '../../auth/services/auth_service.dart';
import '../../notifications/services/push_notification_service.dart';
import '../widgets/admin_widgets.dart';
import 'admin_pages.dart';

class AdminShell extends StatefulWidget {
  const AdminShell({super.key, required this.user});

  final SessionUser user;

  @override
  State<AdminShell> createState() => _AdminShellState();
}

class _AdminShellState extends State<AdminShell> with WidgetsBindingObserver {
  static const _destinations = [
    _AdminDestination(
      'Điều hành',
      Icons.space_dashboard_outlined,
      Icons.space_dashboard_rounded,
    ),
    _AdminDestination(
      'Khách hàng',
      Icons.people_outline_rounded,
      Icons.people_rounded,
    ),
    _AdminDestination(
      'Giám sát',
      Icons.monitor_heart_outlined,
      Icons.monitor_heart_rounded,
    ),
    _AdminDestination(
      'Vận hành',
      Icons.account_balance_outlined,
      Icons.account_balance_rounded,
    ),
    _AdminDestination(
      'Thông báo',
      Icons.notifications_none_rounded,
      Icons.notifications_rounded,
    ),
  ];

  final _bankRepository = const BankRepository();
  final _monitorNavigation = ValueNotifier(AdminMonitoringTab.transactions);
  int _index = 0;
  int _unreadNotifications = 0;
  int _notificationRevision = 0;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _loadUnreadNotifications();
    PushNotificationService.instance.initialize(
      onForegroundNotification: (_, _, _) {
        if (!mounted) return;
        setState(() {
          _unreadNotifications++;
          _notificationRevision++;
        });
      },
    );
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _monitorNavigation.dispose();
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      _loadUnreadNotifications();
    }
  }

  Future<void> _loadUnreadNotifications() async {
    try {
      final items = await _bankRepository.notifications();
      if (!mounted) return;
      setState(() {
        _unreadNotifications = items
            .where((item) => item['is_read'] != true)
            .length;
      });
    } catch (_) {
      // Metadata thông báo không được chặn giao diện điều hành.
    }
  }

  void _selectPage(int value) {
    setState(() {
      _index = value;
      if (value == 4) _notificationRevision++;
    });
  }

  void _openMonitoring(AdminMonitoringTab tab) {
    setState(() => _index = 2);
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _monitorNavigation.value = tab;
    });
  }

  Future<void> _confirmLogout() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        icon: const Icon(Icons.logout_rounded, color: Color(0xFFFFB566)),
        title: const Text('Đăng xuất Admin?'),
        content: const Text(
          'Phiên quản trị trên thiết bị này sẽ kết thúc.',
          textAlign: TextAlign.center,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(dialogContext, false),
            child: const Text('Hủy'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(dialogContext, true),
            child: const Text('Đăng xuất'),
          ),
        ],
      ),
    );
    if (confirmed == true) await _logout();
  }

  Future<void> _logout() async {
    final navigator = mounted ? Navigator.of(context) : null;
    await PushNotificationService.instance.unregister();
    await AuthService().logout();
    if (navigator == null) return;
    dismissActiveNotice();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      navigator.pushAndRemoveUntil(
        MaterialPageRoute(builder: (_) => const LoginScreen()),
        (_) => false,
      );
    });
  }

  @override
  Widget build(BuildContext context) {
    final width = MediaQuery.sizeOf(context).width;
    final wide = width >= 900;
    final extended = width >= 1180;
    final pages = [
      AdminOverviewPage(
        user: widget.user,
        onTabRequested: (index) {
          if (index == 2) {
            _openMonitoring(AdminMonitoringTab.transactions);
          } else {
            _selectPage(index);
          }
        },
        onMonitoringRequested: _openMonitoring,
      ),
      const AdminUsersPage(),
      AdminMonitoringPage(navigation: _monitorNavigation),
      AdminOperationsPage(user: widget.user),
      AdminNotificationsPage(
        key: ValueKey('admin-notifications-$_notificationRevision'),
        onUnreadCountChanged: (value) {
          if (mounted && value != _unreadNotifications) {
            setState(() => _unreadNotifications = value);
          }
        },
      ),
    ];

    return Scaffold(
      backgroundColor: const Color(0xFF0B1020),
      appBar: wide
          ? null
          : AppBar(
              leadingWidth: 54,
              leading: const Padding(
                padding: EdgeInsets.only(left: 13),
                child: _AdminBrandMark(),
              ),
              titleSpacing: 8,
              title: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    _destinations[_index].label,
                    style: const TextStyle(
                      fontSize: 17,
                      fontWeight: FontWeight.w900,
                    ),
                  ),
                  Text(
                    adminRoleLabel(widget.user.role),
                    style: const TextStyle(
                      color: Color(0xFF8792AD),
                      fontSize: 10,
                    ),
                  ),
                ],
              ),
              actions: [
                IconButton(
                  tooltip: 'Thông báo',
                  onPressed: () => _selectPage(4),
                  icon: _NotificationIcon(count: _unreadNotifications),
                ),
                IconButton(
                  tooltip: 'Đăng xuất',
                  onPressed: _confirmLogout,
                  icon: const Icon(Icons.logout_rounded),
                ),
                const SizedBox(width: 4),
              ],
            ),
      body: Row(
        children: [
          if (wide)
            Container(
              decoration: const BoxDecoration(
                color: Color(0xFF10182A),
                border: Border(right: BorderSide(color: Color(0xFF25304A))),
              ),
              child: NavigationRail(
                backgroundColor: Colors.transparent,
                extended: extended,
                minWidth: 76,
                minExtendedWidth: 248,
                selectedIndex: _index,
                onDestinationSelected: _selectPage,
                leading: Padding(
                  padding: const EdgeInsets.fromLTRB(14, 22, 14, 30),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const _AdminBrandMark(),
                      if (extended) ...[
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
                              'OPERATIONS CONSOLE',
                              style: TextStyle(
                                color: Color(0xFF77829C),
                                fontSize: 9,
                                letterSpacing: 1.1,
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
                      padding: const EdgeInsets.fromLTRB(10, 0, 10, 18),
                      child: extended
                          ? _AdminIdentityCard(
                              user: widget.user,
                              onLogout: _confirmLogout,
                            )
                          : IconButton(
                              tooltip: 'Đăng xuất',
                              onPressed: _confirmLogout,
                              icon: const Icon(Icons.logout_rounded),
                            ),
                    ),
                  ),
                ),
                destinations: [
                  for (var index = 0; index < _destinations.length; index++)
                    NavigationRailDestination(
                      icon: index == 4
                          ? _NotificationIcon(count: _unreadNotifications)
                          : Icon(_destinations[index].icon),
                      selectedIcon: index == 4
                          ? _NotificationIcon(count: _unreadNotifications)
                          : Icon(_destinations[index].selectedIcon),
                      label: Text(_destinations[index].label),
                    ),
                ],
              ),
            ),
          Expanded(
            child: SafeArea(
              child: Padding(
                padding: EdgeInsets.all(wide ? 24 : 14),
                child: IndexedStack(index: _index, children: pages),
              ),
            ),
          ),
        ],
      ),
      bottomNavigationBar: wide
          ? null
          : NavigationBar(
              height: 68,
              selectedIndex: _index,
              onDestinationSelected: _selectPage,
              destinations: [
                for (var index = 0; index < _destinations.length; index++)
                  NavigationDestination(
                    icon: index == 4
                        ? _NotificationIcon(count: _unreadNotifications)
                        : Icon(_destinations[index].icon),
                    selectedIcon: index == 4
                        ? _NotificationIcon(count: _unreadNotifications)
                        : Icon(_destinations[index].selectedIcon),
                    label: _destinations[index].label,
                  ),
              ],
            ),
    );
  }
}

class _AdminBrandMark extends StatelessWidget {
  const _AdminBrandMark();

  @override
  Widget build(BuildContext context) => Container(
    width: 38,
    height: 38,
    decoration: BoxDecoration(
      gradient: const LinearGradient(
        colors: [Color(0xFF7D86FF), Color(0xFF4E5AC7)],
        begin: Alignment.topLeft,
        end: Alignment.bottomRight,
      ),
      borderRadius: BorderRadius.circular(12),
      boxShadow: const [
        BoxShadow(
          color: Color(0x447D86FF),
          blurRadius: 14,
          offset: Offset(0, 5),
        ),
      ],
    ),
    child: const Icon(
      Icons.admin_panel_settings_rounded,
      color: Colors.white,
      size: 21,
    ),
  );
}

class _NotificationIcon extends StatelessWidget {
  const _NotificationIcon({required this.count});

  final int count;

  @override
  Widget build(BuildContext context) => Stack(
    clipBehavior: Clip.none,
    children: [
      const Icon(Icons.notifications_none_rounded),
      if (count > 0)
        Positioned(
          right: -7,
          top: -6,
          child: Container(
            constraints: const BoxConstraints(minWidth: 17, minHeight: 17),
            padding: const EdgeInsets.symmetric(horizontal: 4),
            decoration: BoxDecoration(
              color: const Color(0xFFFF6577),
              borderRadius: BorderRadius.circular(10),
              border: Border.all(color: const Color(0xFF10182A), width: 1.5),
            ),
            alignment: Alignment.center,
            child: Text(
              count > 99 ? '99+' : '$count',
              style: const TextStyle(
                color: Colors.white,
                fontSize: 9,
                height: 1,
                fontWeight: FontWeight.w900,
              ),
            ),
          ),
        ),
    ],
  );
}

class _AdminIdentityCard extends StatelessWidget {
  const _AdminIdentityCard({required this.user, required this.onLogout});

  final SessionUser user;
  final VoidCallback onLogout;

  @override
  Widget build(BuildContext context) {
    final initial = user.fullName.trim().isEmpty
        ? 'A'
        : user.fullName.trim().characters.first.toUpperCase();
    return Container(
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: const Color(0xFF151F34),
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: const Color(0xFF2B3855)),
      ),
      child: Row(
        children: [
          CircleAvatar(
            radius: 17,
            backgroundColor: const Color(0xFF344170),
            child: Text(
              initial,
              style: const TextStyle(fontWeight: FontWeight.w900),
            ),
          ),
          const SizedBox(width: 9),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  user.fullName,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    fontSize: 11,
                    fontWeight: FontWeight.w800,
                  ),
                ),
                Text(
                  adminRoleLabel(user.role),
                  style: const TextStyle(color: Color(0xFF7F8BA6), fontSize: 9),
                ),
              ],
            ),
          ),
          IconButton(
            tooltip: 'Đăng xuất',
            visualDensity: VisualDensity.compact,
            onPressed: onLogout,
            icon: const Icon(Icons.logout_rounded, size: 18),
          ),
        ],
      ),
    );
  }
}

class _AdminDestination {
  const _AdminDestination(this.label, this.icon, this.selectedIcon);

  final String label;
  final IconData icon;
  final IconData selectedIcon;
}
