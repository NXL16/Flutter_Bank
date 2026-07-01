import 'package:flutter/material.dart';

import '../../../shared/utils/formatters.dart';
import '../../../shared/widgets/common.dart';
import '../models/admin_models.dart';
import '../services/admin_repository.dart';
import '../widgets/admin_detail_dialog.dart';
import '../widgets/admin_widgets.dart';

enum AdminMonitoringTab { transactions, audit }

class AdminMonitoringPage extends StatefulWidget {
  const AdminMonitoringPage({super.key, required this.navigation});

  final ValueNotifier<AdminMonitoringTab> navigation;

  @override
  State<AdminMonitoringPage> createState() => _AdminMonitoringPageState();
}

class _AdminMonitoringPageState extends State<AdminMonitoringPage>
    with SingleTickerProviderStateMixin {
  final _repository = const AdminRepository();
  final _search = TextEditingController();
  late final TabController _tabs;
  bool _loading = true;
  String? _error;
  List<AdminTransactionSummary> _transactions = const [];
  List<AdminAuditLog> _auditLogs = const [];

  @override
  void initState() {
    super.initState();
    _tabs = TabController(
      length: 2,
      vsync: this,
      initialIndex: widget.navigation.value.index,
    );
    widget.navigation.addListener(_handleNavigation);
    _load();
  }

  @override
  void dispose() {
    widget.navigation.removeListener(_handleNavigation);
    _tabs.dispose();
    _search.dispose();
    super.dispose();
  }

  void _handleNavigation() {
    _tabs.animateTo(widget.navigation.value.index);
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final results = await Future.wait([
        _repository.transactions(limit: 200),
        _repository.auditLogs(limit: 200),
      ]);
      if (!mounted) return;
      setState(() {
        _transactions = results[0] as List<AdminTransactionSummary>;
        _auditLogs = results[1] as List<AdminAuditLog>;
      });
    } catch (error) {
      if (mounted) setState(() => _error = '$error');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final query = _search.text.trim().toLowerCase();
    final transactions = _transactions.where((item) {
      return query.isEmpty ||
          '${item.referenceCode} ${item.type} ${item.description}'
              .toLowerCase()
              .contains(query);
    }).toList();
    final audits = _auditLogs.where((item) {
      return query.isEmpty ||
          '${item.actorName} ${item.action} ${item.summary} ${item.targetId}'
              .toLowerCase()
              .contains(query);
    }).toList();

    return AsyncPage(
      loading: _loading,
      error: _error,
      onRetry: _load,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          AdminSectionHeader(
            title: 'Giám sát hệ thống',
            subtitle: 'Giao dịch tài chính và nhật ký quản trị bất biến',
            trailing: IconButton(
              tooltip: 'Làm mới',
              onPressed: _load,
              icon: const Icon(Icons.refresh_rounded),
            ),
          ),
          const SizedBox(height: 14),
          AdminPanel(
            padding: const EdgeInsets.fromLTRB(12, 10, 12, 12),
            child: Column(
              children: [
                TabBar(
                  controller: _tabs,
                  tabs: const [
                    Tab(
                      icon: Icon(Icons.swap_horiz_rounded),
                      text: 'Giao dịch',
                    ),
                    Tab(
                      icon: Icon(Icons.fact_check_outlined),
                      text: 'Nhật ký quản trị',
                    ),
                  ],
                ),
                const SizedBox(height: 10),
                TextField(
                  controller: _search,
                  onChanged: (_) => setState(() {}),
                  decoration:
                      fieldDecoration(
                        'Tìm kiếm',
                        hint: 'Mã giao dịch, nội dung, Admin...',
                      ).copyWith(
                        prefixIcon: const Icon(Icons.search_rounded),
                        suffixIcon: _search.text.isEmpty
                            ? null
                            : IconButton(
                                onPressed: () {
                                  _search.clear();
                                  setState(() {});
                                },
                                icon: const Icon(Icons.close_rounded),
                              ),
                      ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 12),
          Expanded(
            child: TabBarView(
              controller: _tabs,
              children: [
                _TransactionList(items: transactions),
                _AuditList(items: audits),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _TransactionList extends StatelessWidget {
  const _TransactionList({required this.items});

  final List<AdminTransactionSummary> items;

  @override
  Widget build(BuildContext context) => AdminPanel(
    padding: EdgeInsets.zero,
    child: items.isEmpty
        ? const EmptyState(
            icon: Icons.receipt_long_outlined,
            title: 'Không có giao dịch',
            message: 'Không tìm thấy giao dịch phù hợp.',
          )
        : ListView.separated(
            itemCount: items.length,
            separatorBuilder: (_, _) => const Divider(height: 1),
            itemBuilder: (context, index) {
              final item = items[index];
              return ListTile(
                onTap: () => showAdminTransactionSummaryDetail(context, item),
                contentPadding: const EdgeInsets.symmetric(
                  horizontal: 16,
                  vertical: 5,
                ),
                leading: Container(
                  width: 39,
                  height: 39,
                  decoration: BoxDecoration(
                    color: const Color(0xFF273457),
                    borderRadius: BorderRadius.circular(11),
                  ),
                  child: const Icon(
                    Icons.swap_horiz_rounded,
                    color: Color(0xFFAEB5FF),
                  ),
                ),
                title: Text(
                  item.description.isEmpty ? item.type : item.description,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                subtitle: Text(
                  '${item.referenceCode} · ${item.type} · ${dateTimeText(item.createdAt)}',
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                trailing: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Column(
                      mainAxisAlignment: MainAxisAlignment.center,
                      crossAxisAlignment: CrossAxisAlignment.end,
                      children: [
                        Text(
                          money(item.amount, item.currency),
                          style: const TextStyle(
                            fontSize: 12,
                            fontWeight: FontWeight.w900,
                          ),
                        ),
                        const SizedBox(height: 3),
                        AdminStatusBadge(
                          label: item.status,
                          color: item.status == 'SUCCESS'
                              ? const Color(0xFF50D4A3)
                              : const Color(0xFFFF7D8A),
                        ),
                      ],
                    ),
                    const SizedBox(width: 7),
                    const Icon(
                      Icons.chevron_right_rounded,
                      color: Color(0xFF77829C),
                      size: 20,
                    ),
                  ],
                ),
              );
            },
          ),
  );
}

class _AuditList extends StatelessWidget {
  const _AuditList({required this.items});

  final List<AdminAuditLog> items;

  @override
  Widget build(BuildContext context) => AdminPanel(
    padding: EdgeInsets.zero,
    child: items.isEmpty
        ? const EmptyState(
            icon: Icons.fact_check_outlined,
            title: 'Không có nhật ký',
            message: 'Không tìm thấy hành động quản trị phù hợp.',
          )
        : ListView.separated(
            itemCount: items.length,
            separatorBuilder: (_, _) => const Divider(height: 1),
            itemBuilder: (context, index) {
              final item = items[index];
              return ListTile(
                onTap: () => showAdminAuditDetail(context, item),
                contentPadding: const EdgeInsets.symmetric(
                  horizontal: 16,
                  vertical: 5,
                ),
                leading: Container(
                  width: 39,
                  height: 39,
                  decoration: BoxDecoration(
                    color: const Color(0x33FFB566),
                    borderRadius: BorderRadius.circular(11),
                  ),
                  child: const Icon(
                    Icons.verified_user_outlined,
                    color: Color(0xFFFFB566),
                  ),
                ),
                title: Text(
                  item.summary,
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                ),
                subtitle: Text(
                  '${item.actorName} · ${dateTimeText(item.createdAt)}'
                  '${item.ipAddress.isEmpty ? '' : ' · ${item.ipAddress}'}',
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
                trailing: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    AdminStatusBadge(
                      label: item.action.replaceAll('_', ' '),
                      color: const Color(0xFFFFB566),
                    ),
                    const SizedBox(width: 7),
                    const Icon(
                      Icons.chevron_right_rounded,
                      color: Color(0xFF77829C),
                      size: 20,
                    ),
                  ],
                ),
              );
            },
          ),
  );
}
