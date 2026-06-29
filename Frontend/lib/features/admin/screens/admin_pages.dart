import 'package:flutter/material.dart';

import '../../../core/network/api_service.dart';
import '../../../core/storage/token_storage.dart';
import '../../../shared/utils/formatters.dart';
import '../../../shared/widgets/common.dart';
import '../services/admin_repository.dart';

const _adminRepo = AdminRepository();

class AdminOverviewPage extends StatefulWidget {
  const AdminOverviewPage({super.key, required this.user});

  final SessionUser user;

  @override
  State<AdminOverviewPage> createState() => _AdminOverviewPageState();
}

class _AdminOverviewPageState extends State<AdminOverviewPage> {
  bool loading = true;
  String? error;
  List<Map<String, dynamic>> users = [];

  @override
  void initState() {
    super.initState();
    load();
  }

  Future<void> load() async {
    setState(() {
      loading = true;
      error = null;
    });
    try {
      users = await _adminRepo.users();
    } catch (e) {
      error = '$e';
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final customers = users.where((item) => item['role'] == 'user').length;
    final admins = users.where((item) => item['role'] != 'user').length;
    final locked = users.where((item) => item['is_locked'] == true).length;
    final verified = users.where((item) => item['is_verified'] == true).length;

    return AsyncPage(
      loading: loading,
      error: error,
      onRetry: load,
      child: RefreshIndicator(
        onRefresh: load,
        child: ListView(
          children: [
            PageTitle(
              'Chào ${widget.user.fullName}',
              subtitle: widget.user.role == 'super_admin'
                  ? 'Trung tâm điều hành Super Admin'
                  : 'Trung tâm điều hành quản trị viên',
              trailing: IconButton(
                onPressed: load,
                icon: const Icon(Icons.refresh_rounded),
              ),
            ),
            const SizedBox(height: 24),
            LayoutBuilder(
              builder: (context, constraints) {
                final columns = constraints.maxWidth >= 900
                    ? 4
                    : constraints.maxWidth >= 520
                    ? 2
                    : 1;
                final width =
                    (constraints.maxWidth - (columns - 1) * 14) / columns;
                return Wrap(
                  spacing: 14,
                  runSpacing: 14,
                  children: [
                    _MetricCard(
                      width: width,
                      label: 'Khách hàng',
                      value: '$customers',
                      icon: Icons.people_alt_outlined,
                      color: const Color(0xFF6C7CFF),
                    ),
                    _MetricCard(
                      width: width,
                      label: 'Quản trị viên',
                      value: '$admins',
                      icon: Icons.admin_panel_settings_outlined,
                      color: const Color(0xFF9B6CFF),
                    ),
                    _MetricCard(
                      width: width,
                      label: 'Đã xác minh',
                      value: '$verified',
                      icon: Icons.verified_user_outlined,
                      color: const Color(0xFF25A77A),
                    ),
                    _MetricCard(
                      width: width,
                      label: 'Đang bị khóa',
                      value: '$locked',
                      icon: Icons.lock_outline_rounded,
                      color: const Color(0xFFE06868),
                    ),
                  ],
                );
              },
            ),
            const SizedBox(height: 28),
            const PageTitle(
              'Tài khoản mới',
              subtitle: 'Những người dùng được tạo gần đây.',
            ),
            const SizedBox(height: 14),
            SurfaceCard(
              padding: EdgeInsets.zero,
              child: users.isEmpty
                  ? const EmptyState(
                      icon: Icons.people_outline,
                      title: 'Chưa có người dùng',
                      message: 'Dữ liệu người dùng sẽ xuất hiện tại đây.',
                    )
                  : Column(
                      children: users
                          .take(6)
                          .map(
                            (item) => ListTile(
                              contentPadding: const EdgeInsets.symmetric(
                                horizontal: 18,
                                vertical: 4,
                              ),
                              leading: _UserAvatar(item: item),
                              title: Text(
                                item['full_name']?.toString() ?? 'Người dùng',
                              ),
                              subtitle: Text(
                                '${item['phone'] ?? ''} · ${item['role'] ?? ''}',
                              ),
                              trailing: item['is_locked'] == true
                                  ? const Icon(
                                      Icons.lock_rounded,
                                      color: Colors.redAccent,
                                    )
                                  : const Icon(
                                      Icons.check_circle_outline,
                                      color: Color(0xFF68D391),
                                    ),
                            ),
                          )
                          .toList(),
                    ),
            ),
          ],
        ),
      ),
    );
  }
}

class AdminUsersPage extends StatefulWidget {
  const AdminUsersPage({super.key});

  @override
  State<AdminUsersPage> createState() => _AdminUsersPageState();
}

class _AdminUsersPageState extends State<AdminUsersPage> {
  bool loading = true;
  String? error;
  String query = '';
  String role = 'all';
  List<Map<String, dynamic>> users = [];

  @override
  void initState() {
    super.initState();
    load();
  }

  Future<void> load() async {
    setState(() => loading = true);
    try {
      users = await _adminRepo.users();
      error = null;
    } catch (e) {
      error = '$e';
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  Future<void> toggleLock(Map<String, dynamic> user) async {
    final currentlyLocked = user['is_locked'] == true;
    try {
      await _adminRepo.setLocked(
        (user['id'] as num).toInt(),
        locked: !currentlyLocked,
      );
      if (mounted) {
        showMessage(
          context,
          currentlyLocked ? 'Đã mở khóa tài khoản' : 'Đã khóa tài khoản',
        );
      }
      await load();
    } on ApiException catch (e) {
      if (mounted) showMessage(context, e.message, error: true);
    }
  }

  Future<void> showDetail(Map<String, dynamic> summary) async {
    final id = (summary['id'] as num).toInt();
    try {
      final results = await Future.wait([
        _adminRepo.user(id),
        _adminRepo.userAccounts(id),
      ]);
      if (!mounted) return;
      final user = results[0] as Map<String, dynamic>;
      final accounts = results[1] as List<Map<String, dynamic>>;
      await showDialog<void>(
        context: context,
        builder: (context) => AlertDialog(
          title: Row(
            children: [
              _UserAvatar(item: user),
              const SizedBox(width: 12),
              Expanded(child: Text(user['full_name']?.toString() ?? '')),
            ],
          ),
          content: SizedBox(
            width: 640,
            child: SingleChildScrollView(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  _InfoRow('Điện thoại', user['phone']),
                  _InfoRow('Vai trò', user['role']),
                  _InfoRow(
                    'Trạng thái',
                    user['is_locked'] == true ? 'Đã khóa' : 'Đang hoạt động',
                  ),
                  _InfoRow(
                    'Xác minh',
                    user['is_verified'] == true
                        ? 'Đã xác minh'
                        : 'Chưa xác minh',
                  ),
                  const Divider(height: 30),
                  Row(
                    children: [
                      Text(
                        'Tài khoản ngân hàng',
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      const Spacer(),
                      TextButton.icon(
                        onPressed: () {
                          Navigator.pop(context);
                          showCreateAccount(id);
                        },
                        icon: const Icon(Icons.add_rounded),
                        label: const Text('Cấp tài khoản'),
                      ),
                    ],
                  ),
                  if (accounts.isEmpty)
                    const Padding(
                      padding: EdgeInsets.symmetric(vertical: 20),
                      child: Text(
                        'Chưa có tài khoản',
                        style: TextStyle(color: Colors.white54),
                      ),
                    )
                  else
                    for (final account in accounts)
                      ListTile(
                        contentPadding: EdgeInsets.zero,
                        leading: const CircleAvatar(
                          child: Icon(Icons.account_balance_wallet_outlined),
                        ),
                        title: Text(
                          '${account['account_type']} · ${account['account_number']}',
                        ),
                        subtitle: Text(
                          money(account['balance'], account['currency']),
                        ),
                        trailing: IconButton(
                          tooltip: 'Xem giao dịch',
                          icon: const Icon(Icons.receipt_long_outlined),
                          onPressed: () =>
                              showTransactions((account['id'] as num).toInt()),
                        ),
                      ),
                ],
              ),
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('Đóng'),
            ),
          ],
        ),
      );
    } on ApiException catch (e) {
      if (mounted) showMessage(context, e.message, error: true);
    }
  }

  Future<void> showCreateAccount(int userId) async {
    String type = 'PAYMENT';
    String currency = 'VND';
    await showDialog<void>(
      context: context,
      builder: (context) => StatefulBuilder(
        builder: (context, setDialogState) => AlertDialog(
          title: const Text('Cấp tài khoản mới'),
          content: SizedBox(
            width: 420,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                DropdownButtonFormField<String>(
                  initialValue: type,
                  decoration: fieldDecoration('Loại tài khoản'),
                  items: const [
                    DropdownMenuItem(
                      value: 'PAYMENT',
                      child: Text('Thanh toán'),
                    ),
                    DropdownMenuItem(
                      value: 'SAVINGS',
                      child: Text('Tiết kiệm'),
                    ),
                    DropdownMenuItem(value: 'CREDIT', child: Text('Tín dụng')),
                  ],
                  onChanged: (value) =>
                      setDialogState(() => type = value ?? type),
                ),
                const SizedBox(height: 12),
                DropdownButtonFormField<String>(
                  initialValue: currency,
                  decoration: fieldDecoration('Tiền tệ'),
                  items: const [
                    DropdownMenuItem(value: 'VND', child: Text('VND')),
                    DropdownMenuItem(value: 'USD', child: Text('USD')),
                  ],
                  onChanged: (value) =>
                      setDialogState(() => currency = value ?? currency),
                ),
              ],
            ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('Hủy'),
            ),
            FilledButton(
              onPressed: () async {
                try {
                  await _adminRepo.createUserAccount(userId, type, currency);
                  if (context.mounted) Navigator.pop(context);
                  if (mounted) {
                    showMessage(this.context, 'Đã cấp tài khoản');
                  }
                } on ApiException catch (e) {
                  if (context.mounted) {
                    showMessage(context, e.message, error: true);
                  }
                }
              },
              child: const Text('Xác nhận'),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> showTransactions(int accountId) async {
    try {
      final transactions = await _adminRepo.accountTransactions(accountId);
      if (!mounted) return;
      showDialog<void>(
        context: context,
        builder: (context) => AlertDialog(
          title: const Text('Lịch sử tài khoản'),
          content: SizedBox(
            width: 620,
            height: 420,
            child: transactions.isEmpty
                ? const EmptyState(
                    icon: Icons.receipt_long_outlined,
                    title: 'Chưa có giao dịch',
                    message: 'Tài khoản chưa phát sinh giao dịch.',
                  )
                : ListView.builder(
                    itemCount: transactions.length,
                    itemBuilder: (context, index) {
                      final tx = transactions[index];
                      return ListTile(
                        leading: const CircleAvatar(
                          child: Icon(Icons.swap_horiz_rounded),
                        ),
                        title: Text(
                          tx['description']?.toString().isNotEmpty == true
                              ? tx['description'].toString()
                              : tx['type']?.toString() ?? 'Giao dịch',
                        ),
                        subtitle: Text(tx['reference_code']?.toString() ?? ''),
                        trailing: Text(
                          money(tx['amount'], tx['currency']),
                          style: const TextStyle(fontWeight: FontWeight.w700),
                        ),
                      );
                    },
                  ),
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('Đóng'),
            ),
          ],
        ),
      );
    } on ApiException catch (e) {
      if (mounted) showMessage(context, e.message, error: true);
    }
  }

  @override
  Widget build(BuildContext context) {
    final filtered = users.where((item) {
      final text = '${item['full_name']} ${item['phone']}'.toLowerCase();
      return text.contains(query.toLowerCase()) &&
          (role == 'all' || item['role'] == role);
    }).toList();

    return AsyncPage(
      loading: loading,
      error: error,
      onRetry: load,
      child: Column(
        children: [
          PageTitle(
            'Quản lý người dùng',
            subtitle: '${filtered.length}/${users.length} tài khoản',
            trailing: IconButton(
              onPressed: load,
              icon: const Icon(Icons.refresh_rounded),
            ),
          ),
          const SizedBox(height: 18),
          Row(
            children: [
              Expanded(
                child: TextField(
                  onChanged: (value) => setState(() => query = value),
                  decoration: fieldDecoration(
                    'Tìm tên hoặc số điện thoại',
                  ).copyWith(prefixIcon: const Icon(Icons.search_rounded)),
                ),
              ),
              const SizedBox(width: 12),
              SizedBox(
                width: 150,
                child: DropdownButtonFormField<String>(
                  initialValue: role,
                  decoration: fieldDecoration('Vai trò'),
                  items: const [
                    DropdownMenuItem(value: 'all', child: Text('Tất cả')),
                    DropdownMenuItem(value: 'user', child: Text('User')),
                    DropdownMenuItem(value: 'admin', child: Text('Admin')),
                    DropdownMenuItem(
                      value: 'super_admin',
                      child: Text('Super Admin'),
                    ),
                  ],
                  onChanged: (value) => setState(() => role = value ?? 'all'),
                ),
              ),
            ],
          ),
          const SizedBox(height: 14),
          Expanded(
            child: SurfaceCard(
              padding: EdgeInsets.zero,
              child: filtered.isEmpty
                  ? const EmptyState(
                      icon: Icons.person_search_outlined,
                      title: 'Không có kết quả',
                      message: 'Thử thay đổi từ khóa hoặc bộ lọc.',
                    )
                  : ListView.separated(
                      itemCount: filtered.length,
                      separatorBuilder: (_, _) => const Divider(height: 1),
                      itemBuilder: (context, index) {
                        final user = filtered[index];
                        final isAdmin = user['role'] != 'user';
                        return ListTile(
                          onTap: () => showDetail(user),
                          contentPadding: const EdgeInsets.symmetric(
                            horizontal: 18,
                            vertical: 5,
                          ),
                          leading: _UserAvatar(item: user),
                          title: Text(
                            user['full_name']?.toString() ?? 'Người dùng',
                          ),
                          subtitle: Text(
                            '${user['phone'] ?? ''} · ${user['role'] ?? ''}',
                          ),
                          trailing: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              if (user['is_locked'] == true)
                                const Padding(
                                  padding: EdgeInsets.only(right: 6),
                                  child: Icon(
                                    Icons.lock_rounded,
                                    color: Colors.redAccent,
                                    size: 20,
                                  ),
                                ),
                              IconButton(
                                tooltip: user['is_locked'] == true
                                    ? 'Mở khóa'
                                    : 'Khóa tài khoản',
                                onPressed: isAdmin
                                    ? null
                                    : () => toggleLock(user),
                                icon: Icon(
                                  user['is_locked'] == true
                                      ? Icons.lock_open_rounded
                                      : Icons.lock_outline_rounded,
                                ),
                              ),
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

class AdminOperationsPage extends StatelessWidget {
  const AdminOperationsPage({super.key, required this.user});

  final SessionUser user;

  @override
  Widget build(BuildContext context) => ListView(
    children: [
      const PageTitle(
        'Nghiệp vụ quản trị',
        subtitle: 'Thực hiện các tác vụ tài chính và phân quyền.',
      ),
      const SizedBox(height: 20),
      LayoutBuilder(
        builder: (context, constraints) {
          final width = constraints.maxWidth >= 760
              ? (constraints.maxWidth - 16) / 2
              : constraints.maxWidth;
          return Wrap(
            spacing: 16,
            runSpacing: 16,
            children: [
              SizedBox(width: width, child: const _DepositCard()),
              if (user.role == 'super_admin')
                SizedBox(width: width, child: const _CreateAdminCard()),
            ],
          );
        },
      ),
    ],
  );
}

class _DepositCard extends StatefulWidget {
  const _DepositCard();

  @override
  State<_DepositCard> createState() => _DepositCardState();
}

class _DepositCardState extends State<_DepositCard> {
  final form = GlobalKey<FormState>();
  final account = TextEditingController();
  final amount = TextEditingController();
  final description = TextEditingController();
  bool loading = false;

  Future<void> submit() async {
    if (!form.currentState!.validate()) return;
    setState(() => loading = true);
    try {
      final result = await _adminRepo.deposit(
        accountNumber: account.text.trim(),
        amount: int.parse(amount.text.replaceAll('.', '')),
        description: description.text.trim(),
      );
      if (mounted) {
        showMessage(
          context,
          'Nạp tiền thành công · ${result['reference_code'] ?? ''}',
        );
        form.currentState!.reset();
        account.clear();
        amount.clear();
        description.clear();
      }
    } on ApiException catch (e) {
      if (mounted) showMessage(context, e.message, error: true);
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  @override
  Widget build(BuildContext context) => SurfaceCard(
    child: Form(
      key: form,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const Row(
            children: [
              CircleAvatar(child: Icon(Icons.add_card_rounded)),
              SizedBox(width: 12),
              Text(
                'Nạp tiền khách hàng',
                style: TextStyle(fontSize: 18, fontWeight: FontWeight.w800),
              ),
            ],
          ),
          const SizedBox(height: 20),
          TextFormField(
            controller: account,
            decoration: fieldDecoration('Số tài khoản nhận'),
            validator: _required,
          ),
          const SizedBox(height: 12),
          TextFormField(
            controller: amount,
            keyboardType: TextInputType.number,
            decoration: fieldDecoration('Số tiền (VND)'),
            validator: (value) =>
                (int.tryParse(value?.replaceAll('.', '') ?? '') ?? 0) > 0
                ? null
                : 'Số tiền phải lớn hơn 0',
          ),
          const SizedBox(height: 12),
          TextField(
            controller: description,
            decoration: fieldDecoration('Nội dung'),
          ),
          const SizedBox(height: 18),
          FilledButton.icon(
            onPressed: loading ? null : submit,
            icon: const Icon(Icons.payments_outlined),
            label: const Text('Xác nhận nạp tiền'),
          ),
        ],
      ),
    ),
  );
}

class _CreateAdminCard extends StatefulWidget {
  const _CreateAdminCard();

  @override
  State<_CreateAdminCard> createState() => _CreateAdminCardState();
}

class _CreateAdminCardState extends State<_CreateAdminCard> {
  final form = GlobalKey<FormState>();
  final name = TextEditingController();
  final phone = TextEditingController();
  final password = TextEditingController();
  bool loading = false;

  Future<void> submit() async {
    if (!form.currentState!.validate()) return;
    setState(() => loading = true);
    try {
      final result = await _adminRepo.createAdmin(
        fullName: name.text.trim(),
        phone: phone.text.trim(),
        password: password.text,
      );
      if (!mounted) return;
      await showDialog<void>(
        context: context,
        barrierDismissible: false,
        builder: (context) => AlertDialog(
          icon: const Icon(
            Icons.verified_user_rounded,
            size: 48,
            color: Color(0xFF68D391),
          ),
          title: const Text('Tạo Admin thành công'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Text(
                'Hãy lưu TOTP secret và bàn giao bằng kênh bảo mật.',
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 16),
              SelectableText(
                result['totp_secret']?.toString() ?? '—',
                style: const TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w800,
                  letterSpacing: 1.4,
                ),
              ),
            ],
          ),
          actions: [
            FilledButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('Đã lưu'),
            ),
          ],
        ),
      );
      form.currentState!.reset();
      name.clear();
      phone.clear();
      password.clear();
    } on ApiException catch (e) {
      if (mounted) showMessage(context, e.message, error: true);
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  @override
  Widget build(BuildContext context) => SurfaceCard(
    child: Form(
      key: form,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const Row(
            children: [
              CircleAvatar(child: Icon(Icons.admin_panel_settings_outlined)),
              SizedBox(width: 12),
              Text(
                'Tạo Admin mới',
                style: TextStyle(fontSize: 18, fontWeight: FontWeight.w800),
              ),
            ],
          ),
          const SizedBox(height: 20),
          TextFormField(
            controller: name,
            decoration: fieldDecoration('Họ và tên'),
            validator: _required,
          ),
          const SizedBox(height: 12),
          TextFormField(
            controller: phone,
            decoration: fieldDecoration('Số điện thoại'),
            validator: _required,
          ),
          const SizedBox(height: 12),
          TextFormField(
            controller: password,
            obscureText: true,
            decoration: fieldDecoration('Mật khẩu tạm thời'),
            validator: (value) =>
                (value?.length ?? 0) >= 8 ? null : 'Tối thiểu 8 ký tự',
          ),
          const SizedBox(height: 18),
          FilledButton.icon(
            onPressed: loading ? null : submit,
            icon: const Icon(Icons.person_add_alt_1_rounded),
            label: const Text('Tạo tài khoản Admin'),
          ),
        ],
      ),
    ),
  );
}

class _MetricCard extends StatelessWidget {
  const _MetricCard({
    required this.width,
    required this.label,
    required this.value,
    required this.icon,
    required this.color,
  });

  final double width;
  final String label;
  final String value;
  final IconData icon;
  final Color color;

  @override
  Widget build(BuildContext context) => SizedBox(
    width: width,
    child: SurfaceCard(
      child: Row(
        children: [
          CircleAvatar(
            radius: 25,
            backgroundColor: color.withValues(alpha: .18),
            foregroundColor: color,
            child: Icon(icon),
          ),
          const SizedBox(width: 14),
          Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                value,
                style: const TextStyle(
                  fontSize: 25,
                  fontWeight: FontWeight.w900,
                ),
              ),
              Text(label, style: const TextStyle(color: Colors.white60)),
            ],
          ),
        ],
      ),
    ),
  );
}

class _UserAvatar extends StatelessWidget {
  const _UserAvatar({required this.item});

  final Map<String, dynamic> item;

  @override
  Widget build(BuildContext context) {
    final name = item['full_name']?.toString().trim() ?? '';
    return CircleAvatar(
      child: Text(name.isEmpty ? '?' : name.characters.first.toUpperCase()),
    );
  }
}

class _InfoRow extends StatelessWidget {
  const _InfoRow(this.label, this.value);

  final String label;
  final dynamic value;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 7),
    child: Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 125,
          child: Text(label, style: const TextStyle(color: Colors.white54)),
        ),
        Expanded(child: SelectableText(value?.toString() ?? '—')),
      ],
    ),
  );
}

String? _required(String? value) =>
    value == null || value.trim().isEmpty ? 'Không được để trống' : null;
