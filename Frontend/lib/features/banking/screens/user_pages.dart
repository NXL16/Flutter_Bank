import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';

import '../../../core/network/api_service.dart';
import '../../../core/services/bank_repository.dart';
import '../../../core/storage/token_storage.dart';
import '../../../shared/utils/formatters.dart';
import '../../../shared/widgets/common.dart';
import '../../auth/services/auth_service.dart';
import '../../auth/services/phone_auth_service.dart';
import '../../transfers/screens/transfer_page.dart'
    show showTransactionPINSheet;
import '../../transfers/services/transfer_repository.dart';

const _repo = BankRepository();

class _PasswordChangeDraft {
  const _PasswordChangeDraft(this.oldPassword, this.newPassword);

  final String oldPassword;
  final String newPassword;
}

String _maskPhone(String phone) {
  final digits = phone.replaceAll(RegExp(r'\D'), '');
  if (digits.length < 6) return phone;
  return '${digits.substring(0, 3)}••••${digits.substring(digits.length - 3)}';
}

class DashboardPage extends StatefulWidget {
  const DashboardPage({super.key, this.onOpenSavings});

  final VoidCallback? onOpenSavings;

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
          ),
          const SizedBox(height: 24),
          LayoutBuilder(
            builder: (context, constraints) {
              final width = constraints.maxWidth >= 700
                  ? (constraints.maxWidth - 16) / 2
                  : constraints.maxWidth;
              final savingsAccounts = accounts
                  .where(
                    (account) =>
                        account['account_type']?.toString() == 'SAVINGS',
                  )
                  .toList();
              final regularAccounts = accounts
                  .where(
                    (account) =>
                        account['account_type']?.toString() != 'SAVINGS',
                  )
                  .toList();
              final savingsBalance = savingsAccounts.fold<int>(
                0,
                (total, account) =>
                    total + ((account['balance'] as num?)?.toInt() ?? 0),
              );

              return Wrap(
                spacing: 16,
                runSpacing: 16,
                children: [
                  for (final account in regularAccounts)
                    SizedBox(
                      width: width,
                      child: _OverviewAccountCard(
                        title: account['account_type']?.toString() == 'CREDIT'
                            ? 'TÍN DỤNG'
                            : 'THANH TOÁN',
                        balance: money(
                          account['balance'],
                          account['currency']?.toString() ?? 'VND',
                        ),
                        subtitle: account['account_number']?.toString() ?? '—',
                        icon: Icons.contactless_rounded,
                        colors: const [Color(0xFF4D5CE5), Color(0xFF252951)],
                      ),
                    ),
                  if (savingsAccounts.isNotEmpty)
                    SizedBox(
                      width: width,
                      child: _OverviewAccountCard(
                        title: 'TIẾT KIỆM',
                        balance: money(savingsBalance),
                        subtitle:
                            '${savingsAccounts.length} sổ tiết kiệm đang hoạt động',
                        actionLabel: 'Xem danh sách sổ',
                        icon: Icons.savings_rounded,
                        colors: const [Color(0xFF078A91), Color(0xFF16304C)],
                        onTap: widget.onOpenSavings,
                      ),
                    ),
                ],
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

class _OverviewAccountCard extends StatelessWidget {
  const _OverviewAccountCard({
    required this.title,
    required this.balance,
    required this.subtitle,
    required this.icon,
    required this.colors,
    this.actionLabel,
    this.onTap,
  });

  final String title;
  final String balance;
  final String subtitle;
  final IconData icon;
  final List<Color> colors;
  final String? actionLabel;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) => Material(
    color: Colors.transparent,
    borderRadius: BorderRadius.circular(24),
    clipBehavior: Clip.antiAlias,
    child: InkWell(
      onTap: onTap,
      child: Ink(
        height: 190,
        padding: const EdgeInsets.all(24),
        decoration: BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
            colors: colors,
          ),
          borderRadius: BorderRadius.circular(24),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Text(
                  title,
                  style: const TextStyle(
                    color: Colors.white70,
                    fontWeight: FontWeight.w700,
                    letterSpacing: 1.1,
                  ),
                ),
                const Spacer(),
                Container(
                  width: 38,
                  height: 38,
                  decoration: BoxDecoration(
                    color: Colors.white.withValues(alpha: .12),
                    shape: BoxShape.circle,
                  ),
                  child: Icon(icon, size: 21),
                ),
              ],
            ),
            const Spacer(),
            Text(
              balance,
              style: const TextStyle(fontSize: 28, fontWeight: FontWeight.w800),
            ),
            const SizedBox(height: 10),
            Row(
              children: [
                Expanded(
                  child: Text(
                    actionLabel ?? subtitle,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(color: Colors.white70, fontSize: 13),
                  ),
                ),
                if (actionLabel != null) ...[
                  const SizedBox(width: 8),
                  const Icon(
                    Icons.arrow_forward_rounded,
                    size: 18,
                    color: Colors.white70,
                  ),
                ],
              ],
            ),
            if (actionLabel != null) ...[
              const SizedBox(height: 3),
              Text(
                subtitle,
                style: TextStyle(
                  color: Colors.white.withValues(alpha: .48),
                  fontSize: 11,
                ),
              ),
            ],
          ],
        ),
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
  bool loading = false;

  Future<void> submit() async {
    if (!form.currentState!.validate()) return;
    setState(() => loading = true);
    try {
      final result = await _repo.transfer(
        accountNumber: account.text.trim(),
        amount: int.parse(amount.text.replaceAll('.', '')),
        description: description.text.trim(),
        idToken: '', // Firebase tạm bỏ qua khi test
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
                    decoration: fieldDecoration('Nội dung chuyển tiền'),
                  ),
                  const SizedBox(height: 20),
                  FilledButton.icon(
                    onPressed: loading ? null : submit,
                    icon: const Icon(Icons.send_rounded),
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
                // _DetailRow('Tài khoản gửi', data['sender_account_id']),
                // _DetailRow('Tài khoản nhận', data['receiver_account_id']),
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
  final amount = TextEditingController();
  final pinRepository = const TransferRepository();
  bool initialLoading = true;
  bool loading = false;
  bool showOpenForm = false;
  bool? hasTransactionPIN;
  Map<String, dynamic>? sourceAccount;
  List<Map<String, dynamic>> products = [];
  List<Map<String, dynamic>> savingsAccounts = [];
  int termMonths = 12;
  String maturityInstruction = 'PAYOUT';
  String? savingsRequestID;

  int get parsedAmount =>
      int.tryParse(amount.text.replaceAll(RegExp(r'[^0-9]'), '')) ?? 0;
  Map<String, dynamic>? get selectedProduct {
    for (final product in products) {
      if ((product['term_months'] as num?)?.toInt() == termMonths) {
        return product;
      }
    }
    return null;
  }

  double get interestRate =>
      (selectedProduct?['interest_rate'] as num?)?.toDouble() ?? 0;
  int get minimumAmount =>
      (selectedProduct?['minimum_amount'] as num?)?.toInt() ?? 5000000;
  int get expectedInterest =>
      (parsedAmount * interestRate / 100 * termMonths / 12).round();

  @override
  void initState() {
    super.initState();
    loadContext(initializeView: true);
  }

  @override
  void dispose() {
    amount.dispose();
    super.dispose();
  }

  Future<void> loadContext({bool initializeView = false}) async {
    try {
      final responses = await Future.wait([
        _repo.accounts(),
        pinRepository.hasTransactionPin(),
        _repo.savingsProducts(),
        _repo.savingsAccounts(),
      ]);
      final accounts = responses[0] as List<Map<String, dynamic>>;
      final loadedProducts = responses[2] as List<Map<String, dynamic>>;
      final loadedSavings = responses[3] as List<Map<String, dynamic>>;
      Map<String, dynamic>? paymentAccount;
      for (final account in accounts) {
        if (account['account_type']?.toString() == 'PAYMENT') {
          paymentAccount = account;
          break;
        }
      }
      if (!mounted) return;
      setState(() {
        sourceAccount = paymentAccount;
        hasTransactionPIN = responses[1] as bool;
        products = loadedProducts;
        savingsAccounts = loadedSavings;
        if (loadedProducts.isNotEmpty &&
            !loadedProducts.any(
              (item) => (item['term_months'] as num?)?.toInt() == termMonths,
            )) {
          termMonths =
              (loadedProducts.first['term_months'] as num?)?.toInt() ?? 12;
        }
        if (initializeView) showOpenForm = loadedSavings.isEmpty;
        initialLoading = false;
      });
    } on ApiException catch (error) {
      if (!mounted) return;
      setState(() => initialLoading = false);
      showMessage(context, error.message, error: true);
    }
  }

  Future<void> review() async {
    final value = parsedAmount;
    if (value < minimumAmount) {
      showMessage(
        context,
        'Số tiền tối thiểu là ${money(minimumAmount)}',
        error: true,
      );
      return;
    }
    final balance = (sourceAccount?['balance'] as num?)?.toInt();
    if (balance != null && value > balance) {
      showMessage(
        context,
        'Số dư tài khoản thanh toán không đủ',
        error: true,
        transaction: true,
      );
      return;
    }

    final confirmed = await showModalBottomSheet<bool>(
      context: context,
      useSafeArea: true,
      isScrollControlled: true,
      backgroundColor: Colors.transparent,
      builder: (_) => _SavingsReviewSheet(
        amount: value,
        expectedInterest: expectedInterest,
        sourceAccount: sourceAccount,
        termMonths: termMonths,
        interestRate: interestRate,
        maturityInstruction: maturityInstruction,
      ),
    );
    if (confirmed == true && mounted) {
      savingsRequestID ??= TransferRepository.createIdempotencyKey();
      await authorizeAndOpen(value);
    }
  }

  Future<void> authorizeAndOpen(int value) async {
    bool hasPIN;
    try {
      hasPIN = hasTransactionPIN ?? await pinRepository.hasTransactionPin();
    } on ApiException catch (error) {
      if (mounted) {
        showMessage(context, error.message, error: true, transaction: true);
      }
      return;
    }

    if (!mounted) return;
    final pinInput = await showTransactionPINSheet(
      context,
      createPIN: !hasPIN,
      actionLabel: 'mở sổ tiết kiệm',
    );
    if (pinInput == null || !mounted) return;

    setState(() => loading = true);
    try {
      if (!hasPIN) {
        await pinRepository.setupTransactionPin(
          pinInput.pin,
          pinInput.confirmPIN!,
        );
        hasTransactionPIN = true;
      }
      await _repo.openSavings(
        amount: value,
        termMonths: termMonths,
        maturityInstruction: maturityInstruction,
        transactionPin: pinInput.pin,
        idempotencyKey: savingsRequestID ??=
            TransferRepository.createIdempotencyKey(),
      );
      if (mounted) {
        amount.clear();
        savingsRequestID = null;
        await loadContext();
        if (!mounted) return;
        setState(() => showOpenForm = false);
        showMessage(context, 'Mở sổ tiết kiệm thành công', transaction: true);
      }
    } on ApiException catch (e) {
      if (mounted) {
        showMessage(context, e.message, error: true, transaction: true);
      }
    } finally {
      if (mounted) setState(() => loading = false);
    }
  }

  void showSavingsDetail(Map<String, dynamic> item) {
    showModalBottomSheet<void>(
      context: context,
      useSafeArea: true,
      isScrollControlled: true,
      backgroundColor: Colors.transparent,
      builder: (context) => _SavingsDetailSheet(item: item),
    );
  }

  @override
  Widget build(BuildContext context) {
    if (initialLoading) {
      return const Center(child: CircularProgressIndicator());
    }
    final totalPrincipal = savingsAccounts.fold<int>(
      0,
      (total, item) =>
          total + ((item['original_principal'] as num?)?.toInt() ?? 0),
    );
    return ListView(
      children: [
        PageTitle(
          'Tiết kiệm trực tuyến',
          subtitle: showOpenForm
              ? 'Chọn kỳ hạn và kiểm tra quyền lợi trước khi xác nhận.'
              : '${savingsAccounts.length} sổ đang quản lý',
          trailing: savingsAccounts.isNotEmpty && !showOpenForm
              ? FilledButton.icon(
                  onPressed: () => setState(() => showOpenForm = true),
                  icon: const Icon(Icons.add_rounded),
                  label: const Text('Mở sổ mới'),
                )
              : null,
        ),
        const SizedBox(height: 16),
        Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 680),
            child: showOpenForm
                ? _buildOpenForm()
                : _buildSavingsList(totalPrincipal),
          ),
        ),
      ],
    );
  }

  Widget _buildSavingsList(int totalPrincipal) => Column(
    children: [
      Container(
        width: double.infinity,
        padding: const EdgeInsets.all(20),
        decoration: BoxDecoration(
          gradient: const LinearGradient(
            colors: [Color(0xFF5259D9), Color(0xFF26345A)],
          ),
          borderRadius: BorderRadius.circular(18),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'Tổng tiền gửi',
              style: TextStyle(color: Colors.white70, fontSize: 13),
            ),
            const SizedBox(height: 5),
            Text(
              money(totalPrincipal),
              style: const TextStyle(fontSize: 27, fontWeight: FontWeight.w900),
            ),
            const SizedBox(height: 12),
            Text(
              '${savingsAccounts.length} khoản tiền gửi có kỳ hạn',
              style: const TextStyle(color: Colors.white70, fontSize: 12),
            ),
          ],
        ),
      ),
      const SizedBox(height: 12),
      for (final item in savingsAccounts) ...[
        _SavingsAccountCard(item: item, onTap: () => showSavingsDetail(item)),
        const SizedBox(height: 10),
      ],
    ],
  );

  Widget _buildOpenForm() => Column(
    children: [
      Row(
        children: [
          const Expanded(
            child: Text(
              'Mở sổ tiết kiệm mới',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.w900),
            ),
          ),
          if (savingsAccounts.isNotEmpty)
            IconButton(
              tooltip: 'Đóng',
              onPressed: () => setState(() => showOpenForm = false),
              icon: const Icon(Icons.close_rounded),
            ),
        ],
      ),
      const SizedBox(height: 10),
      Container(
        width: double.infinity,
        padding: const EdgeInsets.all(20),
        decoration: BoxDecoration(
          gradient: const LinearGradient(
            colors: [Color(0xFF5259D9), Color(0xFF26345A)],
          ),
          borderRadius: BorderRadius.circular(18),
        ),
        child: Column(
          children: [
            Row(
              children: [
                const Icon(Icons.savings_rounded, size: 26),
                const SizedBox(width: 10),
                const Expanded(
                  child: Text(
                    'Tiết kiệm An tâm',
                    style: TextStyle(fontSize: 19, fontWeight: FontWeight.w800),
                  ),
                ),
                Text(
                  '$interestRate%/năm',
                  style: const TextStyle(
                    color: Color(0xFFB7BBFF),
                    fontWeight: FontWeight.w900,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 14),
            Text(
              sourceAccount == null
                  ? 'Nguồn tiền: Tài khoản thanh toán'
                  : 'Nguồn tiền: ${sourceAccount!['account_number']} · ${money(sourceAccount!['balance'])}',
              style: const TextStyle(color: Colors.white70, fontSize: 12),
            ),
          ],
        ),
      ),
      const SizedBox(height: 12),
      SurfaceCard(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            DropdownButtonFormField<int>(
              initialValue: termMonths,
              decoration: fieldDecoration('Kỳ hạn'),
              items: products
                  .map(
                    (product) => DropdownMenuItem<int>(
                      value: (product['term_months'] as num).toInt(),
                      child: Text(
                        '${product['term_months']} tháng · ${product['interest_rate']}%/năm',
                      ),
                    ),
                  )
                  .toList(),
              onChanged: (value) {
                if (value != null) {
                  setState(() {
                    termMonths = value;
                    savingsRequestID = null;
                  });
                }
              },
            ),
            const SizedBox(height: 12),
            TextField(
              controller: amount,
              keyboardType: TextInputType.number,
              inputFormatters: [CurrencyInputFormatter()],
              onChanged: (_) => setState(() => savingsRequestID = null),
              decoration: fieldDecoration(
                'Số tiền gửi',
              ).copyWith(suffixText: 'VND'),
            ),
            if (parsedAmount > 0) ...[
              const SizedBox(height: 6),
              Text(
                moneyInVietnameseWords(parsedAmount),
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(color: Colors.white54, fontSize: 12),
              ),
            ],
            const SizedBox(height: 10),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [5000000, 10000000, 20000000, 50000000]
                  .map(
                    (value) => ActionChip(
                      label: Text(money(value).replaceAll(' VND', '')),
                      onPressed: () {
                        amount.text = formatCurrencyInput(value);
                        setState(() => savingsRequestID = null);
                      },
                    ),
                  )
                  .toList(),
            ),
            const SizedBox(height: 12),
            DropdownButtonFormField<String>(
              initialValue: maturityInstruction,
              decoration: fieldDecoration('Khi đến hạn'),
              items: const [
                DropdownMenuItem(
                  value: 'PAYOUT',
                  child: Text('Tất toán về tài khoản thanh toán'),
                ),
                DropdownMenuItem(
                  value: 'RENEW_PRINCIPAL',
                  child: Text('Tái tục tiền gốc'),
                ),
              ],
              onChanged: (value) {
                if (value != null) {
                  setState(() {
                    maturityInstruction = value;
                    savingsRequestID = null;
                  });
                }
              },
            ),
            const SizedBox(height: 14),
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: const Color(0xFF11192B),
                borderRadius: BorderRadius.circular(12),
              ),
              child: Column(
                children: [
                  _DetailRow('Tiền lãi dự kiến', money(expectedInterest)),
                  _DetailRow(
                    'Nhận khi đáo hạn',
                    money(parsedAmount + expectedInterest),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 15),
            FilledButton(
              onPressed: loading ? null : review,
              child: Text(loading ? 'Đang xử lý...' : 'Tiếp tục'),
            ),
          ],
        ),
      ),
    ],
  );
}

String _maturityInstructionLabel(String value) =>
    value == 'RENEW_PRINCIPAL' ? 'Tái tục tiền gốc' : 'Tất toán về tài khoản';

class _SavingsAccountCard extends StatelessWidget {
  const _SavingsAccountCard({required this.item, required this.onTap});

  final Map<String, dynamic> item;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) => SurfaceCard(
    padding: EdgeInsets.zero,
    child: InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(16),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            Container(
              width: 44,
              height: 44,
              decoration: BoxDecoration(
                color: const Color(0xFF7C83FD).withValues(alpha: .14),
                borderRadius: BorderRadius.circular(13),
              ),
              child: const Icon(
                Icons.savings_rounded,
                color: Color(0xFF9EA4FF),
              ),
            ),
            const SizedBox(width: 13),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    money(item['original_principal']),
                    style: const TextStyle(
                      fontSize: 17,
                      fontWeight: FontWeight.w900,
                    ),
                  ),
                  const SizedBox(height: 3),
                  Text(
                    '${item['term_months']} tháng · ${item['interest_rate']}%/năm',
                    style: const TextStyle(
                      color: Color(0xFF9BA8C7),
                      fontSize: 12,
                    ),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    'Đáo hạn ${shortDate(item['end_date'])}',
                    style: const TextStyle(color: Colors.white54, fontSize: 11),
                  ),
                ],
              ),
            ),
            Column(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 8,
                    vertical: 4,
                  ),
                  decoration: BoxDecoration(
                    color: const Color(0xFF68D391).withValues(alpha: .12),
                    borderRadius: BorderRadius.circular(20),
                  ),
                  child: const Text(
                    'Đang hoạt động',
                    style: TextStyle(
                      color: Color(0xFF68D391),
                      fontSize: 10,
                      fontWeight: FontWeight.w800,
                    ),
                  ),
                ),
                const SizedBox(height: 8),
                const Icon(Icons.chevron_right_rounded, color: Colors.white38),
              ],
            ),
          ],
        ),
      ),
    ),
  );
}

class _SavingsDetailSheet extends StatelessWidget {
  const _SavingsDetailSheet({required this.item});

  final Map<String, dynamic> item;

  @override
  Widget build(BuildContext context) => Container(
    decoration: const BoxDecoration(
      color: Color(0xFF151D31),
      borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
    ),
    child: SafeArea(
      top: false,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(20, 10, 20, 18),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 42,
              height: 3,
              decoration: BoxDecoration(
                color: Colors.white24,
                borderRadius: BorderRadius.circular(20),
              ),
            ),
            const SizedBox(height: 18),
            const Text(
              'Chi tiết sổ tiết kiệm',
              style: TextStyle(fontSize: 20, fontWeight: FontWeight.w900),
            ),
            const SizedBox(height: 16),
            _DetailRow('Số tài khoản', item['account_number']),
            _DetailRow('Tiền gốc', money(item['original_principal'])),
            _DetailRow('Kỳ hạn', '${item['term_months']} tháng'),
            _DetailRow('Lãi suất', '${item['interest_rate']}%/năm'),
            _DetailRow('Ngày mở', shortDate(item['start_date'])),
            _DetailRow('Ngày đáo hạn', shortDate(item['end_date'])),
            _DetailRow('Tiền lãi dự kiến', money(item['expected_interest'])),
            _DetailRow('Tổng tiền đáo hạn', money(item['maturity_amount'])),
            _DetailRow(
              'Chỉ thị đáo hạn',
              _maturityInstructionLabel(
                item['maturity_instruction']?.toString() ?? 'PAYOUT',
              ),
            ),
            const SizedBox(height: 16),
            SizedBox(
              width: double.infinity,
              child: FilledButton(
                onPressed: () => Navigator.pop(context),
                child: const Text('Đóng'),
              ),
            ),
          ],
        ),
      ),
    ),
  );
}

class _SavingsReviewSheet extends StatelessWidget {
  const _SavingsReviewSheet({
    required this.amount,
    required this.expectedInterest,
    required this.sourceAccount,
    required this.termMonths,
    required this.interestRate,
    required this.maturityInstruction,
  });

  final int amount;
  final int expectedInterest;
  final Map<String, dynamic>? sourceAccount;
  final int termMonths;
  final double interestRate;
  final String maturityInstruction;

  @override
  Widget build(BuildContext context) => Container(
    decoration: const BoxDecoration(
      color: Color(0xFF151D31),
      borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
    ),
    child: SafeArea(
      top: false,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(20, 10, 20, 18),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 42,
              height: 3,
              decoration: BoxDecoration(
                color: Colors.white24,
                borderRadius: BorderRadius.circular(20),
              ),
            ),
            const SizedBox(height: 18),
            const Text(
              'Xác nhận mở sổ',
              style: TextStyle(fontSize: 20, fontWeight: FontWeight.w900),
            ),
            const SizedBox(height: 18),
            _DetailRow(
              'Tài khoản nguồn',
              sourceAccount?['account_number'] ?? 'PAYMENT',
            ),
            _DetailRow('Số tiền gửi', money(amount)),
            _DetailRow('Kỳ hạn', '$termMonths tháng'),
            _DetailRow('Lãi suất', '$interestRate%/năm'),
            _DetailRow('Tiền lãi dự kiến', money(expectedInterest)),
            _DetailRow('Tổng tiền đáo hạn', money(amount + expectedInterest)),
            _DetailRow(
              'Khi đến hạn',
              _maturityInstructionLabel(maturityInstruction),
            ),
            const SizedBox(height: 10),
            const Text(
              'Tiền sẽ được trích ngay sau khi bạn xác thực mã PIN giao dịch.',
              textAlign: TextAlign.center,
              style: TextStyle(color: Color(0xFF9BA8C7), fontSize: 12),
            ),
            const SizedBox(height: 18),
            Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: () => Navigator.pop(context, false),
                    child: const Text('Quay lại'),
                  ),
                ),
                const SizedBox(width: 10),
                Expanded(
                  flex: 2,
                  child: FilledButton(
                    onPressed: () => Navigator.pop(context, true),
                    child: const Text('Xác nhận'),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    ),
  );
}

class NotificationsPage extends StatefulWidget {
  const NotificationsPage({super.key, this.onUnreadCountChanged});

  final ValueChanged<int>? onUnreadCountChanged;

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
      widget.onUnreadCountChanged?.call(
        items.where((item) => item['is_read'] != true).length,
      );
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
  const ProfilePage({super.key, required this.onLogout, this.onAvatarChanged});

  final Future<void> Function() onLogout;
  final ValueChanged<String>? onAvatarChanged;

  @override
  State<ProfilePage> createState() => _ProfilePageState();
}

class _ProfilePageState extends State<ProfilePage> {
  bool loading = true;
  String? error;
  Map<String, dynamic> profile = {};
  final address = TextEditingController();
  final date = TextEditingController();
  final imagePicker = ImagePicker();
  String avatarUrl = '';
  Uint8List? selectedAvatarBytes;
  XFile? selectedAvatarFile;
  bool pickingAvatar = false;
  bool savingProfile = false;
  String gender = '';

  ImageProvider<Object>? get avatarImage {
    final bytes = selectedAvatarBytes;
    if (bytes != null) return MemoryImage(bytes);
    if (avatarUrl.isNotEmpty) return NetworkImage(avatarUrl);
    return null;
  }

  @override
  void initState() {
    super.initState();
    load();
  }

  Future<void> load() async {
    try {
      profile = await _repo.profile();
      address.text = profile['address']?.toString() ?? '';
      avatarUrl = profile['avatar_url']?.toString() ?? '';
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
    if (savingProfile) return;
    setState(() => savingProfile = true);
    try {
      final avatarFile = selectedAvatarFile;
      if (avatarFile != null) {
        final bytes = selectedAvatarBytes ?? await avatarFile.readAsBytes();
        avatarUrl = await _repo.uploadAvatar(
          bytes: bytes,
          filename: avatarFile.name,
        );
        selectedAvatarFile = null;
        widget.onAvatarChanged?.call(avatarUrl);
      }
      await _repo.updateProfile({
        'address': address.text.trim(),
        'gender': gender,
        'date_of_birth': date.text.trim().isEmpty
            ? null
            : '${date.text.trim()}T00:00:00Z',
      });
      if (mounted) {
        showMessage(context, 'Đã cập nhật hồ sơ');
      }
      await load();
    } on ApiException catch (e) {
      if (mounted) showMessage(context, e.message, error: true);
    } finally {
      if (mounted) setState(() => savingProfile = false);
    }
  }

  Future<void> showAvatarSource() async {
    if (pickingAvatar) return;
    final source = await showModalBottomSheet<ImageSource>(
      context: context,
      backgroundColor: Colors.transparent,
      builder: (context) => Container(
        decoration: const BoxDecoration(
          color: Color(0xFF151D31),
          borderRadius: BorderRadius.vertical(top: Radius.circular(22)),
          border: Border(top: BorderSide(color: Color(0x337C83FD))),
        ),
        child: SafeArea(
          top: false,
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 10, 16, 16),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Container(
                  width: 38,
                  height: 3,
                  decoration: BoxDecoration(
                    color: Colors.white24,
                    borderRadius: BorderRadius.circular(10),
                  ),
                ),
                const SizedBox(height: 14),
                const Align(
                  alignment: Alignment.centerLeft,
                  child: Text(
                    'Chọn ảnh đại diện',
                    style: TextStyle(fontSize: 17, fontWeight: FontWeight.w900),
                  ),
                ),
                const SizedBox(height: 8),
                ListTile(
                  contentPadding: const EdgeInsets.symmetric(horizontal: 4),
                  leading: const Icon(
                    Icons.camera_alt_outlined,
                    color: Color(0xFF9EA4FF),
                  ),
                  title: const Text('Chụp ảnh'),
                  subtitle: const Text('Sử dụng camera của thiết bị'),
                  onTap: () => Navigator.pop(context, ImageSource.camera),
                ),
                ListTile(
                  contentPadding: const EdgeInsets.symmetric(horizontal: 4),
                  leading: const Icon(
                    Icons.photo_library_outlined,
                    color: Color(0xFF4FD1C5),
                  ),
                  title: const Text('Chọn từ thư viện'),
                  subtitle: const Text('Sử dụng ảnh có sẵn trên thiết bị'),
                  onTap: () => Navigator.pop(context, ImageSource.gallery),
                ),
              ],
            ),
          ),
        ),
      ),
    );
    if (source != null) await pickAvatar(source);
  }

  Future<void> pickAvatar(ImageSource source) async {
    setState(() => pickingAvatar = true);
    try {
      final file = await imagePicker.pickImage(
        source: source,
        maxWidth: 1200,
        maxHeight: 1200,
        imageQuality: 85,
      );
      if (file == null) return;
      final bytes = await file.readAsBytes();
      if (bytes.lengthInBytes > 8 * 1024 * 1024) {
        if (mounted) {
          showMessage(
            context,
            'Ảnh vượt quá 8 MB, vui lòng chọn ảnh khác',
            error: true,
          );
        }
        return;
      }
      if (!mounted) return;
      setState(() {
        selectedAvatarFile = file;
        selectedAvatarBytes = bytes;
      });
    } catch (_) {
      if (mounted) {
        showMessage(
          context,
          'Không thể mở camera hoặc thư viện ảnh',
          error: true,
        );
      }
    } finally {
      if (mounted) setState(() => pickingAvatar = false);
    }
  }

  Future<void> selectDateOfBirth() async {
    final today = DateUtils.dateOnly(DateTime.now());
    final firstDate = DateTime(1900);
    final current = DateTime.tryParse(date.text);
    final initialDate =
        current != null &&
            !current.isBefore(firstDate) &&
            !current.isAfter(today)
        ? current
        : DateTime(today.year - 18, today.month, today.day);
    final selected = await showDatePicker(
      context: context,
      initialDate: initialDate,
      firstDate: firstDate,
      lastDate: today,
      helpText: 'CHỌN NGÀY SINH',
      cancelText: 'Hủy',
      confirmText: 'Chọn',
      fieldLabelText: 'Ngày sinh',
      builder: (context, child) => Theme(
        data: Theme.of(context).copyWith(
          colorScheme: Theme.of(context).colorScheme.copyWith(
            primary: const Color(0xFF7C83FD),
            surface: const Color(0xFF151D31),
          ),
          dialogTheme: const DialogThemeData(
            backgroundColor: Color(0xFF151D31),
          ),
        ),
        child: child!,
      ),
    );
    if (selected == null || !mounted) return;
    setState(() {
      date.text =
          '${selected.year.toString().padLeft(4, '0')}-'
          '${selected.month.toString().padLeft(2, '0')}-'
          '${selected.day.toString().padLeft(2, '0')}';
    });
  }

  Future<void> changePassword() async {
    final old = TextEditingController();
    final next = TextEditingController();
    final confirm = TextEditingController();
    final draft = await showDialog<_PasswordChangeDraft>(
      context: context,
      builder: (dialogContext) => AlertDialog(
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
              const SizedBox(height: 12),
              TextField(
                controller: confirm,
                obscureText: true,
                decoration: fieldDecoration('Nhập lại mật khẩu mới'),
              ),
              const SizedBox(height: 12),
              const Text(
                'Sau bước này, mã OTP sẽ được gửi tới số điện thoại của bạn.',
                style: TextStyle(color: Color(0xFF9BA8C7), fontSize: 12),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(dialogContext),
            child: const Text('Hủy'),
          ),
          FilledButton(
            onPressed: () {
              if (old.text.isEmpty) {
                showMessage(
                  dialogContext,
                  'Vui lòng nhập mật khẩu hiện tại',
                  error: true,
                );
                return;
              }
              if (next.text.length < 8) {
                showMessage(
                  dialogContext,
                  'Mật khẩu mới phải có ít nhất 8 ký tự',
                  error: true,
                );
                return;
              }
              if (next.text != confirm.text) {
                showMessage(
                  dialogContext,
                  'Mật khẩu nhập lại không khớp',
                  error: true,
                );
                return;
              }
              Navigator.pop(
                dialogContext,
                _PasswordChangeDraft(old.text, next.text),
              );
            },
            child: const Text('Gửi mã OTP'),
          ),
        ],
      ),
    );
    old.dispose();
    next.dispose();
    confirm.dispose();
    if (draft == null || !mounted) return;

    final phone = profile['phone']?.toString() ?? '';
    if (phone.isEmpty) {
      showMessage(
        context,
        'Không tìm thấy số điện thoại của tài khoản',
        error: true,
      );
      return;
    }

    try {
      final phoneAuth = PhoneAuthService();
      final verificationID = await phoneAuth.sendCode(phone);
      if (!mounted) return;

      String idToken;
      if (verificationID.startsWith('AUTO:')) {
        idToken = await phoneAuth.verifyCode(verificationID, '');
      } else {
        final otp = TextEditingController();
        final smsCode = await showDialog<String>(
          context: context,
          barrierDismissible: false,
          builder: (dialogContext) => AlertDialog(
            title: const Text('Xác nhận OTP'),
            content: SizedBox(
              width: 400,
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    'Nhập mã gồm 6 số đã gửi tới ${_maskPhone(phone)}.',
                    style: const TextStyle(color: Color(0xFF9BA8C7)),
                  ),
                  const SizedBox(height: 16),
                  TextField(
                    controller: otp,
                    autofocus: true,
                    keyboardType: TextInputType.number,
                    maxLength: 6,
                    decoration: fieldDecoration('Mã OTP').copyWith(
                      counterText: '',
                      prefixIcon: const Icon(Icons.sms_outlined),
                    ),
                  ),
                ],
              ),
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(dialogContext),
                child: const Text('Hủy'),
              ),
              FilledButton(
                onPressed: () {
                  if (!RegExp(r'^\d{6}$').hasMatch(otp.text.trim())) {
                    showMessage(
                      dialogContext,
                      'Mã OTP phải gồm đúng 6 số',
                      error: true,
                    );
                    return;
                  }
                  Navigator.pop(dialogContext, otp.text.trim());
                },
                child: const Text('Xác nhận'),
              ),
            ],
          ),
        );
        otp.dispose();
        if (smsCode == null || !mounted) return;
        idToken = await phoneAuth.verifyCode(verificationID, smsCode);
      }

      await AuthService().changePassword(
        draft.oldPassword,
        draft.newPassword,
        idToken,
      );
      if (mounted) await widget.onLogout();
    } on ApiException catch (error) {
      if (mounted) showMessage(context, error.message, error: true);
    }
  }

  Future<void> confirmLogout() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => Dialog(
        backgroundColor: Colors.transparent,
        insetPadding: const EdgeInsets.symmetric(horizontal: 28),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 360),
          child: Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              color: const Color(0xFF151D31),
              borderRadius: BorderRadius.circular(16),
              border: Border.all(color: const Color(0x337C83FD)),
            ),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                const Text(
                  'Đăng xuất',
                  style: TextStyle(fontSize: 18, fontWeight: FontWeight.w900),
                ),
                const SizedBox(height: 7),
                const Text(
                  'Bạn có chắc muốn đăng xuất khỏi NF Bank?',
                  style: TextStyle(
                    color: Color(0xFF9BA8C7),
                    fontSize: 13,
                    height: 1.4,
                  ),
                ),
                const SizedBox(height: 20),
                Row(
                  children: [
                    Expanded(
                      child: OutlinedButton(
                        onPressed: () => Navigator.pop(context, false),
                        style: OutlinedButton.styleFrom(
                          minimumSize: const Size.fromHeight(42),
                          foregroundColor: const Color(0xFFB7C0D7),
                          side: const BorderSide(color: Color(0xFF36425D)),
                          shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(11),
                          ),
                        ),
                        child: const Text('Hủy'),
                      ),
                    ),
                    const SizedBox(width: 10),
                    Expanded(
                      child: FilledButton(
                        onPressed: () => Navigator.pop(context, true),
                        style: FilledButton.styleFrom(
                          minimumSize: const Size.fromHeight(42),
                          backgroundColor: const Color(0xFF6D74F7),
                          shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(11),
                          ),
                        ),
                        child: const Text(
                          'Đăng xuất',
                          style: TextStyle(fontWeight: FontWeight.w800),
                        ),
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
    if (confirmed == true) await widget.onLogout();
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
                      GestureDetector(
                        onTap: showAvatarSource,
                        child: Stack(
                          clipBehavior: Clip.none,
                          children: [
                            Container(
                              padding: const EdgeInsets.all(1.5),
                              decoration: BoxDecoration(
                                shape: BoxShape.circle,
                                border: Border.all(
                                  color: const Color(0xFF8F95FF),
                                  width: 1.5,
                                ),
                              ),
                              child: CircleAvatar(
                                radius: 34.5,
                                backgroundImage: avatarImage,
                                child: avatarImage == null
                                    ? const Icon(Icons.person, size: 34)
                                    : null,
                              ),
                            ),
                            Positioned(
                              right: -2,
                              bottom: -2,
                              child: Container(
                                width: 25,
                                height: 25,
                                decoration: BoxDecoration(
                                  color: const Color(0xFF6D74F7),
                                  shape: BoxShape.circle,
                                  border: Border.all(
                                    color: const Color(0xFF151D31),
                                    width: 2,
                                  ),
                                ),
                                child: pickingAvatar
                                    ? const Padding(
                                        padding: EdgeInsets.all(6),
                                        child: CircularProgressIndicator(
                                          strokeWidth: 2,
                                          color: Colors.white,
                                        ),
                                      )
                                    : const Icon(
                                        Icons.camera_alt_rounded,
                                        size: 13,
                                        color: Colors.white,
                                      ),
                              ),
                            ),
                          ],
                        ),
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
                              '${profile['phone'] ?? ''} · ${profile['role'] ?? ''}',
                              style: const TextStyle(color: Colors.white60),
                            ),
                            const SizedBox(height: 3),
                            InkWell(
                              onTap: showAvatarSource,
                              child: const Text(
                                'Đổi ảnh đại diện',
                                style: TextStyle(
                                  color: Color(0xFF9EA4FF),
                                  fontSize: 12,
                                  fontWeight: FontWeight.w700,
                                ),
                              ),
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
                    readOnly: true,
                    showCursor: false,
                    enableInteractiveSelection: false,
                    onTap: selectDateOfBirth,
                    decoration:
                        fieldDecoration(
                          'Ngày sinh',
                          hint: 'Chọn ngày tháng năm sinh',
                        ).copyWith(
                          suffixIcon: const Icon(Icons.calendar_month_rounded),
                        ),
                  ),
                  const SizedBox(height: 20),
                  FilledButton(
                    onPressed: savingProfile ? null : save,
                    child: Text(savingProfile ? 'Đang lưu...' : 'Lưu thay đổi'),
                  ),
                  const SizedBox(height: 10),
                  OutlinedButton.icon(
                    onPressed: confirmLogout,
                    style: OutlinedButton.styleFrom(
                      foregroundColor: const Color(0xFFFF8A9B),
                      side: const BorderSide(color: Color(0x66FF6B7A)),
                    ),
                    icon: const Icon(Icons.logout_rounded),
                    label: const Text('Đăng xuất'),
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
  Widget build(BuildContext context) {
    final incoming = tx['direction'] == 'IN';
    final counterparty = tx['counterparty_name']?.toString() ?? '';
    final color = incoming ? const Color(0xFF68D391) : Colors.white;
    final amount = money(tx['amount'], tx['currency']?.toString() ?? 'VND');

    return ListTile(
      contentPadding: const EdgeInsets.symmetric(horizontal: 18, vertical: 5),
      leading: CircleAvatar(
        backgroundColor: color.withValues(alpha: .12),
        foregroundColor: color,
        child: Icon(
          incoming
              ? Icons.south_west_rounded
              : tx['type'] == 'SAVINGS_DEPOSIT'
              ? Icons.savings_outlined
              : Icons.north_east_rounded,
        ),
      ),
      title: Text(
        counterparty.isNotEmpty
            ? counterparty
            : tx['description']?.toString().isNotEmpty == true
            ? tx['description'].toString()
            : tx['type']?.toString() ?? 'Giao dịch',
      ),
      subtitle: Text(
        '${tx['description'] ?? ''}\n${dateTimeText(tx['created_at'])}',
        maxLines: 2,
        overflow: TextOverflow.ellipsis,
      ),
      trailing: Text(
        '${incoming ? '+' : '-'}$amount',
        style: TextStyle(fontWeight: FontWeight.w900, color: color),
      ),
    );
  }
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
