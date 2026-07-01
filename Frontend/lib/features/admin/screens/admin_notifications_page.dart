import 'package:flutter/material.dart';

import '../../../core/network/api_service.dart';
import '../../../core/services/bank_repository.dart';
import '../../../shared/utils/formatters.dart';
import '../../../shared/widgets/common.dart';
import '../widgets/admin_widgets.dart';

class AdminNotificationsPage extends StatefulWidget {
  const AdminNotificationsPage({super.key, this.onUnreadCountChanged});

  final ValueChanged<int>? onUnreadCountChanged;

  @override
  State<AdminNotificationsPage> createState() => _AdminNotificationsPageState();
}

class _AdminNotificationsPageState extends State<AdminNotificationsPage> {
  final _repository = const BankRepository();
  bool _loading = true;
  String? _error;
  List<Map<String, dynamic>> _items = const [];
  final Set<int> _busyIDs = {};

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load({bool showLoading = true}) async {
    if (showLoading) {
      setState(() {
        _loading = true;
        _error = null;
      });
    }
    try {
      final items = await _repository.notifications();
      if (!mounted) return;
      setState(() => _items = items);
      _notifyUnreadCount();
    } catch (error) {
      if (mounted) setState(() => _error = '$error');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  void _notifyUnreadCount() {
    widget.onUnreadCountChanged?.call(
      _items.where((item) => item['is_read'] != true).length,
    );
  }

  Future<void> _markRead(Map<String, dynamic> item) async {
    final id = (item['id'] as num?)?.toInt();
    if (id == null || _busyIDs.contains(id)) return;
    setState(() => _busyIDs.add(id));
    try {
      await _repository.markNotificationRead(id);
      if (!mounted) return;
      setState(() {
        final index = _items.indexWhere(
          (candidate) => candidate['id'] == item['id'],
        );
        if (index >= 0) {
          _items = [..._items];
          _items[index] = {..._items[index], 'is_read': true};
        }
      });
      _notifyUnreadCount();
    } on ApiException catch (error) {
      if (mounted) showMessage(context, error.message, error: true);
    } finally {
      if (mounted) setState(() => _busyIDs.remove(id));
    }
  }

  Future<void> _markAll() async {
    try {
      await _repository.markAllNotificationsRead();
      if (!mounted) return;
      setState(() {
        _items = [
          for (final item in _items) {...item, 'is_read': true},
        ];
      });
      _notifyUnreadCount();
    } on ApiException catch (error) {
      if (mounted) showMessage(context, error.message, error: true);
    }
  }

  @override
  Widget build(BuildContext context) {
    final unread = _items.where((item) => item['is_read'] != true).length;
    return AsyncPage(
      loading: _loading,
      error: _error,
      onRetry: _load,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          AdminSectionHeader(
            title: 'Thông báo quản trị',
            subtitle: '$unread thông báo chưa đọc',
            trailing: TextButton.icon(
              onPressed: unread == 0 ? null : _markAll,
              icon: const Icon(Icons.done_all_rounded),
              label: const Text('Đọc tất cả'),
            ),
          ),
          const SizedBox(height: 14),
          Expanded(
            child: AdminPanel(
              padding: EdgeInsets.zero,
              child: _items.isEmpty
                  ? const EmptyState(
                      icon: Icons.notifications_none_rounded,
                      title: 'Chưa có thông báo',
                      message:
                          'Cảnh báo bảo mật và kết quả nghiệp vụ sẽ xuất hiện tại đây.',
                    )
                  : RefreshIndicator(
                      onRefresh: () => _load(showLoading: false),
                      child: ListView.separated(
                        physics: const AlwaysScrollableScrollPhysics(),
                        itemCount: _items.length,
                        separatorBuilder: (_, _) => const Divider(height: 1),
                        itemBuilder: (context, index) {
                          final item = _items[index];
                          final id = (item['id'] as num?)?.toInt();
                          final read = item['is_read'] == true;
                          final security =
                              item['type']?.toString().contains('SECURITY') ==
                              true;
                          final accent = security
                              ? const Color(0xFFFFB566)
                              : const Color(0xFF8F98FF);
                          return ListTile(
                            onTap: read ? null : () => _markRead(item),
                            contentPadding: const EdgeInsets.symmetric(
                              horizontal: 16,
                              vertical: 7,
                            ),
                            tileColor: read
                                ? Colors.transparent
                                : accent.withValues(alpha: .045),
                            leading: Container(
                              width: 40,
                              height: 40,
                              decoration: BoxDecoration(
                                color: accent.withValues(alpha: .14),
                                borderRadius: BorderRadius.circular(12),
                              ),
                              child: Icon(
                                security
                                    ? Icons.security_rounded
                                    : Icons.notifications_active_outlined,
                                color: accent,
                                size: 20,
                              ),
                            ),
                            title: Text(
                              item['title']?.toString() ?? 'Thông báo',
                              style: TextStyle(
                                fontWeight: read
                                    ? FontWeight.w600
                                    : FontWeight.w900,
                              ),
                            ),
                            subtitle: Padding(
                              padding: const EdgeInsets.only(top: 4),
                              child: Text(
                                '${item['content'] ?? ''}\n${dateTimeText(item['created_at'])}',
                                maxLines: 3,
                                overflow: TextOverflow.ellipsis,
                              ),
                            ),
                            trailing: id != null && _busyIDs.contains(id)
                                ? const SizedBox.square(
                                    dimension: 18,
                                    child: CircularProgressIndicator(
                                      strokeWidth: 2,
                                    ),
                                  )
                                : read
                                ? null
                                : Container(
                                    width: 8,
                                    height: 8,
                                    decoration: BoxDecoration(
                                      color: accent,
                                      shape: BoxShape.circle,
                                    ),
                                  ),
                          );
                        },
                      ),
                    ),
            ),
          ),
        ],
      ),
    );
  }
}
