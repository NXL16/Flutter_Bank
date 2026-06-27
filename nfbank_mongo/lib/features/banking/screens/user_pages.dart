import 'package:flutter/material.dart';

import '../../../core/network/api_service.dart';
import '../../../core/services/bank_repository.dart';
import '../../../core/storage/token_storage.dart';
import '../../../shared/utils/formatters.dart';
import '../../../shared/widgets/common.dart';
import '../../auth/services/auth_service.dart';

const _repo = BankRepository();

class DashboardPage extends StatefulWidget {
  const DashboardPage({super.key});

  @override
  State<DashboardPage> createState() => _DashboardPageState();
}

class _DashboardPageState extends State<DashboardPage> {
  bool loading = true;
  String? error;
  List<Map<String, dynamic>> accounts = [];
  List<Map<String, dynamic>> transactions = [];
  SessionUser? user;

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
      final results = await Future.wait([
        _repo.accounts(),
        _repo.transactions(),
        TokenStorage.getUser(),
      ]);
      accounts = results[0] as List<Map<String, dynamic>>;
      transactions = results[1] as List<Map<String, dynamic>>;
      user = results[2] as SessionUser?;
    } catch (e) {
      error = '$e';
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  @override
  Widget build(BuildContext context) => AsyncPage(
    loading: loading,
    error: error,
    onRetry: load,
    child: RefreshIndicator(
      onRefresh: load,
      child: ListView(
        children: [
          PageTitle(
            'Xin chào, ${user?.fullName ?? 'bạn'}',
            subtitle: 'Đây là tình hình tài chính hôm nay.',
            trailing: IconButton(
              onPressed: load,
              icon: const Icon(Icons.refresh),
            ),
          ),
          const SizedBox(height: 24),
          LayoutBuilder(
            builder: (context, constraints) {
              final width = constraints.maxWidth >= 700
                  ? (constraints.maxWidth - 16) / 2
                  : constraints.maxWidth;
              return Wrap(
                spacing: 16,
                runSpacing: 16,
                children: accounts.map((account) {
                  final type = account['account_type']?.toString() ?? '';
                  return SizedBox(
                    width: width,
                    child: Container(
                      height: 190,
                      padding: const EdgeInsets.all(24),
                      decoration: BoxDecoration(
                        gradient: LinearGradient(
                          begin: Alignment.topLeft,
                          end: Alignment.bottomRight,
                          colors: type == 'SAVINGS'
                              ? const [Color(0xFF087E8B), Color(0xFF132B41)]
                              : const [Color(0xFF4D5CE5), Color(0xFF252951)],
                        ),
                        borderRadius: BorderRadius.circular(24),
                      ),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Row(
                            children: [
                              Text(
                                type == 'SAVINGS'
                                    ? 'TIẾT KIỆM'
                                    : type == 'CREDIT'
                                    ? 'TÍN DỤNG'
                                    : 'THANH TOÁN',
                                style: const TextStyle(
                                  color: Colors.white70,
                                  fontWeight: FontWeight.w700,
                                  letterSpacing: 1.1,
                                ),
                              ),
                              const Spacer(),
                              const Icon(Icons.contactless_rounded),
                            ],
                          ),
                          const Spacer(),
                          Text(
                            money(
                              account['balance'],
                              account['currency']?.toString() ?? 'VND',
                            ),
                            style: const TextStyle(
                              fontSize: 28,
                              fontWeight: FontWeight.w800,
                            ),
                          ),
                          const SizedBox(height: 12),
                          Text(
                            account['account_number']?.toString() ?? '—',
                            style: const TextStyle(
                              color: Colors.white70,
                              letterSpacing: 2,
                            ),
                          ),
                        ],
                      ),
                    ),
                  );
                }).toList(),
              );
            },
          ),
          const SizedBox(height: 28),
          const PageTitle('Giao dịch gần đây'),
          const SizedBox(height: 14),
          SurfaceCard(
            padding: EdgeInsets.zero,
            child: transactions.isEmpty
                ? const EmptyState(
                    icon: Icons.receipt_long_outlined,
                    title: 'Chưa có giao dịch',
                    message: 'Các giao dịch mới sẽ xuất hiện tại đây.',
                  )
                : Column(
                    children: transactions
                        .take(5)
                        .map((tx) => TransactionTile(tx: tx))
                        .toList(),
                  ),
          ),
        ],
      ),
    ),
  );
}

class AccountsPage extends StatefulWidget {
  const AccountsPage({super.key});

  @override
  State<AccountsPage> createState() => _AccountsPageState();
}

class _AccountsPageState extends State<AccountsPage> {
  bool loading = true;
  List<Map<String, dynamic>> accounts = [];
  String? error;

  @override
  void initState() {
    super.initState();
    load();
  }

  Future<void> load() async {
    try {
      accounts = await _repo.accounts();
      error = null;
    } catch (e) {
      error = '$e';
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  @override
  Widget build(BuildContext context) => AsyncPage(
    loading: loading,
    error: error,
    child: ListView(
      children: [
        const PageTitle(
          'Tài khoản của tôi',
          subtitle: 'Theo dõi số dư và trạng thái từng tài khoản.',
        ),
        const SizedBox(height: 20),
        for (final account in accounts) ...[
          SurfaceCard(
            child: Row(
              children: [
                CircleAvatar(
                  radius: 24,
                  child: Icon(
                    account['account_type'] == 'SAVINGS'
                        ? Icons.savings_outlined
                        : Icons.account_balance_wallet_outlined,
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        account['account_type']?.toString() ?? 'Tài khoản',
                        style: Theme.of(context).textTheme.titleMedium,
                      ),
                      const SizedBox(height: 4),
                      SelectableText(
                        account['account_number']?.toString() ?? '—',
                        style: const TextStyle(color: Colors.white60),
                      ),
                    ],
                  ),
                ),
                Column(
                  crossAxisAlignment: CrossAxisAlignment.end,
                  children: [
                    Text(
                      money(
                        account['balance'],
                        account['currency']?.toString() ?? 'VND',
                      ),
                      style: const TextStyle(
                        fontWeight: FontWeight.w800,
                        fontSize: 18,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      account['status']?.toString() ?? '',
                      style: const TextStyle(color: Color(0xFF68D391)),
                    ),
                  ],
                ),
              ],
            ),
          ),
          const SizedBox(height: 12),
        ],
      ],
    ),
  );
}

class TransferPage extends StatefulWidget {
  const TransferPage({super.key});

  @override
  State<TransferPage> createState() => _TransferPageState();
}

class _TransferPageState extends State<TransferPage> {
  final form = GlobalKey<FormState>();
  final account = TextEditingController();
  final amount = TextEditingController();
  final description = TextEditingController();
  final idToken = TextEditingController();
  bool loading = false;

  Future<void> submit() async {
    if (!form.currentState!.validate()) return;
    setState(() => loading = true);
    try {
      final result = await _repo.transfer(
        accountNumber: account.text.trim(),
        amount: int.parse(amount.text.replaceAll('.', '')),
        description: description.text.trim(),
        idToken: idToken.text.trim(),
      );
      if (mounted) {
        showDialog<void>(
          context: context,
          builder: (_) => AlertDialog(
            icon: const Icon(
              Icons.check_circle_rounded,
              size: 54,
              color: Color(0xFF68D391),
            ),
            title: const Text('Chuyển tiền thành công'),
            content: Text(
              'Mã giao dịch: ${result['reference_code'] ?? '—'}\n'
              'Số tiền: ${money(result['amount'])}',
              textAlign: TextAlign.center,
            ),
            actions: [
              FilledButton(
                onPressed: () => Navigator.pop(context),
                child: const Text('Hoàn tất'),
              ),
            ],
          ),
        );
        form.currentState!.reset();
        account.clear();
        amount.clear();
        description.clear();
        idToken.clear();
      }
    } on ApiException catch (e) {
      if (mounted) showMessage(context, e.message, error: true);
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  @override
  Widget build(BuildContext context) => ListView(
    children: [
      const PageTitle(
        'Chuyển tiền',
        subtitle: 'Chuyển khoản nội bộ NF Bank theo số tài khoản.',
      ),
      const SizedBox(height: 20),
      Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 620),
          child: SurfaceCard(
            child: Form(
              key: form,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  TextFormField(
                    controller: account,
                    decoration: fieldDecoration('Số tài khoản người nhận'),
                    validator: _required,
                  ),
                  const SizedBox(height: 14),
                  TextFormField(
                    controller: amount,
                    keyboardType: TextInputType.number,
                    decoration: fieldDecoration('Số tiền (VND)'),
                    validator: (value) =>
                        (int.tryParse(value?.replaceAll('.', '') ?? '') ?? 0) >
                            0
                        ? null
                        : 'Số tiền phải lớn hơn 0',
                  ),
                  const SizedBox(height: 14),
                  TextField(
                    controller: description,
                    decoration: fieldDecoration('Nội dung'),
                  ),
                  const SizedBox(height: 14),
                  TextFormField(
                    controller: idToken,
                    minLines: 2,
                    maxLines: 4,
                    decoration: fieldDecoration(
                      'Firebase Phone ID token',
                      hint: 'Token xác thực số điện thoại cho giao dịch',
                    ),
                    validator: _required,
                  ),
                  const SizedBox(height: 8),
                  const Text(
                    'Backend bắt buộc Firebase ID token khớp số điện thoại tài khoản. '
                    'Repo hiện chưa chứa cấu hình Firebase client để tự sinh token.',
                    style: TextStyle(color: Colors.amberAccent, fontSize: 12),
                  ),
                  const SizedBox(height: 20),
                  FilledButton.icon(
                    onPressed: loading ? null : submit,
                    icon: const Icon(Icons.lock_outline),
                    label: const Text('Xác nhận chuyển tiền'),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    ],
  );

  String? _required(String? value) =>
      value == null || value.trim().isEmpty ? 'Không được để trống' : null;
}

class TransactionsPage extends StatefulWidget {
  const TransactionsPage({super.key});

  @override
  State<TransactionsPage> createState() => _TransactionsPageState();
}

class _TransactionsPageState extends State<TransactionsPage> {
  bool loading = true;
  String? error;
  List<Map<String, dynamic>> items = [];
  String query = '';

  @override
  void initState() {
    super.initState();
    load();
  }

  Future<void> load() async {
    try {
      items = await _repo.transactions();
      error = null;
    } catch (e) {
      error = '$e';
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  Future<void> detail(Map<String, dynamic> tx) async {
    try {
      final data = await _repo.transaction(
        tx['reference_code']?.toString() ?? '',
      );
      if (!mounted) return;
      showDialog<void>(
        context: context,
        builder: (context) => AlertDialog(
          title: const Text('Chi tiết giao dịch'),
          content: SizedBox(
            width: 430,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                _DetailRow('Mã giao dịch', data['reference_code']),
                _DetailRow('Loại', data['type']),
                _DetailRow('Số tiền', money(data['amount'], data['currency'])),
                _DetailRow('Trạng thái', data['status']),
                _DetailRow('Nội dung', data['description']),
                _DetailRow('Tài khoản gửi', data['sender_account_id']),
                _DetailRow('Tài khoản nhận', data['receiver_account_id']),
              ],
            ),
          ),
          actions: [
            FilledButton(
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
    final filtered = items.where((tx) {
      final haystack =
          '${tx['reference_code']} ${tx['description']} ${tx['type']}'
              .toLowerCase();
      return haystack.contains(query.toLowerCase());
    }).toList();
    return AsyncPage(
      loading: loading,
      error: error,
      child: ListView(
        children: [
          const PageTitle(
            'Lịch sử giao dịch',
            subtitle: 'Tra cứu và xem chi tiết mọi biến động.',
          ),
          const SizedBox(height: 18),
          TextField(
            onChanged: (value) => setState(() => query = value),
            decoration: fieldDecoration(
              'Tìm theo mã hoặc nội dung',
            ).copyWith(prefixIcon: const Icon(Icons.search)),
          ),
          const SizedBox(height: 16),
          SurfaceCard(
            padding: EdgeInsets.zero,
            child: filtered.isEmpty
                ? const EmptyState(
                    icon: Icons.search_off,
                    title: 'Không có kết quả',
                    message: 'Hãy thử một từ khóa khác.',
                  )
                : Column(
                    children: filtered
                        .map(
                          (tx) => InkWell(
                            onTap: () => detail(tx),
                            child: TransactionTile(tx: tx),
                          ),
                        )
                        .toList(),
                  ),
          ),
        ],
      ),
    );
  }
}

class SavingsPage extends StatefulWidget {
  const SavingsPage({super.key});

  @override
  State<SavingsPage> createState() => _SavingsPageState();
}

class _SavingsPageState extends State<SavingsPage> {
  final amount = TextEditingController(text: '5000000');
  bool loading = false;
  Map<String, dynamic>? result;

  Future<void> open() async {
    final value = int.tryParse(amount.text.replaceAll('.', '')) ?? 0;
    if (value < 5000000) {
      showMessage(context, 'Số tiền tối thiểu là 5.000.000 VND', error: true);
      return;
    }
    setState(() => loading = true);
    try {
      result = await _repo.openSavings(value);
      if (mounted) {
        setState(() {});
        showMessage(context, 'Mở sổ tiết kiệm thành công');
      }
    } on ApiException catch (e) {
      if (mounted) showMessage(context, e.message, error: true);
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  @override
  Widget build(BuildContext context) => ListView(
    children: [
      const PageTitle(
        'Tiết kiệm trực tuyến',
        subtitle: 'Kỳ hạn 12 tháng · Lãi suất 8,5%/năm.',
      ),
      const SizedBox(height: 22),
      Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 650),
          child: Column(
            children: [
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(28),
                decoration: BoxDecoration(
                  gradient: const LinearGradient(
                    colors: [Color(0xFF087E8B), Color(0xFF153C50)],
                  ),
                  borderRadius: BorderRadius.circular(24),
                ),
                child: const Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Icon(Icons.savings_rounded, size: 38),
                    SizedBox(height: 24),
                    Text(
                      'Tích lũy tương lai',
                      style: TextStyle(
                        fontSize: 24,
                        fontWeight: FontWeight.w800,
                      ),
                    ),
                    SizedBox(height: 8),
                    Text(
                      'Tiền được trích từ tài khoản PAYMENT. Mỗi khách hàng được mở một sổ.',
                      style: TextStyle(color: Colors.white70),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 16),
              SurfaceCard(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    TextField(
                      controller: amount,
                      keyboardType: TextInputType.number,
                      decoration: fieldDecoration('Số tiền gửi (VND)'),
                    ),
                    const SizedBox(height: 18),
                    FilledButton(
                      onPressed: loading ? null : open,
                      child: const Text('Mở sổ tiết kiệm'),
                    ),
                  ],
                ),
              ),
              if (result != null) ...[
                const SizedBox(height: 16),
                SurfaceCard(
                  child: Column(
                    children: [
                      _DetailRow('Số tài khoản', result!['account_number']),
                      _DetailRow(
                        'Tiền gốc',
                        money(result!['original_principal']),
                      ),
                      _DetailRow(
                        'Lãi suất',
                        '${result!['interest_rate']}%/năm',
                      ),
                      _DetailRow(
                        'Ngày đáo hạn',
                        shortDate(result!['end_date']),
                      ),
                    ],
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
    ],
  );
}

class NotificationsPage extends StatefulWidget {
  const NotificationsPage({super.key});

  @override
  State<NotificationsPage> createState() => _NotificationsPageState();
}

class _NotificationsPageState extends State<NotificationsPage> {
  bool loading = true;
  String? error;
  List<Map<String, dynamic>> items = [];

  @override
  void initState() {
    super.initState();
    load();
  }

  Future<void> load() async {
    try {
      items = await _repo.notifications();
      error = null;
    } catch (e) {
      error = '$e';
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  Future<void> markAll() async {
    try {
      await _repo.markAllNotificationsRead();
      await load();
    } on ApiException catch (e) {
      if (mounted) showMessage(context, e.message, error: true);
    }
  }

  @override
  Widget build(BuildContext context) => AsyncPage(
    loading: loading,
    error: error,
    child: ListView(
      children: [
        PageTitle(
          'Thông báo',
          subtitle:
              '${items.where((item) => item['is_read'] != true).length} chưa đọc',
          trailing: TextButton.icon(
            onPressed: markAll,
            icon: const Icon(Icons.done_all),
            label: const Text('Đọc tất cả'),
          ),
        ),
        const SizedBox(height: 18),
        if (items.isEmpty)
          const EmptyState(
            icon: Icons.notifications_none,
            title: 'Chưa có thông báo',
            message: 'Thông báo biến động số dư sẽ xuất hiện tại đây.',
          )
        else
          for (final item in items) ...[
            SurfaceCard(
              child: ListTile(
                contentPadding: EdgeInsets.zero,
                leading: CircleAvatar(
                  backgroundColor: item['is_read'] == true
                      ? Colors.white10
                      : Theme.of(context).colorScheme.primaryContainer,
                  child: const Icon(Icons.notifications_outlined),
                ),
                title: Text(
                  item['title']?.toString() ?? 'Thông báo',
                  style: TextStyle(
                    fontWeight: item['is_read'] == true
                        ? FontWeight.w500
                        : FontWeight.w800,
                  ),
                ),
                subtitle: Padding(
                  padding: const EdgeInsets.only(top: 6),
                  child: Text(
                    '${item['content'] ?? ''}\n${dateTimeText(item['created_at'])}',
                  ),
                ),
                onTap: item['is_read'] == true
                    ? null
                    : () async {
                        await _repo.markNotificationRead(
                          (item['id'] as num).toInt(),
                        );
                        await load();
                      },
              ),
            ),
            const SizedBox(height: 10),
          ],
      ],
    ),
  );
}

class ProfilePage extends StatefulWidget {
  const ProfilePage({super.key});

  @override
  State<ProfilePage> createState() => _ProfilePageState();
}

class _ProfilePageState extends State<ProfilePage> {
  bool loading = true;
  String? error;
  Map<String, dynamic> profile = {};
  final address = TextEditingController();
  final avatar = TextEditingController();
  final date = TextEditingController();
  String gender = '';

  @override
  void initState() {
    super.initState();
    load();
  }

  Future<void> load() async {
    try {
      profile = await _repo.profile();
      address.text = profile['address']?.toString() ?? '';
      avatar.text = profile['avatar_url']?.toString() ?? '';
      date.text = profile['date_of_birth']?.toString().split('T').first ?? '';
      gender = profile['gender']?.toString() ?? '';
      error = null;
    } catch (e) {
      error = '$e';
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  Future<void> save() async {
    try {
      await _repo.updateProfile({
        'address': address.text.trim(),
        'avatar_url': avatar.text.trim(),
        'gender': gender,
        'date_of_birth': date.text.trim().isEmpty
            ? null
            : '${date.text.trim()}T00:00:00Z',
      });
      if (mounted) showMessage(context, 'Đã cập nhật hồ sơ');
      await load();
    } on ApiException catch (e) {
      if (mounted) showMessage(context, e.message, error: true);
    }
  }

  void changePassword() {
    final old = TextEditingController();
    final next = TextEditingController();
    showDialog<void>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Đổi mật khẩu'),
        content: SizedBox(
          width: 400,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: old,
                obscureText: true,
                decoration: fieldDecoration('Mật khẩu hiện tại'),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: next,
                obscureText: true,
                decoration: fieldDecoration('Mật khẩu mới'),
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
                await AuthService().changePassword(old.text, next.text);
                if (context.mounted) Navigator.pop(context);
                if (mounted) showMessage(this.context, 'Đã đổi mật khẩu');
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
    );
  }

  @override
  Widget build(BuildContext context) => AsyncPage(
    loading: loading,
    error: error,
    child: ListView(
      children: [
        PageTitle(
          'Hồ sơ cá nhân',
          subtitle: 'Cập nhật thông tin và bảo mật tài khoản.',
          trailing: OutlinedButton.icon(
            onPressed: changePassword,
            icon: const Icon(Icons.password),
            label: const Text('Đổi mật khẩu'),
          ),
        ),
        const SizedBox(height: 20),
        Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 700),
            child: SurfaceCard(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Row(
                    children: [
                      CircleAvatar(
                        radius: 34,
                        backgroundImage: avatar.text.isNotEmpty
                            ? NetworkImage(avatar.text)
                            : null,
                        child: avatar.text.isEmpty
                            ? const Icon(Icons.person, size: 34)
                            : null,
                      ),
                      const SizedBox(width: 16),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              profile['full_name']?.toString() ?? '',
                              style: Theme.of(context).textTheme.titleLarge,
                            ),
                            Text(
                              '${profile['email'] ?? ''} · ${profile['role'] ?? ''}',
                              style: const TextStyle(color: Colors.white60),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 24),
                  TextField(
                    controller: address,
                    decoration: fieldDecoration('Địa chỉ'),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: avatar,
                    decoration: fieldDecoration('URL ảnh đại diện'),
                  ),
                  const SizedBox(height: 12),
                  DropdownButtonFormField<String>(
                    initialValue: gender.isEmpty ? null : gender,
                    decoration: fieldDecoration('Giới tính'),
                    items: const [
                      DropdownMenuItem(value: 'male', child: Text('Nam')),
                      DropdownMenuItem(value: 'female', child: Text('Nữ')),
                      DropdownMenuItem(value: 'other', child: Text('Khác')),
                    ],
                    onChanged: (value) => setState(() => gender = value ?? ''),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: date,
                    decoration: fieldDecoration(
                      'Ngày sinh',
                      hint: 'YYYY-MM-DD',
                    ),
                  ),
                  const SizedBox(height: 20),
                  FilledButton(
                    onPressed: save,
                    child: const Text('Lưu thay đổi'),
                  ),
                ],
              ),
            ),
          ),
        ),
      ],
    ),
  );
}

class TransactionTile extends StatelessWidget {
  const TransactionTile({super.key, required this.tx});

  final Map<String, dynamic> tx;

  @override
  Widget build(BuildContext context) => ListTile(
    contentPadding: const EdgeInsets.symmetric(horizontal: 18, vertical: 5),
    leading: CircleAvatar(
      child: Icon(
        tx['type'] == 'DEPOSIT'
            ? Icons.south_west_rounded
            : tx['type'] == 'SAVINGS_DEPOSIT'
            ? Icons.savings_outlined
            : Icons.north_east_rounded,
      ),
    ),
    title: Text(
      tx['description']?.toString().isNotEmpty == true
          ? tx['description'].toString()
          : tx['type']?.toString() ?? 'Giao dịch',
    ),
    subtitle: Text(tx['reference_code']?.toString() ?? ''),
    trailing: Text(
      money(tx['amount'], tx['currency']?.toString() ?? 'VND'),
      style: const TextStyle(fontWeight: FontWeight.w800),
    ),
  );
}

class _DetailRow extends StatelessWidget {
  const _DetailRow(this.label, this.value);

  final String label;
  final dynamic value;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 8),
    child: Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 130,
          child: Text(label, style: const TextStyle(color: Colors.white54)),
        ),
        Expanded(child: SelectableText(value?.toString() ?? '—')),
      ],
    ),
  );
}
