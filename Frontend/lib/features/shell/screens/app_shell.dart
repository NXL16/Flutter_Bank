import 'package:flutter/material.dart';

import '../../../core/services/bank_repository.dart';
import '../../../core/storage/token_storage.dart';
import '../../auth/screens/login_screen.dart';
import '../../auth/services/auth_service.dart';
import '../../banking/screens/user_pages.dart';
import '../../notifications/services/push_notification_service.dart';
import '../../transfers/screens/transfer_page.dart';

class AppShell extends StatefulWidget {
  const AppShell({super.key});

  @override
  State<AppShell> createState() => _AppShellState();
}

class _AppShellState extends State<AppShell> with WidgetsBindingObserver {
  static const _savingsIndex = 4;
  static const _notificationIndex = 5;
  static const _profileIndex = 6;

  final _bankRepository = const BankRepository();
  SessionUser? _user;
  String _avatarUrl = '';
  int _unreadNotifications = 0;
  int _notificationRevision = 0;
  int _index = 0;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _loadUser();
    _loadShellMetadata();
    PushNotificationService.instance.initialize(
      onForegroundNotification: (_, _, _) {
        if (mounted) {
          setState(() {
            _unreadNotifications++;
            _notificationRevision++;
          });
        }
      },
    );
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      _loadShellMetadata();
    }
  }

  Future<void> _loadUser() async {
    final user = await TokenStorage.getUser();
    if (mounted) setState(() => _user = user);
  }

  Future<void> _loadShellMetadata() async {
    try {
      final results = await Future.wait([
        _bankRepository.profile(),
        _bankRepository.notifications(),
      ]);
      final profile = results[0] as Map<String, dynamic>;
      final notifications = results[1] as List<Map<String, dynamic>>;
      if (!mounted) return;
      setState(() {
        _avatarUrl = profile['avatar_url']?.toString().trim() ?? '';
        _unreadNotifications = notifications
            .where((item) => item['is_read'] != true)
            .length;
      });
    } catch (_) {
      // Metadata không được làm gián đoạn việc mở ứng dụng.
    }
  }

  void _setUnreadNotifications(int value) {
    if (mounted && value != _unreadNotifications) {
      setState(() => _unreadNotifications = value);
    }
  }

  void _openNotifications() {
    setState(() {
      _index = _notificationIndex;
      _notificationRevision++;
    });
  }

  void _openSavings() => setState(() => _index = _savingsIndex);

  void _openProfile() => setState(() => _index = _profileIndex);

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
      _NavItem(
        'Tổng quan',
        Icons.grid_view_rounded,
        DashboardPage(onOpenSavings: _openSavings),
      ),
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
      _NavItem(
        'Thông báo',
        Icons.notifications_none_rounded,
        NotificationsPage(
          key: ValueKey('notifications-$_notificationRevision'),
          onUnreadCountChanged: _setUnreadNotifications,
        ),
      ),
      _NavItem(
        'Hồ sơ',
        Icons.person_outline_rounded,
        ProfilePage(
          onLogout: _logout,
          onAvatarChanged: (url) => setState(() => _avatarUrl = url),
        ),
      ),
    ];
    if (_index >= items.length) _index = 0;

    final wide = MediaQuery.sizeOf(context).width >= 900;
    return Scaffold(
      appBar: wide
          ? null
          : AppBar(
              leadingWidth: 57,
              titleSpacing: 4,
              leading: IconButton(
                tooltip: 'Mở hồ sơ',
                onPressed: _openProfile,
                icon: Container(
                  padding: const EdgeInsets.all(1.5),
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    border: Border.all(
                      color: const Color(0xFF8F95FF),
                      width: 1.5,
                    ),
                  ),
                  child: CircleAvatar(
                    radius: 15.5,
                    backgroundImage: _avatarUrl.isEmpty
                        ? null
                        : NetworkImage(_avatarUrl),
                    child: _avatarUrl.isEmpty
                        ? Text(
                            user.fullName.trim().isEmpty
                                ? 'N'
                                : user.fullName.trim()[0].toUpperCase(),
                            style: const TextStyle(
                              fontSize: 13,
                              fontWeight: FontWeight.w900,
                            ),
                          )
                        : null,
                  ),
                ),
              ),
              title: Text(items[_index].label),
              actions: [
                _NotificationButton(
                  unreadCount: _unreadNotifications,
                  onPressed: _openNotifications,
                ),
                const SizedBox(width: 6),
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
          : _MobileBottomBar(
              items: items.take(5).toList(growable: false),
              selectedIndex: _index < 5 ? _index : null,
              onSelected: (value) => setState(() => _index = value),
            ),
    );
  }
}

class _NotificationButton extends StatelessWidget {
  const _NotificationButton({
    required this.unreadCount,
    required this.onPressed,
  });

  final int unreadCount;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) => IconButton(
    tooltip: 'Thông báo',
    onPressed: onPressed,
    icon: Stack(
      clipBehavior: Clip.none,
      children: [
        const Icon(Icons.notifications_none_rounded),
        if (unreadCount > 0)
          Positioned(
            right: -7,
            top: -6,
            child: Container(
              constraints: const BoxConstraints(minWidth: 17, minHeight: 17),
              padding: const EdgeInsets.symmetric(horizontal: 4),
              decoration: BoxDecoration(
                color: const Color(0xFFFF5C72),
                borderRadius: BorderRadius.circular(10),
                border: Border.all(
                  color: Theme.of(context).colorScheme.surface,
                  width: 1.5,
                ),
              ),
              alignment: Alignment.center,
              child: Text(
                unreadCount > 99 ? '99+' : '$unreadCount',
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
    ),
  );
}

class _MobileBottomBar extends StatelessWidget {
  const _MobileBottomBar({
    required this.items,
    required this.selectedIndex,
    required this.onSelected,
  });

  final List<_NavItem> items;
  final int? selectedIndex;
  final ValueChanged<int> onSelected;

  @override
  Widget build(BuildContext context) => Material(
    color: Theme.of(context).colorScheme.surface,
    elevation: 8,
    child: SafeArea(
      top: false,
      child: SizedBox(
        height: 66,
        child: Row(
          children: [
            for (var index = 0; index < items.length; index++)
              Expanded(
                child: _BottomBarItem(
                  item: items[index],
                  selected: selectedIndex == index,
                  onTap: () => onSelected(index),
                ),
              ),
          ],
        ),
      ),
    ),
  );
}

class _BottomBarItem extends StatelessWidget {
  const _BottomBarItem({
    required this.item,
    required this.selected,
    required this.onTap,
  });

  final _NavItem item;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final color = selected ? const Color(0xFF9EA4FF) : const Color(0xFF7F8BA5);
    return InkResponse(
      onTap: onTap,
      radius: 30,
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          AnimatedContainer(
            duration: const Duration(milliseconds: 180),
            width: 38,
            height: 28,
            decoration: BoxDecoration(
              color: selected
                  ? const Color(0xFF7C83FD).withValues(alpha: .15)
                  : Colors.transparent,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Icon(item.icon, size: 21, color: color),
          ),
          const SizedBox(height: 3),
          Text(
            item.label,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(
              color: color,
              fontSize: 10,
              fontWeight: selected ? FontWeight.w800 : FontWeight.w500,
            ),
          ),
        ],
      ),
    );
  }
}

class _NavItem {
  const _NavItem(this.label, this.icon, this.page);

  final String label;
  final IconData icon;
  final Widget page;
}
