import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../../core/network/api_service.dart';
import '../../../core/storage/token_storage.dart';
import '../../../shared/utils/formatters.dart';
import '../../../shared/utils/idempotency.dart';
import '../../../shared/widgets/common.dart';
import '../../transfers/models/account_resolution.dart';
import '../../transfers/services/transfer_repository.dart';
import '../models/admin_models.dart';
import '../services/admin_repository.dart';
import '../widgets/admin_step_up.dart';
import '../widgets/admin_widgets.dart';

class AdminOperationsPage extends StatelessWidget {
  const AdminOperationsPage({super.key, required this.user});

  final SessionUser user;

  @override
  Widget build(BuildContext context) => ListView(
    children: [
      const AdminSectionHeader(
        title: 'Vận hành ngân hàng',
        subtitle:
            'Các thao tác tài chính và phân quyền đều được xác nhận và ghi vết.',
      ),
      const SizedBox(height: 14),
      LayoutBuilder(
        builder: (context, constraints) {
          final wide = constraints.maxWidth >= 900;
          final deposit = const _DepositPanel();
          final access = user.role == 'super_admin'
              ? const _CreateAdminPanel()
              : const _PermissionNotice();
          if (!wide) {
            return Column(
              children: [deposit, const SizedBox(height: 14), access],
            );
          }
          return Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Expanded(flex: 3, child: deposit),
              const SizedBox(width: 14),
              Expanded(flex: 2, child: access),
            ],
          );
        },
      ),
    ],
  );
}

class _DepositPanel extends StatefulWidget {
  const _DepositPanel();

  @override
  State<_DepositPanel> createState() => _DepositPanelState();
}

class _DepositPanelState extends State<_DepositPanel> {
  final _form = GlobalKey<FormState>();
  final _account = TextEditingController();
  final _amount = TextEditingController();
  final _description = TextEditingController(text: 'Cap tien tai khoan');
  final _repository = const AdminRepository();
  final _transferRepository = const TransferRepository();

  Timer? _resolveDebounce;
  AccountResolution? _recipient;
  String? _resolutionError;
  bool _resolving = false;
  bool _submitting = false;
  String? _requestFingerprint;
  String? _requestKey;

  int get _parsedAmount =>
      int.tryParse(_amount.text.replaceAll(RegExp(r'[^0-9]'), '')) ?? 0;

  @override
  void dispose() {
    _resolveDebounce?.cancel();
    _account.dispose();
    _amount.dispose();
    _description.dispose();
    super.dispose();
  }

  void _onAccountChanged(String value) {
    _resolveDebounce?.cancel();
    setState(() {
      _recipient = null;
      _resolutionError = null;
      _resolving = false;
    });
    final accountNumber = value.trim();
    if (!RegExp(r'^\d{12}$').hasMatch(accountNumber)) return;
    _resolveDebounce = Timer(
      const Duration(milliseconds: 350),
      () => _resolve(accountNumber),
    );
  }

  Future<void> _showAccountSelector() async {
    final selectedAccountNumber = await showModalBottomSheet<String>(
      context: context,
      useSafeArea: true,
      isScrollControlled: true,
      backgroundColor: Colors.transparent,
      builder: (sheetContext) => _AccountSelectorSheet(repository: _repository),
    );

    if (selectedAccountNumber != null && mounted) {
      _account.text = selectedAccountNumber;
      _onAccountChanged(selectedAccountNumber);
    }
  }

  Future<void> _resolve(String accountNumber) async {
    setState(() {
      _resolving = true;
      _resolutionError = null;
    });
    try {
      final recipient = await _transferRepository.resolveAccount(accountNumber);
      if (!mounted || _account.text.trim() != accountNumber) return;
      setState(() => _recipient = recipient);
    } on ApiException catch (error) {
      if (!mounted || _account.text.trim() != accountNumber) return;
      setState(() => _resolutionError = error.message);
    } finally {
      if (mounted && _account.text.trim() == accountNumber) {
        setState(() => _resolving = false);
      }
    }
  }

  Future<void> _submit() async {
    if (!_form.currentState!.validate() || _submitting) return;
    var recipient = _recipient;
    if (recipient == null) {
      await _resolve(_account.text.trim());
      recipient = _recipient;
    }
    if (!mounted) return;
    if (recipient == null) {
      showMessage(
        context,
        _resolutionError ?? 'Không xác minh được tài khoản nhận',
        error: true,
      );
      return;
    }
    final verifiedRecipient = recipient;

    final description = _description.text.trim();
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        icon: const Icon(Icons.fact_check_outlined, color: Color(0xFFFFB566)),
        title: const Text('Xác nhận cấp tiền'),
        content: SizedBox(
          width: 430,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              AdminInfoRow('Người nhận', verifiedRecipient.accountName),
              AdminInfoRow('Số tài khoản', verifiedRecipient.accountNumber),
              AdminInfoRow('Số tiền', money(_parsedAmount)),
              AdminInfoRow(
                'Nội dung',
                description.isEmpty ? 'Cap tien tai khoan' : description,
              ),
              const SizedBox(height: 12),
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: const Color(0x1FFFAB5E),
                  borderRadius: BorderRadius.circular(10),
                  border: Border.all(color: const Color(0x44FFAB5E)),
                ),
                child: const Text(
                  'Đây là nghiệp vụ tài chính có ghi nhận sổ cái. Hãy kiểm tra đúng khách hàng và số tiền.',
                  textAlign: TextAlign.center,
                  style: TextStyle(color: Color(0xFFFFD19A), fontSize: 12),
                ),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(dialogContext, false),
            child: const Text('Kiểm tra lại'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(dialogContext, true),
            child: const Text('Phê duyệt nạp tiền'),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;

    final stepUpToken = await showAdminStepUp(
      context,
      action: AdminSensitiveAction.deposit,
      binding:
          '${AdminSensitiveAction.deposit}|${_account.text.trim()}|$_parsedAmount|$description',
      title: 'Phê duyệt cấp tiền',
      transactionSummary:
          '${verifiedRecipient.accountName}\n${verifiedRecipient.accountNumber}\n${money(_parsedAmount)}',
    );
    if (stepUpToken == null || !mounted) return;

    final fingerprint = '${_account.text.trim()}|$_parsedAmount|$description';
    if (_requestFingerprint != fingerprint) {
      _requestFingerprint = fingerprint;
      _requestKey = createIdempotencyKey();
    }
    setState(() => _submitting = true);
    try {
      final result = await _repository.deposit(
        accountNumber: _account.text.trim(),
        amount: _parsedAmount,
        description: description,
        idempotencyKey: _requestKey ??= createIdempotencyKey(),
        stepUpToken: stepUpToken,
      );
      if (!mounted) return;
      _requestFingerprint = null;
      _requestKey = null;
      await showDialog<void>(
        context: context,
        builder: (dialogContext) => AlertDialog(
          icon: const Icon(
            Icons.check_circle_rounded,
            color: Color(0xFF50D4A3),
            size: 48,
          ),
          title: const Text('Nạp tiền thành công'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                money(result['amount']),
                style: const TextStyle(
                  fontSize: 25,
                  fontWeight: FontWeight.w900,
                ),
              ),
              const SizedBox(height: 10),
              Text(
                verifiedRecipient.accountName,
                style: const TextStyle(fontWeight: FontWeight.w800),
              ),
              Text(
                verifiedRecipient.accountNumber,
                style: const TextStyle(color: Color(0xFF8792AD)),
              ),
              const SizedBox(height: 14),
              SelectableText(
                result['reference_code']?.toString() ?? '—',
                style: const TextStyle(color: Color(0xFFAEB5FF), fontSize: 12),
              ),
            ],
          ),
          actions: [
            FilledButton(
              onPressed: () => Navigator.pop(dialogContext),
              child: const Text('Hoàn tất'),
            ),
          ],
        ),
      );
      _form.currentState?.reset();
      _account.clear();
      _amount.clear();
      _description.text = 'Cap tien tai khoan';
      setState(() {
        _recipient = null;
        _resolutionError = null;
      });
    } on ApiException catch (error) {
      if (mounted) {
        showMessage(context, error.message, error: true, transaction: true);
      }
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) => AdminPanel(
    child: Form(
      key: _form,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const AdminSectionHeader(
            title: 'Cấp tiền cho khách hàng',
            subtitle:
                'Nguồn đối ứng: NF Bank Operations · Không dùng ví cá nhân Admin',
            trailing: AdminStatusBadge(
              label: 'GHI SỔ KÉP',
              color: Color(0xFF50D4A3),
            ),
          ),
          const SizedBox(height: 18),
          TextFormField(
            controller: _account,
            keyboardType: TextInputType.number,
            maxLength: 12,
            inputFormatters: [FilteringTextInputFormatter.digitsOnly],
            decoration:
                fieldDecoration(
                  'Số tài khoản khách hàng',
                  hint: 'Nhập 12 chữ số',
                ).copyWith(
                  counterText: '',
                  suffixIcon: _resolving
                      ? const Padding(
                          padding: EdgeInsets.all(14),
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : IconButton(
                          icon: const Icon(Icons.manage_search_rounded),
                          tooltip: 'Chọn tài khoản có sẵn',
                          onPressed: _showAccountSelector,
                        ),
                ),
            onChanged: _onAccountChanged,
            validator: (value) =>
                RegExp(r'^\d{12}$').hasMatch(value?.trim() ?? '')
                ? null
                : 'Số tài khoản phải gồm 12 chữ số',
          ),
          if (_recipient != null) ...[
            const SizedBox(height: 8),
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: const Color(0xFF132F32),
                borderRadius: BorderRadius.circular(12),
                border: Border.all(color: const Color(0xFF275C59)),
              ),
              child: Row(
                children: [
                  const Icon(Icons.verified_rounded, color: Color(0xFF50D4A3)),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          _recipient!.accountName,
                          style: const TextStyle(fontWeight: FontWeight.w900),
                        ),
                        Text(
                          '${_recipient!.bankName} · Đã xác minh',
                          style: const TextStyle(
                            color: Color(0xFF76CBB3),
                            fontSize: 11,
                          ),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ] else if (_resolutionError != null) ...[
            const SizedBox(height: 6),
            Text(
              _resolutionError!,
              style: const TextStyle(color: Color(0xFFFF7D8A), fontSize: 11),
            ),
          ],
          const SizedBox(height: 12),
          TextFormField(
            controller: _amount,
            keyboardType: TextInputType.number,
            inputFormatters: [CurrencyInputFormatter()],
            decoration: fieldDecoration('Số tiền').copyWith(suffixText: 'VND'),
            onChanged: (_) => setState(() {}),
            validator: (_) => _parsedAmount >= 10000
                ? null
                : 'Số tiền tối thiểu là 10.000 VND',
          ),
          if (_parsedAmount > 0) ...[
            const SizedBox(height: 6),
            Text(
              moneyInVietnameseWords(_parsedAmount),
              style: const TextStyle(color: Color(0xFF8792AD), fontSize: 11),
            ),
          ],
          const SizedBox(height: 12),
          TextFormField(
            controller: _description,
            maxLength: 140,
            decoration: fieldDecoration(
              'Nội dung nghiệp vụ',
            ).copyWith(counterText: ''),
          ),
          const SizedBox(height: 16),
          FilledButton.icon(
            onPressed: _submitting ? null : _submit,
            icon: _submitting
                ? const SizedBox.square(
                    dimension: 18,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.fact_check_outlined),
            label: Text(
              _submitting ? 'Đang ghi nhận...' : 'Kiểm tra và phê duyệt',
            ),
          ),
        ],
      ),
    ),
  );
}

class _CreateAdminPanel extends StatefulWidget {
  const _CreateAdminPanel();

  @override
  State<_CreateAdminPanel> createState() => _CreateAdminPanelState();
}

class _CreateAdminPanelState extends State<_CreateAdminPanel> {
  final _form = GlobalKey<FormState>();
  final _name = TextEditingController();
  final _phone = TextEditingController();
  final _password = TextEditingController();
  final _repository = const AdminRepository();
  bool _loading = false;
  bool _passwordHidden = true;

  @override
  void dispose() {
    _name.dispose();
    _phone.dispose();
    _password.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (!_form.currentState!.validate() || _loading) return;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        icon: const Icon(
          Icons.admin_panel_settings_outlined,
          color: Color(0xFFFFB566),
        ),
        title: const Text('Tạo quản trị viên mới?'),
        content: Text(
          '${_name.text.trim()} sẽ có quyền vận hành khách hàng và nghiệp vụ tài chính. Tài khoản bắt buộc đăng nhập bằng TOTP.',
          textAlign: TextAlign.center,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(dialogContext, false),
            child: const Text('Hủy'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(dialogContext, true),
            child: const Text('Tạo Admin'),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;

    final stepUpToken = await showAdminStepUp(
      context,
      action: AdminSensitiveAction.createAdmin,
      binding:
          '${AdminSensitiveAction.createAdmin}|${_name.text.trim()}|${_phone.text.trim()}',
      title: 'Xác nhận tạo Admin',
      transactionSummary:
          '${_name.text.trim()}\n${_phone.text.trim()}\nQuyền vận hành hệ thống',
    );
    if (stepUpToken == null || !mounted) return;

    setState(() => _loading = true);
    try {
      final result = await _repository.createAdmin(
        fullName: _name.text.trim(),
        phone: _phone.text.trim(),
        password: _password.text,
        stepUpToken: stepUpToken,
      );
      if (!mounted) return;
      final secret = result['totp_secret']?.toString() ?? '';
      await showDialog<void>(
        context: context,
        barrierDismissible: false,
        builder: (dialogContext) => AlertDialog(
          icon: const Icon(
            Icons.key_rounded,
            color: Color(0xFF50D4A3),
            size: 44,
          ),
          title: const Text('Khóa TOTP chỉ hiển thị một lần'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Text(
                'Sao chép khóa và bàn giao qua kênh bảo mật. Không gửi cùng mật khẩu tạm thời.',
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 14),
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(14),
                decoration: BoxDecoration(
                  color: const Color(0xFF0E1525),
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: const Color(0xFF344263)),
                ),
                child: SelectableText(
                  secret.isEmpty ? '—' : secret,
                  textAlign: TextAlign.center,
                  style: const TextStyle(
                    color: Color(0xFFAEB5FF),
                    fontSize: 17,
                    fontWeight: FontWeight.w900,
                    letterSpacing: 1.3,
                  ),
                ),
              ),
            ],
          ),
          actions: [
            OutlinedButton.icon(
              onPressed: secret.isEmpty
                  ? null
                  : () async {
                      await Clipboard.setData(ClipboardData(text: secret));
                      if (dialogContext.mounted) {
                        showMessage(dialogContext, 'Đã sao chép khóa TOTP');
                      }
                    },
              icon: const Icon(Icons.copy_rounded),
              label: const Text('Sao chép'),
            ),
            FilledButton(
              onPressed: () => Navigator.pop(dialogContext),
              child: const Text('Tôi đã lưu an toàn'),
            ),
          ],
        ),
      );
      _form.currentState?.reset();
      _name.clear();
      _phone.clear();
      _password.clear();
    } on ApiException catch (error) {
      if (mounted) showMessage(context, error.message, error: true);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) => AdminPanel(
    child: Form(
      key: _form,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const AdminSectionHeader(
            title: 'Quản trị truy cập',
            subtitle: 'Chỉ Super Admin có quyền cấp tài khoản quản trị',
            trailing: AdminStatusBadge(
              label: 'SUPER ADMIN',
              color: Color(0xFFC9A3FF),
            ),
          ),
          const SizedBox(height: 18),
          TextFormField(
            controller: _name,
            decoration: fieldDecoration('Họ và tên'),
            validator: (value) {
              final length = value?.trim().length ?? 0;
              return length >= 2 && length <= 100
                  ? null
                  : 'Họ tên phải có từ 2 đến 100 ký tự';
            },
          ),
          const SizedBox(height: 12),
          TextFormField(
            controller: _phone,
            keyboardType: TextInputType.phone,
            decoration: fieldDecoration('Số điện thoại'),
            validator: (value) {
              final digits = (value ?? '').replaceAll(RegExp(r'\D'), '');
              return RegExp(r'^(0|84)?(3|5|7|8|9)\d{8}$').hasMatch(digits)
                  ? null
                  : 'Số điện thoại Việt Nam không hợp lệ';
            },
          ),
          const SizedBox(height: 12),
          TextFormField(
            controller: _password,
            obscureText: _passwordHidden,
            decoration: fieldDecoration('Mật khẩu tạm thời').copyWith(
              suffixIcon: IconButton(
                onPressed: () =>
                    setState(() => _passwordHidden = !_passwordHidden),
                icon: Icon(
                  _passwordHidden
                      ? Icons.visibility_outlined
                      : Icons.visibility_off_outlined,
                ),
              ),
            ),
            validator: (value) =>
                RegExp(
                  r'^(?=.*[A-Z])(?=.*[a-z])(?=.*\d)(?=.*[!@#$%^&*(),.?":{}|<>]).{12,}$',
                ).hasMatch(value ?? '')
                ? null
                : 'Tối thiểu 12 ký tự, có hoa, thường, số và ký tự đặc biệt',
          ),
          const SizedBox(height: 12),
          const Text(
            'Admin mới sẽ dùng mật khẩu tạm thời và TOTP để đăng nhập. Mọi thao tác tạo tài khoản đều được ghi vào nhật ký quản trị.',
            style: TextStyle(
              color: Color(0xFF8792AD),
              fontSize: 11,
              height: 1.45,
            ),
          ),
          const SizedBox(height: 16),
          FilledButton.icon(
            onPressed: _loading ? null : _submit,
            icon: _loading
                ? const SizedBox.square(
                    dimension: 18,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.person_add_alt_1_rounded),
            label: Text(_loading ? 'Đang tạo...' : 'Tạo tài khoản Admin'),
          ),
        ],
      ),
    ),
  );
}

class _PermissionNotice extends StatelessWidget {
  const _PermissionNotice();

  @override
  Widget build(BuildContext context) => const AdminPanel(
    child: Column(
      children: [
        Icon(
          Icons.admin_panel_settings_outlined,
          size: 42,
          color: Color(0xFF8792AD),
        ),
        SizedBox(height: 12),
        Text(
          'Quản trị truy cập',
          style: TextStyle(fontSize: 17, fontWeight: FontWeight.w900),
        ),
        SizedBox(height: 7),
        Text(
          'Chỉ Super Admin mới có thể tạo tài khoản quản trị viên mới.',
          textAlign: TextAlign.center,
          style: TextStyle(color: Color(0xFF8792AD), height: 1.4),
        ),
      ],
    ),
  );
}

class _AccountSelectorSheet extends StatefulWidget {
  const _AccountSelectorSheet({required this.repository});

  final AdminRepository repository;

  @override
  State<_AccountSelectorSheet> createState() => _AccountSelectorSheetState();
}

class _AccountSelectorSheetState extends State<_AccountSelectorSheet> {
  final _search = TextEditingController();
  bool _loading = true;
  String? _error;
  List<AdminUserSummary> _users = const [];
  final Map<int, bool> _fetchingAccounts = {};

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

  Future<void> _load() async {
    try {
      final users = await widget.repository.users();
      final customers = users.where((u) => u.role == 'user').toList();
      if (mounted) {
        setState(() {
          _users = customers;
          _loading = false;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _error = '$e';
          _loading = false;
        });
      }
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

    return _users.where((u) {
      final matchesName = u.fullName.toLowerCase().contains(query);
      final matchesPhone =
          u.phone.toLowerCase().contains(query) ||
          (queryDigits.isNotEmpty &&
              cleanDigits(u.phone).contains(cleanQueryPhone));
      return query.isEmpty || matchesName || matchesPhone;
    }).toList();
  }

  Future<void> _selectUser(AdminUserSummary user) async {
    if (_fetchingAccounts[user.id] == true) return;
    setState(() => _fetchingAccounts[user.id] = true);

    try {
      final accounts = await widget.repository.userAccounts(user.id);
      final paymentAccount = accounts.firstWhere(
        (acc) => acc['account_type'] == 'PAYMENT',
        orElse: () => <String, dynamic>{},
      );

      if (!mounted) return;

      final accNumber = paymentAccount['account_number']?.toString();
      if (accNumber != null && accNumber.isNotEmpty) {
        Navigator.pop(context, accNumber);
      } else {
        showMessage(
          context,
          'Khách hàng này chưa có tài khoản thanh toán',
          error: true,
        );
      }
    } catch (e) {
      if (mounted) {
        showMessage(context, 'Không thể lấy tài khoản: $e', error: true);
      }
    } finally {
      if (mounted) {
        setState(() => _fetchingAccounts[user.id] = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final filtered = _filtered;
    return FractionallySizedBox(
      heightFactor: 0.85,
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
              padding: const EdgeInsets.fromLTRB(20, 14, 8, 10),
              child: Row(
                children: [
                  const Expanded(
                    child: Text(
                      'Chọn tài khoản khách hàng',
                      style: TextStyle(
                        fontSize: 18,
                        fontWeight: FontWeight.w900,
                      ),
                    ),
                  ),
                  IconButton(
                    onPressed: () => Navigator.pop(context),
                    icon: const Icon(Icons.close_rounded),
                  ),
                ],
              ),
            ),
            const Divider(height: 1),
            Padding(
              padding: const EdgeInsets.all(12),
              child: TextField(
                controller: _search,
                onChanged: (_) => setState(() {}),
                decoration: fieldDecoration(
                  'Tìm tên hoặc số điện thoại',
                ).copyWith(prefixIcon: const Icon(Icons.search_rounded)),
              ),
            ),
            Expanded(
              child: _loading
                  ? const Center(child: CircularProgressIndicator())
                  : _error != null
                  ? EmptyState(
                      icon: Icons.error_outline_rounded,
                      title: 'Lỗi tải dữ liệu',
                      message: _error!,
                    )
                  : filtered.isEmpty
                  ? const EmptyState(
                      icon: Icons.person_search_outlined,
                      title: 'Không có kết quả',
                      message: 'Không tìm thấy khách hàng phù hợp.',
                    )
                  : ListView.separated(
                      itemCount: filtered.length,
                      separatorBuilder: (_, _) => const Divider(height: 1),
                      itemBuilder: (context, index) {
                        final user = filtered[index];
                        final fetching = _fetchingAccounts[user.id] == true;
                        return ListTile(
                          onTap: () => _selectUser(user),
                          leading: AdminUserAvatar(user: user, radius: 20),
                          title: Text(user.fullName),
                          subtitle: Text(user.phone),
                          trailing: fetching
                              ? const SizedBox(
                                  width: 18,
                                  height: 18,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                  ),
                                )
                              : const Icon(Icons.chevron_right_rounded),
                        );
                      },
                    ),
            ),
          ],
        ),
      ),
    );
  }
}
