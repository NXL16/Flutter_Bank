import 'package:flutter/material.dart';

import '../../../core/network/api_service.dart';
import '../../../core/storage/token_storage.dart';
import '../../../shared/utils/formatters.dart';
import '../../../shared/widgets/common.dart';
import '../models/admin_models.dart';
import '../services/admin_repository.dart';
import '../widgets/admin_detail_dialog.dart';
import '../widgets/admin_widgets.dart';
import 'admin_monitoring_page.dart';

class AdminOverviewPage extends StatefulWidget {
  const AdminOverviewPage({
    super.key,
    required this.user,
    this.onTabRequested,
    this.onMonitoringRequested,
  });

  final SessionUser user;
  final ValueChanged<int>? onTabRequested;
  final ValueChanged<AdminMonitoringTab>? onMonitoringRequested;

  @override
  State<AdminOverviewPage> createState() => _AdminOverviewPageState();
}

class _AdminOverviewPageState extends State<AdminOverviewPage> {
  final _repository = const AdminRepository();
  bool _loading = true;
  String? _error;
  AdminDashboard? _dashboard;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      _dashboard = await _repository.dashboard();
    } catch (error) {
      _error = '$error';
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  void _showTransactionDetail(AdminTransactionSummary tx) {
    showAdminTransactionSummaryDetail(context, tx);
  }

  void _showAuditDetail(AdminAuditLog audit) {
    showDialog<void>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        icon: const Icon(
          Icons.fact_check_outlined,
          color: Color(0xFFFFB566),
          size: 36,
        ),
        title: const Text('Chi tiết hành động'),
        content: SizedBox(
          width: 320,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                audit.action,
                style: const TextStyle(
                  fontSize: 20,
                  fontWeight: FontWeight.w900,
                  color: Color(0xFFFFB566),
                ),
              ),
              const SizedBox(height: 14),
              AdminInfoRow('Người thực hiện', audit.actorName),
              AdminInfoRow('Đối tượng', audit.targetType),
              AdminInfoRow('Mã đối tượng', audit.targetId),
              AdminInfoRow('Địa chỉ IP', audit.ipAddress),
              AdminInfoRow('Mô tả hành động', audit.summary),
              AdminInfoRow('Thời gian', dateTimeText(audit.createdAt)),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(dialogContext),
            child: const Text('Đóng'),
          ),
        ],
      ),
    );
  }

  Future<void> _showDetail(
    AdminUserSummary summary, {
    bool showBackButton = false,
  }) async {
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

  void _showPersonalTxDetail(Map<String, dynamic> tx, String customerName) {
    showAdminAccountTransactionDetail(context, tx, customerName: customerName);
  }

  Future<void> _showTransactions(
    int accountID, {
    required bool showBackButton,
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
                              onTap: () => _showPersonalTxDetail(
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

  Future<void> _showAllPaymentBalances() async {
    showDialog<void>(
      context: context,
      barrierDismissible: false,
      builder: (context) => const Center(child: CircularProgressIndicator()),
    );

    try {
      final users = (await _repository.users())
          .where((u) => u.role == 'user')
          .toList();
      final accountsList = await Future.wait(
        users.map((u) => _repository.userAccounts(u.id)),
      );

      if (!mounted) return;
      Navigator.pop(context); // Pop loading dialog

      final List<Map<String, dynamic>> displayData = [];
      for (int i = 0; i < users.length; i++) {
        final user = users[i];
        final accounts = accountsList[i];
        final paymentAcc = accounts.firstWhere(
          (acc) => acc['account_type'] == 'PAYMENT',
          orElse: () => <String, dynamic>{},
        );
        if (paymentAcc.isNotEmpty) {
          displayData.add({
            'user': user,
            'account_number': paymentAcc['account_number'] ?? '—',
            'balance': paymentAcc['balance'] ?? 0,
          });
        }
      }

      await showModalBottomSheet<void>(
        context: context,
        useSafeArea: true,
        isScrollControlled: true,
        backgroundColor: Colors.transparent,
        builder: (sheetContext) => FractionallySizedBox(
          heightFactor: .75,
          child: Container(
            decoration: const BoxDecoration(
              color: Color(0xFF11192B),
              borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
              border: Border(top: BorderSide(color: Color(0xFF344263))),
            ),
            child: Column(
              children: [
                Padding(
                  padding: const EdgeInsets.fromLTRB(20, 18, 8, 12),
                  child: Row(
                    children: [
                      const Expanded(
                        child: AdminSectionHeader(
                          title: 'Tiền gửi thanh toán',
                          subtitle: 'Số dư tài khoản thanh toán của khách hàng',
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
                  child: displayData.isEmpty
                      ? const EmptyState(
                          icon: Icons.account_balance_wallet_outlined,
                          title: 'Không có tài khoản',
                          message: 'Chưa có tài khoản thanh toán nào.',
                        )
                      : ListView.separated(
                          padding: const EdgeInsets.all(14),
                          itemCount: displayData.length,
                          separatorBuilder: (_, _) => const Divider(height: 1),
                          itemBuilder: (context, index) {
                            final item = displayData[index];
                            final user = item['user'] as AdminUserSummary;
                            return ListTile(
                              leading: AdminUserAvatar(user: user),
                              title: Text(user.fullName),
                              subtitle: Text('STK: ${item['account_number']}'),
                              trailing: Text(
                                money(item['balance']),
                                style: const TextStyle(
                                  fontSize: 14,
                                  fontWeight: FontWeight.w800,
                                  color: Color(0xFF53C7DA),
                                ),
                              ),
                              onTap: () {
                                _showDetail(user, showBackButton: true);
                              },
                            );
                          },
                        ),
                ),
              ],
            ),
          ),
        ),
      );
    } catch (e) {
      if (mounted) {
        Navigator.pop(context); // Pop loading
        showMessage(context, 'Không thể tải số dư thanh toán', error: true);
      }
    }
  }

  Future<void> _showAllSavingsBalances() async {
    showDialog<void>(
      context: context,
      barrierDismissible: false,
      builder: (context) => const Center(child: CircularProgressIndicator()),
    );

    try {
      final users = (await _repository.users())
          .where((u) => u.role == 'user')
          .toList();
      final accountsList = await Future.wait(
        users.map((u) => _repository.userAccounts(u.id)),
      );

      if (!mounted) return;
      Navigator.pop(context); // Pop loading dialog

      final List<Map<String, dynamic>> displayData =
          []; // list of { 'user': user, 'savings': savingsList }
      for (int i = 0; i < users.length; i++) {
        final user = users[i];
        final accounts = accountsList[i];
        final savingsAccs = accounts
            .where((acc) => acc['account_type'] == 'SAVINGS')
            .toList();
        if (savingsAccs.isNotEmpty) {
          displayData.add({'user': user, 'savings': savingsAccs});
        }
      }

      await showModalBottomSheet<void>(
        context: context,
        useSafeArea: true,
        isScrollControlled: true,
        backgroundColor: Colors.transparent,
        builder: (sheetContext) => FractionallySizedBox(
          heightFactor: .75,
          child: Container(
            decoration: const BoxDecoration(
              color: Color(0xFF11192B),
              borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
              border: Border(top: BorderSide(color: Color(0xFF344263))),
            ),
            child: Column(
              children: [
                Padding(
                  padding: const EdgeInsets.fromLTRB(20, 18, 8, 12),
                  child: Row(
                    children: [
                      const Expanded(
                        child: AdminSectionHeader(
                          title: 'Tiết kiệm đang hoạt động',
                          subtitle: 'Danh sách khách hàng đang có sổ tiết kiệm',
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
                  child: displayData.isEmpty
                      ? const EmptyState(
                          icon: Icons.savings_outlined,
                          title: 'Không có sổ tiết kiệm',
                          message: 'Khách hàng chưa mở sổ tiết kiệm nào.',
                        )
                      : ListView.separated(
                          padding: const EdgeInsets.all(14),
                          itemCount: displayData.length,
                          separatorBuilder: (_, _) => const Divider(height: 1),
                          itemBuilder: (context, index) {
                            final item = displayData[index];
                            final user = item['user'] as AdminUserSummary;
                            final savings =
                                item['savings'] as List<Map<String, dynamic>>;

                            // Calculate total savings balance for this user
                            final totalBalance = savings.fold<double>(
                              0.0,
                              (sum, acc) =>
                                  sum +
                                  (acc['balance'] as num? ?? 0).toDouble(),
                            );

                            return ListTile(
                              leading: AdminUserAvatar(user: user),
                              title: Text(user.fullName),
                              subtitle: Text(
                                'Đang có ${savings.length} sổ tiết kiệm',
                              ),
                              trailing: Text(
                                money(totalBalance),
                                style: const TextStyle(
                                  fontSize: 14,
                                  fontWeight: FontWeight.w800,
                                  color: Color(0xFF50D4A3),
                                ),
                              ),
                              onTap: () {
                                _showUserSavingsSheet(
                                  user,
                                  savings,
                                  showBackButton: true,
                                );
                              },
                            );
                          },
                        ),
                ),
              ],
            ),
          ),
        ),
      );
    } catch (e) {
      if (mounted) {
        Navigator.pop(context); // Pop loading
        showMessage(
          context,
          'Không thể tải danh sách sổ tiết kiệm',
          error: true,
        );
      }
    }
  }

  Future<void> _showUserSavingsSheet(
    AdminUserSummary user,
    List<Map<String, dynamic>> savings, {
    bool showBackButton = false,
  }) async {
    await showModalBottomSheet<void>(
      context: context,
      useSafeArea: true,
      isScrollControlled: true,
      backgroundColor: Colors.transparent,
      builder: (sheetContext) => FractionallySizedBox(
        heightFactor: .65,
        child: Container(
          decoration: const BoxDecoration(
            color: Color(0xFF11192B),
            borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
            border: Border(top: BorderSide(color: Color(0xFF344263))),
          ),
          child: Column(
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(20, 18, 8, 12),
                child: Row(
                  children: [
                    Expanded(
                      child: AdminSectionHeader(
                        title: 'Sổ tiết kiệm',
                        subtitle: 'Danh sách sổ của ${user.fullName}',
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
                child: ListView.separated(
                  padding: const EdgeInsets.all(14),
                  itemCount: savings.length,
                  separatorBuilder: (_, _) => const Divider(height: 1),
                  itemBuilder: (context, index) {
                    final acc = savings[index];
                    return ListTile(
                      leading: Container(
                        width: 38,
                        height: 38,
                        decoration: BoxDecoration(
                          color: const Color(0xFF273457),
                          borderRadius: BorderRadius.circular(10),
                        ),
                        child: const Icon(
                          Icons.savings_outlined,
                          color: Color(0xFF50D4A3),
                          size: 20,
                        ),
                      ),
                      title: Text(acc['account_number']?.toString() ?? '—'),
                      subtitle: Text(
                        'Trạng thái: ${acc['status'] ?? 'ACTIVE'}',
                      ),
                      trailing: Text(
                        money(acc['balance']),
                        style: const TextStyle(
                          fontSize: 14,
                          fontWeight: FontWeight.w800,
                          color: Color(0xFF50D4A3),
                        ),
                      ),
                      onTap: () async {
                        await _showTransactions(
                          (acc['id'] as num?)?.toInt() ?? 0,
                          showBackButton: true,
                          customerName: user.fullName,
                        );
                      },
                    );
                  },
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _showTodayTransactionsDetail() async {
    showDialog<void>(
      context: context,
      barrierDismissible: false,
      builder: (context) => const Center(child: CircularProgressIndicator()),
    );

    try {
      final allTxs = await _repository.transactions(limit: 200);
      final now = DateTime.now();
      final todayTxs = allTxs.where((tx) {
        if (tx.createdAt == null) return false;
        final date = tx.createdAt!;
        return date.year == now.year &&
            date.month == now.month &&
            date.day == now.day;
      }).toList();

      if (!mounted) return;
      Navigator.pop(context); // Pop loading dialog

      await showModalBottomSheet<void>(
        context: context,
        useSafeArea: true,
        isScrollControlled: true,
        backgroundColor: Colors.transparent,
        builder: (sheetContext) => FractionallySizedBox(
          heightFactor: .75,
          child: Container(
            decoration: const BoxDecoration(
              color: Color(0xFF11192B),
              borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
              border: Border(top: BorderSide(color: Color(0xFF344263))),
            ),
            child: Column(
              children: [
                Padding(
                  padding: const EdgeInsets.fromLTRB(20, 18, 8, 12),
                  child: Row(
                    children: [
                      Expanded(
                        child: AdminSectionHeader(
                          title: 'Giao dịch hôm nay',
                          subtitle:
                              'Hôm nay ghi nhận ${todayTxs.length} giao dịch thành công',
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
                  child: todayTxs.isEmpty
                      ? const EmptyState(
                          icon: Icons.swap_horiz_rounded,
                          title: 'Chưa có giao dịch hôm nay',
                          message:
                              'Giao dịch phát sinh hôm nay sẽ hiển thị ở đây.',
                        )
                      : ListView.separated(
                          padding: const EdgeInsets.all(14),
                          itemCount: todayTxs.length,
                          separatorBuilder: (_, _) => const Divider(height: 1),
                          itemBuilder: (context, index) {
                            final tx = todayTxs[index];
                            return ListTile(
                              leading: Container(
                                width: 35,
                                height: 35,
                                decoration: BoxDecoration(
                                  color: const Color(0xFF283657),
                                  borderRadius: BorderRadius.circular(10),
                                ),
                                child: const Icon(
                                  Icons.swap_horiz_rounded,
                                  size: 18,
                                  color: Color(0xFFFFB566),
                                ),
                              ),
                              title: Text(
                                tx.description.isEmpty
                                    ? tx.type
                                    : tx.description,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                              ),
                              subtitle: Text(
                                '${tx.referenceCode} · ${dateTimeText(tx.createdAt)}',
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                              ),
                              trailing: Text(
                                money(tx.amount, tx.currency),
                                style: const TextStyle(
                                  fontSize: 14,
                                  fontWeight: FontWeight.w800,
                                  color: Color(0xFFFFB566),
                                ),
                              ),
                              onTap: () {
                                _showTransactionDetail(tx);
                              },
                            );
                          },
                        ),
                ),
              ],
            ),
          ),
        ),
      );
    } catch (e) {
      if (mounted) {
        Navigator.pop(context); // Pop loading
        showMessage(context, 'Không thể tải giao dịch hôm nay', error: true);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final dashboard = _dashboard;
    return AsyncPage(
      loading: _loading,
      error: _error,
      onRetry: _load,
      child: dashboard == null
          ? const SizedBox.shrink()
          : RefreshIndicator(
              onRefresh: _load,
              child: ListView(
                children: [
                  _WelcomeBanner(user: widget.user),
                  const SizedBox(height: 18),
                  _MetricsGrid(
                    dashboard: dashboard,
                    onCustomersTap: () => widget.onTabRequested?.call(1),
                    onPaymentTap: _showAllPaymentBalances,
                    onSavingsTap: _showAllSavingsBalances,
                    onTodayTxsTap: _showTodayTransactionsDetail,
                  ),
                  const SizedBox(height: 18),
                  LayoutBuilder(
                    builder: (context, constraints) {
                      final wide = constraints.maxWidth >= 840;
                      final transactions = _RecentTransactions(
                        items: dashboard.recentTransactions,
                        onTap: _showTransactionDetail,
                        onViewAll: () => widget.onMonitoringRequested?.call(
                          AdminMonitoringTab.transactions,
                        ),
                      );
                      final audits = _RecentAudits(
                        items: dashboard.recentAuditLogs,
                        onTap: _showAuditDetail,
                        onViewAll: () => widget.onMonitoringRequested?.call(
                          AdminMonitoringTab.audit,
                        ),
                      );
                      if (!wide) {
                        return Column(
                          children: [
                            transactions,
                            const SizedBox(height: 16),
                            audits,
                          ],
                        );
                      }
                      return Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Expanded(flex: 3, child: transactions),
                          const SizedBox(width: 16),
                          Expanded(flex: 2, child: audits),
                        ],
                      );
                    },
                  ),
                ],
              ),
            ),
    );
  }
}

class _WelcomeBanner extends StatelessWidget {
  const _WelcomeBanner({required this.user});

  final SessionUser user;

  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 18),
    decoration: BoxDecoration(
      gradient: const LinearGradient(
        colors: [Color(0xFF26356C), Color(0xFF172641)],
        begin: Alignment.topLeft,
        end: Alignment.bottomRight,
      ),
      borderRadius: BorderRadius.circular(18),
      border: Border.all(color: const Color(0xFF3D4D83)),
    ),
    child: Row(
      children: [
        Container(
          width: 46,
          height: 46,
          decoration: BoxDecoration(
            color: Colors.white.withValues(alpha: .1),
            borderRadius: BorderRadius.circular(14),
          ),
          child: const Icon(
            Icons.monitor_heart_outlined,
            color: Color(0xFFB9C0FF),
          ),
        ),
        const SizedBox(width: 14),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'Trung tâm điều hành',
                style: Theme.of(
                  context,
                ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w900),
              ),
              const SizedBox(height: 3),
              Text(
                '${user.fullName} · ${adminRoleLabel(user.role)}',
                style: const TextStyle(color: Color(0xFFB0BAD4), fontSize: 12),
              ),
            ],
          ),
        ),
        const AdminStatusBadge(
          label: 'HỆ THỐNG HOẠT ĐỘNG',
          color: Color(0xFF50D4A3),
        ),
      ],
    ),
  );
}

class _MetricsGrid extends StatelessWidget {
  const _MetricsGrid({
    required this.dashboard,
    required this.onCustomersTap,
    required this.onPaymentTap,
    required this.onSavingsTap,
    required this.onTodayTxsTap,
  });

  final AdminDashboard dashboard;
  final VoidCallback onCustomersTap;
  final VoidCallback onPaymentTap;
  final VoidCallback onSavingsTap;
  final VoidCallback onTodayTxsTap;

  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, constraints) {
      final columns = constraints.maxWidth >= 1040
          ? 4
          : constraints.maxWidth >= 560
          ? 2
          : 1;
      final width = (constraints.maxWidth - (columns - 1) * 12) / columns;
      final cards = [
        AdminMetricCard(
          label: 'KHÁCH HÀNG',
          value: '${dashboard.customerCount}',
          caption: '${dashboard.lockedCustomerCount} tài khoản đang bị khóa',
          icon: Icons.people_alt_outlined,
          accent: const Color(0xFF8F98FF),
          onTap: onCustomersTap,
        ),
        AdminMetricCard(
          label: 'TIỀN GỬI THANH TOÁN',
          value: money(dashboard.paymentBalance),
          caption: 'Tổng số dư khách hàng',
          icon: Icons.account_balance_wallet_outlined,
          accent: const Color(0xFF53C7DA),
          onTap: onPaymentTap,
        ),
        AdminMetricCard(
          label: 'TIẾT KIỆM ĐANG HOẠT ĐỘNG',
          value: money(dashboard.activeSavingsBalance),
          caption: '${dashboard.activeSavingsCount} sổ đang hoạt động',
          icon: Icons.savings_outlined,
          accent: const Color(0xFF50D4A3),
          onTap: onSavingsTap,
        ),
        AdminMetricCard(
          label: 'GIAO DỊCH HÔM NAY',
          value: money(dashboard.todayTransactionValue),
          caption: '${dashboard.todayTransactionCount} giao dịch thành công',
          icon: Icons.swap_horiz_rounded,
          accent: const Color(0xFFFFB566),
          onTap: onTodayTxsTap,
        ),
      ];
      return Wrap(
        spacing: 12,
        runSpacing: 12,
        children: [
          for (final card in cards) SizedBox(width: width, child: card),
        ],
      );
    },
  );
}

class _RecentTransactions extends StatelessWidget {
  const _RecentTransactions({
    required this.items,
    required this.onTap,
    required this.onViewAll,
  });

  final List<AdminTransactionSummary> items;
  final ValueChanged<AdminTransactionSummary> onTap;
  final VoidCallback onViewAll;

  @override
  Widget build(BuildContext context) => AdminPanel(
    padding: EdgeInsets.zero,
    child: Column(
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(18, 16, 8, 12),
          child: AdminSectionHeader(
            title: 'Giao dịch mới nhất',
            subtitle: 'Luồng tiền toàn hệ thống',
            trailing: TextButton(
              onPressed: onViewAll,
              child: const Text('Xem tất cả'),
            ),
          ),
        ),
        const Divider(height: 1),
        if (items.isEmpty)
          const EmptyState(
            icon: Icons.receipt_long_outlined,
            title: 'Chưa có giao dịch',
            message: 'Dữ liệu vận hành sẽ xuất hiện tại đây.',
          )
        else
          for (final item in items)
            ListTile(
              onTap: () => onTap(item),
              dense: true,
              contentPadding: const EdgeInsets.symmetric(
                horizontal: 18,
                vertical: 3,
              ),
              leading: Container(
                width: 35,
                height: 35,
                decoration: BoxDecoration(
                  color: const Color(0xFF283657),
                  borderRadius: BorderRadius.circular(10),
                ),
                child: const Icon(
                  Icons.swap_horiz_rounded,
                  size: 18,
                  color: Color(0xFFAEB5FF),
                ),
              ),
              title: Text(
                item.description.isEmpty ? item.type : item.description,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
              subtitle: Text(
                '${item.referenceCode} · ${dateTimeText(item.createdAt)}',
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
              trailing: Text(
                money(item.amount, item.currency),
                style: const TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w800,
                ),
              ),
            ),
      ],
    ),
  );
}

class _RecentAudits extends StatelessWidget {
  const _RecentAudits({
    required this.items,
    required this.onTap,
    required this.onViewAll,
  });

  final List<AdminAuditLog> items;
  final ValueChanged<AdminAuditLog> onTap;
  final VoidCallback onViewAll;

  @override
  Widget build(BuildContext context) => AdminPanel(
    padding: EdgeInsets.zero,
    child: Column(
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(18, 16, 8, 12),
          child: AdminSectionHeader(
            title: 'Nhật ký quản trị',
            subtitle: 'Hành động nhạy cảm gần đây',
            trailing: TextButton(
              onPressed: onViewAll,
              child: const Text('Xem tất cả'),
            ),
          ),
        ),
        const Divider(height: 1),
        if (items.isEmpty)
          const EmptyState(
            icon: Icons.fact_check_outlined,
            title: 'Chưa có nhật ký',
            message: 'Các thao tác quản trị sẽ được ghi lại.',
          )
        else
          for (final item in items)
            ListTile(
              onTap: () => onTap(item),
              dense: true,
              contentPadding: const EdgeInsets.symmetric(
                horizontal: 18,
                vertical: 2,
              ),
              leading: const Icon(
                Icons.verified_user_outlined,
                size: 20,
                color: Color(0xFFFFB566),
              ),
              title: Text(
                item.summary,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(fontSize: 12),
              ),
              subtitle: Text(dateTimeText(item.createdAt)),
            ),
      ],
    ),
  );
}
