import 'package:flutter/material.dart';

import '../../../core/network/api_service.dart';
import '../../../shared/utils/formatters.dart';
import '../../../shared/widgets/common.dart';
import '../models/admin_models.dart';
import '../services/admin_repository.dart';
import '../widgets/admin_detail_dialog.dart';
import '../widgets/admin_step_up.dart';
import '../widgets/admin_widgets.dart';

class AdminUsersPage extends StatefulWidget {
  const AdminUsersPage({super.key});

  @override
  State<AdminUsersPage> createState() => _AdminUsersPageState();
}

class _AdminUsersPageState extends State<AdminUsersPage> {
  final _repository = const AdminRepository();
  final _search = TextEditingController();
  bool _loading = true;
  String? _error;
  String _role = 'user';
  String _status = 'all';
  List<AdminUserSummary> _users = const [];
  final Set<int> _busyUserIDs = {};

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  Future<void> _load({bool showLoading = true}) async {
    if (showLoading) {
      setState(() {
        _loading = true;
        _error = null;
      });
    }
    try {
      final users = await _repository.users();
      if (mounted) setState(() => _users = users);
    } catch (error) {
      if (mounted) setState(() => _error = '$error');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  List<AdminUserSummary> get _filtered {
    final query = _search.text.trim().toLowerCase();
    final queryDigits = query.replaceAll(RegExp(r'\D'), '');

    String cleanDigits(String text) {
      var d = text.replaceAll(RegExp(r'\D'), '');
      if (d.startsWith('84')) return d.substring(2);
      if (d.startsWith('0')) return d.substring(1);
      return d;
    }

    final cleanQueryPhone = cleanDigits(query);

    return _users.where((user) {
      final matchesName = user.fullName.toLowerCase().contains(query);
      final matchesPhone =
          user.phone.toLowerCase().contains(query) ||
          (queryDigits.isNotEmpty &&
              cleanDigits(user.phone).contains(cleanQueryPhone));

      final matchesQuery = query.isEmpty || matchesName || matchesPhone;
      final matchesRole = _role == 'all' || user.role == _role;
      final matchesStatus =
          _status == 'all' ||
          (_status == 'active' && !user.isLocked) ||
          (_status == 'locked' && user.isLocked);
      return matchesQuery && matchesRole && matchesStatus;
    }).toList();
  }

  Future<void> _toggleLock(AdminUserSummary user) async {
    final willLock = !user.isLocked;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        icon: Icon(
          willLock ? Icons.lock_outline_rounded : Icons.lock_open_rounded,
          color: willLock ? const Color(0xFFFF7D8A) : const Color(0xFF50D4A3),
        ),
        title: Text(willLock ? 'Khóa tài khoản?' : 'Mở khóa tài khoản?'),
        content: Text(
          willLock
              ? '${user.fullName} sẽ bị đăng xuất khỏi mọi thiết bị và không thể đăng nhập cho tới khi được mở khóa.'
              : '${user.fullName} sẽ có thể đăng nhập và sử dụng dịch vụ trở lại.',
          textAlign: TextAlign.center,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(dialogContext, false),
            child: const Text('Hủy'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(dialogContext, true),
            style: FilledButton.styleFrom(
              backgroundColor: willLock
                  ? const Color(0xFFCE5261)
                  : const Color(0xFF357E68),
            ),
            child: Text(willLock ? 'Khóa tài khoản' : 'Mở khóa'),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;

    final stepUpToken = await showAdminStepUp(
      context,
      action: willLock
          ? AdminSensitiveAction.lockUser
          : AdminSensitiveAction.unlockUser,
      binding:
          '${willLock ? AdminSensitiveAction.lockUser : AdminSensitiveAction.unlockUser}|${user.id}',
      title: willLock ? 'Xác nhận khóa tài khoản' : 'Xác nhận mở khóa',
      transactionSummary:
          '${user.fullName}\n${user.phone}\n${willLock ? 'Khóa và thu hồi toàn bộ phiên' : 'Khôi phục quyền đăng nhập'}',
    );
    if (stepUpToken == null || !mounted) return;

    setState(() => _busyUserIDs.add(user.id));
    try {
      await _repository.setLocked(
        user.id,
        locked: willLock,
        stepUpToken: stepUpToken,
      );
      if (!mounted) return;
      showMessage(
        context,
        willLock ? 'Đã khóa tài khoản' : 'Đã mở khóa tài khoản',
      );
      await _load(showLoading: false);
    } on ApiException catch (error) {
      if (mounted) showMessage(context, error.message, error: true);
    } finally {
      if (mounted) setState(() => _busyUserIDs.remove(user.id));
    }
  }

  Future<void> _showDetail(AdminUserSummary summary) async {
    try {
      final results = await Future.wait([
        _repository.user(summary.id),
        _repository.userAccounts(summary.id),
      ]);
      if (!mounted) return;
      final user = results[0] as AdminUserSummary;
      final accounts = results[1] as List<Map<String, dynamic>>;
      await showModalBottomSheet<void>(
        context: context,
        useSafeArea: true,
        isScrollControlled: true,
        backgroundColor: Colors.transparent,
        builder: (sheetContext) => FractionallySizedBox(
          heightFactor: .82,
          child: Container(
            decoration: const BoxDecoration(
              color: Color(0xFF11192B),
              borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
              border: Border(top: BorderSide(color: Color(0xFF344263))),
            ),
            child: Column(
              children: [
                const SizedBox(height: 9),
                Container(
                  width: 42,
                  height: 4,
                  decoration: BoxDecoration(
                    color: Colors.white24,
                    borderRadius: BorderRadius.circular(10),
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.fromLTRB(20, 14, 8, 12),
                  child: Row(
                    children: [
                      AdminUserAvatar(user: user, radius: 24),
                      const SizedBox(width: 13),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              user.fullName,
                              style: const TextStyle(
                                fontSize: 18,
                                fontWeight: FontWeight.w900,
                              ),
                            ),
                            Text(
                              '${adminRoleLabel(user.role)} · ID #${user.id}',
                              style: const TextStyle(
                                color: Color(0xFF8792AD),
                                fontSize: 12,
                              ),
                            ),
                          ],
                        ),
                      ),
                      IconButton(
                        onPressed: () => Navigator.pop(sheetContext),
                        icon: const Icon(Icons.close_rounded),
                      ),
                    ],
                  ),
                ),
                const Divider(height: 1),
                Expanded(
                  child: ListView(
                    padding: const EdgeInsets.all(20),
                    children: [
                      AdminPanel(
                        child: Column(
                          children: [
                            AdminInfoRow('Số điện thoại', user.phone),
                            AdminInfoRow(
                              'Trạng thái',
                              user.isLocked ? 'Đã khóa' : 'Đang hoạt động',
                            ),
                            AdminInfoRow(
                              'Xác minh',
                              user.isVerified ? 'Đã xác minh' : 'Chưa xác minh',
                            ),
                            AdminInfoRow('Ngày tạo', shortDate(user.createdAt)),
                          ],
                        ),
                      ),
                      const SizedBox(height: 18),
                      AdminSectionHeader(
                        title: 'Tài khoản ngân hàng',
                        subtitle: '${accounts.length} tài khoản',
                      ),
                      const SizedBox(height: 10),
                      if (accounts.isEmpty)
                        const AdminPanel(
                          child: EmptyState(
                            icon: Icons.account_balance_wallet_outlined,
                            title: 'Chưa có tài khoản',
                            message: 'Người dùng chưa có tài khoản ngân hàng.',
                          ),
                        )
                      else
                        for (final account in accounts) ...[
                          AdminPanel(
                            padding: const EdgeInsets.symmetric(
                              horizontal: 15,
                              vertical: 10,
                            ),
                            child: ListTile(
                              contentPadding: EdgeInsets.zero,
                              leading: Container(
                                width: 40,
                                height: 40,
                                decoration: BoxDecoration(
                                  color: const Color(0xFF273457),
                                  borderRadius: BorderRadius.circular(11),
                                ),
                                child: Icon(
                                  account['account_type'] == 'SAVINGS'
                                      ? Icons.savings_outlined
                                      : Icons.account_balance_wallet_outlined,
                                  color: const Color(0xFFAEB5FF),
                                ),
                              ),
                              title: Text(
                                account['account_number']?.toString() ?? '—',
                              ),
                              subtitle: Text(
                                '${account['account_type'] ?? 'ACCOUNT'} · ${account['status'] ?? '—'}',
                              ),
                              trailing: Column(
                                mainAxisAlignment: MainAxisAlignment.center,
                                crossAxisAlignment: CrossAxisAlignment.end,
                                children: [
                                  Text(
                                    money(
                                      account['balance'],
                                      account['currency']?.toString() ?? 'VND',
                                    ),
                                    style: const TextStyle(
                                      fontSize: 12,
                                      fontWeight: FontWeight.w800,
                                    ),
                                  ),
                                  const SizedBox(height: 3),
                                  const Text(
                                    'Xem giao dịch',
                                    style: TextStyle(
                                      color: Color(0xFF9EA5FF),
                                      fontSize: 10,
                                    ),
                                  ),
                                ],
                              ),
                              onTap: () async {
                                await _showTransactions(
                                  (account['id'] as num?)?.toInt() ?? 0,
                                  showBackButton: true,
                                  customerName: user.fullName,
                                );
                              },
                            ),
                          ),
                          const SizedBox(height: 10),
                        ],
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      );
    } on ApiException catch (error) {
      if (mounted) showMessage(context, error.message, error: true);
    }
  }

  void _showTransactionDetail(Map<String, dynamic> tx, String customerName) {
    showAdminAccountTransactionDetail(context, tx, customerName: customerName);
  }

  Future<void> _showTransactions(
    int accountID, {
    bool showBackButton = false,
    required String customerName,
  }) async {
    if (accountID <= 0) return;
    try {
      final transactions = await _repository.accountTransactions(accountID);
      if (!mounted) return;
      await showModalBottomSheet<void>(
        context: context,
        useSafeArea: true,
        isScrollControlled: true,
        backgroundColor: Colors.transparent,
        builder: (sheetContext) => FractionallySizedBox(
          heightFactor: .72,
          child: Container(
            decoration: const BoxDecoration(
              color: Color(0xFF11192B),
              borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
            ),
            child: Column(
              children: [
                Padding(
                  padding: const EdgeInsets.fromLTRB(20, 18, 8, 12),
                  child: Row(
                    children: [
                      const Expanded(
                        child: AdminSectionHeader(
                          title: 'Lịch sử tài khoản',
                          subtitle: 'Giao dịch được ghi nhận trên sổ cái',
                        ),
                      ),
                      IconButton(
                        onPressed: () => Navigator.pop(sheetContext),
                        icon: Icon(
                          showBackButton
                              ? Icons.arrow_back_rounded
                              : Icons.close_rounded,
                        ),
                      ),
                    ],
                  ),
                ),
                const Divider(height: 1),
                Expanded(
                  child: transactions.isEmpty
                      ? const EmptyState(
                          icon: Icons.receipt_long_outlined,
                          title: 'Chưa có giao dịch',
                          message: 'Tài khoản chưa phát sinh giao dịch.',
                        )
                      : ListView.separated(
                          padding: const EdgeInsets.all(14),
                          itemCount: transactions.length,
                          separatorBuilder: (_, _) => const Divider(height: 1),
                          itemBuilder: (context, index) {
                            final transaction = transactions[index];
                            return ListTile(
                              onTap: () => _showTransactionDetail(
                                transaction,
                                customerName,
                              ),
                              leading: const Icon(Icons.swap_horiz_rounded),
                              title: Text(
                                transaction['description']
                                            ?.toString()
                                            .isNotEmpty ==
                                        true
                                    ? transaction['description'].toString()
                                    : transaction['type']?.toString() ??
                                          'Giao dịch',
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                              ),
                              subtitle: Text(
                                '${transaction['reference_code'] ?? '—'} · ${dateTimeText(transaction['created_at'])}',
                              ),
                              trailing: Text(
                                money(
                                  transaction['amount'],
                                  transaction['currency']?.toString() ?? 'VND',
                                ),
                                style: const TextStyle(
                                  fontSize: 12,
                                  fontWeight: FontWeight.w800,
                                ),
                              ),
                            );
                          },
                        ),
                ),
              ],
            ),
          ),
        ),
      );
    } on ApiException catch (error) {
      if (mounted) showMessage(context, error.message, error: true);
    }
  }

  @override
  Widget build(BuildContext context) {
    final filtered = _filtered;
    return AsyncPage(
      loading: _loading,
      error: _error,
      onRetry: _load,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          AdminSectionHeader(
            title: 'Khách hàng & tài khoản',
            subtitle: '${filtered.length}/${_users.length} kết quả',
            trailing: IconButton(
              tooltip: 'Làm mới',
              onPressed: _load,
              icon: const Icon(Icons.refresh_rounded),
            ),
          ),
          const SizedBox(height: 14),
          AdminPanel(
            padding: const EdgeInsets.all(12),
            child: LayoutBuilder(
              builder: (context, constraints) {
                final narrow = constraints.maxWidth < 620;
                final search = TextField(
                  controller: _search,
                  onChanged: (_) => setState(() {}),
                  decoration: fieldDecoration(
                    'Tìm tên hoặc số điện thoại',
                  ).copyWith(prefixIcon: const Icon(Icons.search_rounded)),
                );
                final filters = Row(
                  children: [
                    Expanded(
                      child: DropdownButtonFormField<String>(
                        initialValue: _role,
                        decoration: fieldDecoration('Vai trò'),
                        items: const [
                          DropdownMenuItem(
                            value: 'all',
                            child: Text('Tất cả vai trò'),
                          ),
                          DropdownMenuItem(
                            value: 'user',
                            child: Text('Khách hàng'),
                          ),
                          DropdownMenuItem(
                            value: 'admin',
                            child: Text('Admin'),
                          ),
                          DropdownMenuItem(
                            value: 'super_admin',
                            child: Text('Super Admin'),
                          ),
                        ],
                        onChanged: (value) =>
                            setState(() => _role = value ?? 'all'),
                      ),
                    ),
                    const SizedBox(width: 10),
                    Expanded(
                      child: DropdownButtonFormField<String>(
                        initialValue: _status,
                        decoration: fieldDecoration('Trạng thái'),
                        items: const [
                          DropdownMenuItem(value: 'all', child: Text('Tất cả')),
                          DropdownMenuItem(
                            value: 'active',
                            child: Text('Hoạt động'),
                          ),
                          DropdownMenuItem(
                            value: 'locked',
                            child: Text('Đã khóa'),
                          ),
                        ],
                        onChanged: (value) =>
                            setState(() => _status = value ?? 'all'),
                      ),
                    ),
                  ],
                );
                if (narrow) {
                  return Column(
                    children: [search, const SizedBox(height: 10), filters],
                  );
                }
                return Row(
                  children: [
                    Expanded(flex: 3, child: search),
                    const SizedBox(width: 10),
                    Expanded(flex: 2, child: filters),
                  ],
                );
              },
            ),
          ),
          const SizedBox(height: 12),
          Expanded(
            child: AdminPanel(
              padding: EdgeInsets.zero,
              child: filtered.isEmpty
                  ? const EmptyState(
                      icon: Icons.person_search_outlined,
                      title: 'Không có kết quả',
                      message: 'Hãy thay đổi từ khóa hoặc bộ lọc.',
                    )
                  : ListView.separated(
                      itemCount: filtered.length,
                      separatorBuilder: (_, _) => const Divider(height: 1),
                      itemBuilder: (context, index) {
                        final user = filtered[index];
                        final busy = _busyUserIDs.contains(user.id);
                        return ListTile(
                          onTap: () => _showDetail(user),
                          contentPadding: const EdgeInsets.symmetric(
                            horizontal: 16,
                            vertical: 5,
                          ),
                          leading: AdminUserAvatar(user: user),
                          title: Text(
                            user.fullName,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                          ),
                          subtitle: Text(
                            '${user.phone} · ${adminRoleLabel(user.role)}',
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                          ),
                          trailing: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              AdminStatusBadge(
                                label: user.isLocked ? 'ĐÃ KHÓA' : 'HOẠT ĐỘNG',
                                color: user.isLocked
                                    ? const Color(0xFFFF7D8A)
                                    : const Color(0xFF50D4A3),
                              ),
                              if (user.role == 'user') ...[
                                const SizedBox(width: 4),
                                IconButton(
                                  tooltip: user.isLocked
                                      ? 'Mở khóa'
                                      : 'Khóa tài khoản',
                                  onPressed: busy
                                      ? null
                                      : () => _toggleLock(user),
                                  icon: busy
                                      ? const SizedBox.square(
                                          dimension: 18,
                                          child: CircularProgressIndicator(
                                            strokeWidth: 2,
                                          ),
                                        )
                                      : Icon(
                                          user.isLocked
                                              ? Icons.lock_open_rounded
                                              : Icons.lock_outline_rounded,
                                        ),
                                ),
                              ],
                              const Icon(Icons.chevron_right_rounded),
                            ],
                          ),
                        );
                      },
                    ),
            ),
          ),
        ],
      ),
    );
  }
}
