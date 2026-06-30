import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../../core/network/api_service.dart';
import '../../../core/services/bank_repository.dart';
import '../../../core/storage/token_storage.dart';
import '../../../shared/utils/formatters.dart';
import '../../../shared/widgets/common.dart';
import '../models/account_resolution.dart';
import '../models/recent_recipient.dart';
import '../models/transfer_receipt.dart';
import '../services/transfer_repository.dart';

enum _TransferStep { details, confirmation, receipt }

class TransferPage extends StatefulWidget {
  const TransferPage({super.key});

  @override
  State<TransferPage> createState() => _TransferPageState();
}

class _TransferPageState extends State<TransferPage> {
  final _form = GlobalKey<FormState>();
  final _account = TextEditingController();
  final _amount = TextEditingController();
  final _description = TextEditingController();
  final _repository = const TransferRepository();
  final _bankRepository = const BankRepository();

  _TransferStep _step = _TransferStep.details;
  SessionUser? _user;
  Map<String, dynamic>? _sourceAccount;
  String _senderAvatarURL = '';
  AccountResolution? _recipient;
  List<RecentRecipient> _recentRecipients = const [];
  TransferReceipt? _receipt;
  String? _idempotencyKey;
  String? _resolutionError;
  String _defaultDescription = 'Chuyen khoan';
  Timer? _resolveDebounce;
  int _resolutionRequest = 0;
  bool _resolving = false;
  bool? _hasTransactionPin;
  bool _loading = false;

  int get _parsedAmount =>
      int.tryParse(_amount.text.replaceAll(RegExp(r'[^0-9]'), '')) ?? 0;

  @override
  void initState() {
    super.initState();
    _loadSourceAccount();
    _loadRecentRecipients();
    _loadDefaultDescription();
    _loadTransactionPINStatus();
  }

  @override
  void dispose() {
    _resolveDebounce?.cancel();
    _account.dispose();
    _amount.dispose();
    _description.dispose();
    super.dispose();
  }

  Future<void> _loadDefaultDescription() async {
    final userName = await TokenStorage.getUserName();
    final normalizedName = removeVietnameseDiacritics(userName);
    final description = normalizedName.isEmpty
        ? 'Chuyen khoan'
        : '$normalizedName chuyen khoan';
    if (!mounted) return;
    _defaultDescription = description;
    if (_description.text.trim().isEmpty) {
      _description.text = description;
    }
  }

  Future<void> _loadSourceAccount() async {
    try {
      final results = await Future.wait([
        TokenStorage.getUser(),
        _bankRepository.accounts(),
        _bankRepository.profile(),
      ]);
      final user = results[0] as SessionUser?;
      final accounts = results[1] as List<Map<String, dynamic>>;
      final profile = results[2] as Map<String, dynamic>;
      Map<String, dynamic>? source;
      for (final account in accounts) {
        if (account['account_type']?.toString() == 'PAYMENT') {
          source = account;
          break;
        }
      }
      source ??= accounts.isEmpty ? null : accounts.first;
      if (mounted) {
        setState(() {
          _user = user;
          _sourceAccount = source;
          _senderAvatarURL = profile['avatar_url']?.toString().trim() ?? '';
        });
      }
    } on ApiException {
      // Người dùng vẫn có thể nhập giao dịch; backend sẽ kiểm tra số dư.
    }
  }

  Future<void> _loadRecentRecipients() async {
    try {
      final transactions = await _bankRepository.transactions();
      if (mounted) {
        setState(() {
          _recentRecipients = recentRecipientsFromTransactions(transactions);
        });
      }
    } on ApiException {
      // Danh sách chuyển nhanh là tiện ích phụ, không chặn giao dịch mới.
    }
  }

  Future<void> _loadTransactionPINStatus() async {
    try {
      final hasPIN = await _repository.hasTransactionPin();
      if (mounted) setState(() => _hasTransactionPin = hasPIN);
    } on ApiException {
      // Trạng thái sẽ được thử lại ngay trước khi xác nhận giao dịch.
    }
  }

  void _onAccountChanged(String value) {
    _resolveDebounce?.cancel();
    final accountNumber = value.trim();
    final request = ++_resolutionRequest;
    setState(() {
      _recipient = null;
      _resolutionError = null;
      _resolving = false;
    });

    if (!RegExp(r'^[0-9]{12}$').hasMatch(accountNumber)) return;
    _resolveDebounce = Timer(
      const Duration(milliseconds: 450),
      () => _resolveRecipient(accountNumber, request),
    );
  }

  Future<AccountResolution?> _resolveRecipient(
    String accountNumber,
    int request, {
    bool showMessageOnError = false,
  }) async {
    if (mounted) {
      setState(() {
        _resolving = true;
        _resolutionError = null;
      });
    }
    try {
      final recipient = await _repository.resolveAccount(accountNumber);
      if (!mounted ||
          request != _resolutionRequest ||
          _account.text.trim() != accountNumber) {
        return null;
      }
      setState(() => _recipient = recipient);
      return recipient;
    } on ApiException catch (error) {
      if (mounted && request == _resolutionRequest) {
        setState(() => _resolutionError = error.message);
        if (showMessageOnError) {
          showMessage(context, error.message, error: true, transaction: true);
        }
      }
      return null;
    } finally {
      if (mounted && request == _resolutionRequest) {
        setState(() => _resolving = false);
      }
    }
  }

  Future<void> _review() async {
    if (!_form.currentState!.validate()) return;
    setState(() => _loading = true);
    try {
      final accountNumber = _account.text.trim();
      var recipient = _recipient;
      if (recipient == null || recipient.accountNumber != accountNumber) {
        recipient = await _resolveRecipient(
          accountNumber,
          ++_resolutionRequest,
          showMessageOnError: true,
        );
      }
      if (recipient == null) return;
      if (!mounted) return;
      setState(() {
        _recipient = recipient;
        _idempotencyKey = TransferRepository.createIdempotencyKey();
        _step = _TransferStep.confirmation;
      });
    } on ApiException catch (error) {
      if (mounted) {
        showMessage(context, error.message, error: true, transaction: true);
      }
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _authorizeAndSubmit() async {
    bool hasPIN;
    try {
      hasPIN = _hasTransactionPin ?? await _repository.hasTransactionPin();
    } on ApiException catch (error) {
      if (mounted) {
        showMessage(context, error.message, error: true, transaction: true);
      }
      return;
    }

    if (!mounted) return;
    final pinResult = await showTransactionPINSheet(
      context,
      createPIN: !hasPIN,
    );
    if (pinResult == null || !mounted) return;

    setState(() => _loading = true);
    try {
      if (!hasPIN) {
        await _repository.setupTransactionPin(
          pinResult.pin,
          pinResult.confirmPIN!,
        );
        _hasTransactionPin = true;
      }
      await _submit(pinResult.pin);
    } on ApiException catch (error) {
      if (mounted) {
        showMessage(context, error.message, error: true, transaction: true);
      }
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _submit(String transactionPIN) async {
    final recipient = _recipient;
    final requestID = _idempotencyKey;
    if (recipient == null || requestID == null) return;

    final receipt = await _repository.transfer(
      accountNumber: recipient.accountNumber,
      amount: _parsedAmount,
      description: _description.text.trim(),
      idempotencyKey: requestID,
      transactionPin: transactionPIN,
    );
    if (!mounted) return;
    final updatedRecent = [
      RecentRecipient(
        accountNumber: recipient.accountNumber,
        accountName: recipient.accountName,
      ),
      ..._recentRecipients.where(
        (item) => item.accountNumber != recipient.accountNumber,
      ),
    ].take(10).toList(growable: false);
    setState(() {
      _receipt = receipt;
      _recentRecipients = updatedRecent;
      _step = _TransferStep.receipt;
    });
  }

  Future<void> _showRecentRecipients() async {
    if (_recentRecipients.isEmpty) return;
    final selected = await showModalBottomSheet<RecentRecipient>(
      context: context,
      backgroundColor: Colors.transparent,
      useSafeArea: true,
      isScrollControlled: true,
      builder: (_) => _RecentRecipientsSheet(recipients: _recentRecipients),
    );
    if (selected == null || !mounted) return;

    _resolveDebounce?.cancel();
    final request = ++_resolutionRequest;
    _account.text = selected.accountNumber;
    setState(() {
      _recipient = null;
      _resolutionError = null;
      _resolving = false;
    });
    await _resolveRecipient(
      selected.accountNumber,
      request,
      showMessageOnError: true,
    );
  }

  void _reset() {
    _account.clear();
    _amount.clear();
    _description.text = _defaultDescription;
    _resolveDebounce?.cancel();
    _resolutionRequest++;
    setState(() {
      _step = _TransferStep.details;
      _recipient = null;
      _receipt = null;
      _idempotencyKey = null;
      _resolutionError = null;
      _resolving = false;
    });
  }

  @override
  Widget build(BuildContext context) => ListView(
    padding: EdgeInsets.zero,
    children: [
      PageTitle(switch (_step) {
        _TransferStep.details => 'Chuyển tiền tới tài khoản',
        _TransferStep.confirmation => 'Xác nhận thông tin',
        _TransferStep.receipt => 'Giao dịch hoàn tất',
      }, subtitle: _subtitle),
      const SizedBox(height: 12),
      Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 640),
          child: AnimatedSwitcher(
            duration: const Duration(milliseconds: 220),
            child: switch (_step) {
              _TransferStep.details => _detailsCard(),
              _TransferStep.confirmation => _confirmationCard(),
              _TransferStep.receipt => _receiptCard(),
            },
          ),
        ),
      ),
    ],
  );

  String get _subtitle => switch (_step) {
    _TransferStep.details => 'Chuyển khoản nội bộ NF Bank.',
    _TransferStep.confirmation => 'Kiểm tra chính xác thông tin giao dịch.',
    _TransferStep.receipt => 'Tiền đã được ghi nhận trong hệ thống NF Bank.',
  };

  Widget _detailsCard() => KeyedSubtree(
    key: const ValueKey('details'),
    child: Form(
      key: _form,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Container(
            padding: const EdgeInsets.fromLTRB(16, 14, 16, 10),
            decoration: _panelDecoration(),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                const _SectionLabel('Nguồn chuyển tiền'),
                const SizedBox(height: 7),
                _SourceAccountCard(account: _sourceAccount),
                Divider(height: 22, color: Colors.white.withValues(alpha: .08)),
                const _SectionLabel('Chuyển đến'),
                const SizedBox(height: 5),
                const _BankSelector(),
                if (_recentRecipients.isNotEmpty)
                  _RecentRecipientsButton(
                    count: _recentRecipients.length,
                    onTap: _showRecentRecipients,
                  ),
                Divider(height: 1, color: Colors.white.withValues(alpha: .08)),
                Padding(
                  padding: const EdgeInsets.only(top: 5),
                  child: TextFormField(
                    controller: _account,
                    keyboardType: TextInputType.number,
                    maxLength: 12,
                    inputFormatters: [FilteringTextInputFormatter.digitsOnly],
                    decoration: const InputDecoration(
                      labelText: 'Số tài khoản',
                      hintText: 'Nhập 12 chữ số',
                      counterText: '',
                      border: InputBorder.none,
                      isDense: true,
                      suffixIcon: Icon(Icons.contact_page_outlined),
                    ),
                    onChanged: _onAccountChanged,
                    validator: (value) =>
                        RegExp(r'^[0-9]{12}$').hasMatch(value?.trim() ?? '')
                        ? null
                        : 'Số tài khoản phải gồm 12 chữ số',
                  ),
                ),
                if (_resolving)
                  const LinearProgressIndicator(minHeight: 2)
                else if (_recipient != null)
                  _RecipientPreview(recipient: _recipient!)
                else if (_resolutionError != null)
                  Padding(
                    padding: const EdgeInsets.only(bottom: 5),
                    child: Align(
                      alignment: Alignment.centerLeft,
                      child: Text(
                        _resolutionError!,
                        style: const TextStyle(color: Color(0xFFFF8A80)),
                      ),
                    ),
                  ),
                Divider(height: 18, color: Colors.white.withValues(alpha: .08)),
                TextFormField(
                  controller: _amount,
                  keyboardType: TextInputType.number,
                  inputFormatters: [CurrencyInputFormatter()],
                  style: const TextStyle(
                    color: Color(0xFF9EA4FF),
                    fontSize: 23,
                    fontWeight: FontWeight.w800,
                  ),
                  decoration: const InputDecoration(
                    labelText: 'Số tiền',
                    hintText: '0',
                    suffixText: 'VND',
                    border: InputBorder.none,
                    isDense: true,
                  ),
                  onChanged: (_) => setState(() {}),
                  validator: (_) => _parsedAmount >= 10000
                      ? null
                      : 'Số tiền tối thiểu là 10.000 VND',
                ),
                if (_parsedAmount > 0)
                  Padding(
                    padding: const EdgeInsets.only(top: 3),
                    child: Text(
                      _amountInWords,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        color: Colors.white54,
                        fontSize: 12,
                      ),
                    ),
                  ),
                Divider(height: 18, color: Colors.white.withValues(alpha: .08)),
                TextFormField(
                  controller: _description,
                  maxLength: 140,
                  decoration: InputDecoration(
                    labelText: 'Nội dung chuyển tiền',
                    hintText: _defaultDescription,
                    counterText: '',
                    border: InputBorder.none,
                    isDense: true,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 12),
          _BottomActions(
            primaryLabel: _loading ? 'Đang xác minh...' : 'Tiếp tục',
            onPrimary: _loading ? null : _review,
            onBack: () => Navigator.maybePop(context),
          ),
        ],
      ),
    ),
  );

  Widget _confirmationCard() {
    final recipient = _recipient!;
    return Column(
      key: const ValueKey('confirmation'),
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Container(
          padding: const EdgeInsets.all(16),
          decoration: _panelDecoration(),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Text(
                'Số tiền giao dịch',
                style: TextStyle(color: Colors.white60, fontSize: 15),
              ),
              const SizedBox(height: 5),
              Text(
                money(_parsedAmount, recipient.currency),
                style: const TextStyle(
                  color: Color(0xFF7C83FD),
                  fontSize: 25,
                  fontWeight: FontWeight.w900,
                ),
              ),
              const SizedBox(height: 3),
              Text(
                _amountInWords,
                style: const TextStyle(color: Colors.white60),
              ),
              const Divider(height: 24),
              const Text(
                'Người chuyển',
                style: TextStyle(color: Colors.white60, fontSize: 15),
              ),
              const SizedBox(height: 8),
              _PartyTile(
                name: _user?.fullName ?? 'Người dùng NF Bank',
                accountNumber:
                    _sourceAccount?['account_number']?.toString() ?? '—',
                bankName: 'NF Bank',
                avatarURL: _senderAvatarURL,
              ),
              const SizedBox(height: 14),
              const Text(
                'Người nhận',
                style: TextStyle(color: Colors.white60, fontSize: 15),
              ),
              const SizedBox(height: 8),
              _PartyTile(
                name: recipient.accountName,
                accountNumber: recipient.accountNumber,
                bankName: recipient.bankName,
                avatarURL: recipient.avatarUrl,
              ),
              const Divider(height: 24),
              _InfoRow(
                'Nội dung chuyển tiền',
                _description.text.trim().isEmpty
                    ? 'Chuyển tiền'
                    : _description.text.trim(),
              ),
              const _InfoRow('Hình thức chuyển tiền', 'Trong NF Bank'),
              Divider(height: 22, color: Colors.white.withValues(alpha: .08)),
              const Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Icon(
                    Icons.warning_amber_rounded,
                    size: 20,
                    color: Color(0xFFFFB547),
                  ),
                  SizedBox(width: 9),
                  Expanded(
                    child: Text(
                      'Kiểm tra chính xác thông tin trước khi xác nhận.',
                      style: TextStyle(
                        fontSize: 12,
                        height: 1.35,
                        color: Color(0xFFFFD58A),
                      ),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
        const SizedBox(height: 12),
        _BottomActions(
          primaryLabel: _loading ? 'Đang xử lý...' : 'Xác nhận',
          onPrimary: _loading ? null : _authorizeAndSubmit,
          onBack: _loading
              ? null
              : () => setState(() => _step = _TransferStep.details),
        ),
      ],
    );
  }

  String get _amountInWords {
    if (_parsedAmount <= 0) return '';
    return moneyInVietnameseWords(_parsedAmount);
  }

  BoxDecoration _panelDecoration() => BoxDecoration(
    color: const Color(0xFF151D31),
    borderRadius: BorderRadius.circular(14),
    border: Border.all(color: Colors.white.withValues(alpha: .08)),
  );

  Widget _receiptCard() {
    final receipt = _receipt!;
    return SurfaceCard(
      key: const ValueKey('receipt'),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const Icon(
            Icons.check_circle_rounded,
            color: Color(0xFF68D391),
            size: 66,
          ),
          const SizedBox(height: 12),
          Text(
            money(receipt.amount, receipt.currency),
            textAlign: TextAlign.center,
            style: const TextStyle(fontSize: 30, fontWeight: FontWeight.w900),
          ),
          const SizedBox(height: 6),
          Text(
            _recipient?.accountName ?? '',
            textAlign: TextAlign.center,
            style: const TextStyle(color: Colors.white70),
          ),
          const Divider(height: 34),
          _InfoRow('Mã giao dịch', receipt.referenceCode),
          _InfoRow('Trạng thái', receipt.status),
          _InfoRow('Số tài khoản', _recipient?.accountNumber ?? ''),
          _InfoRow(
            'Nội dung',
            receipt.description.isEmpty ? 'Chuyển tiền' : receipt.description,
          ),
          if (receipt.createdAt != null)
            _InfoRow('Thời gian', dateTimeText(receipt.createdAt)),
          const SizedBox(height: 22),
          FilledButton(
            onPressed: _reset,
            child: const Text('Thực hiện giao dịch mới'),
          ),
        ],
      ),
    );
  }
}

class _SectionLabel extends StatelessWidget {
  const _SectionLabel(this.label);

  final String label;

  @override
  Widget build(BuildContext context) => Text(
    label,
    style: const TextStyle(
      fontSize: 15,
      fontWeight: FontWeight.w800,
      letterSpacing: -.2,
    ),
  );
}

class _SourceAccountCard extends StatelessWidget {
  const _SourceAccountCard({required this.account});

  final Map<String, dynamic>? account;

  @override
  Widget build(BuildContext context) {
    final accountNumber =
        account?['account_number']?.toString() ?? 'Đang tải...';
    final currency = account?['currency']?.toString() ?? 'VND';
    return Row(
      children: [
        Container(
          width: 38,
          height: 38,
          decoration: BoxDecoration(
            color: const Color(0xFF7C83FD).withValues(alpha: .12),
            borderRadius: BorderRadius.circular(11),
          ),
          child: const Icon(
            Icons.account_balance_wallet_rounded,
            size: 20,
            color: Color(0xFF9EA4FF),
          ),
        ),
        const SizedBox(width: 11),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'THANH TOÁN · $accountNumber',
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  color: Color(0xFF9EA4FF),
                  fontSize: 11,
                  fontWeight: FontWeight.w800,
                  letterSpacing: .2,
                ),
              ),
              const SizedBox(height: 2),
              Text(
                account == null ? '—' : money(account!['balance'], currency),
                style: const TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w800,
                ),
              ),
            ],
          ),
        ),
        const Icon(
          Icons.keyboard_arrow_down_rounded,
          size: 20,
          color: Colors.white54,
        ),
      ],
    );
  }
}

class _BankSelector extends StatelessWidget {
  const _BankSelector();

  @override
  Widget build(BuildContext context) => const Padding(
    padding: EdgeInsets.symmetric(vertical: 7),
    child: Row(
      children: [
        _BankMark(),
        SizedBox(width: 10),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'Ngân hàng',
                style: TextStyle(color: Color(0xFF8D98B4), fontSize: 12),
              ),
              SizedBox(height: 3),
              Text(
                'NF Bank',
                style: TextStyle(fontSize: 15, fontWeight: FontWeight.w700),
              ),
            ],
          ),
        ),
        Icon(Icons.keyboard_arrow_down_rounded, color: Color(0xFF7C83FD)),
      ],
    ),
  );
}

class _RecentRecipientsButton extends StatelessWidget {
  const _RecentRecipientsButton({required this.count, required this.onTap});

  final int count;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) => Material(
    color: Colors.transparent,
    child: InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(8),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 6),
        child: Row(
          children: [
            const Icon(
              Icons.history_rounded,
              size: 18,
              color: Color(0xFF9EA4FF),
            ),
            const SizedBox(width: 8),
            const Expanded(
              child: Text(
                'Người nhận gần đây',
                style: TextStyle(
                  color: Color(0xFFB7BBFF),
                  fontSize: 12,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ),
            Text(
              '$count',
              style: const TextStyle(color: Color(0xFF77839E), fontSize: 11),
            ),
            const SizedBox(width: 3),
            const Icon(
              Icons.chevron_right_rounded,
              size: 18,
              color: Color(0xFF77839E),
            ),
          ],
        ),
      ),
    ),
  );
}

class _RecentRecipientsSheet extends StatelessWidget {
  const _RecentRecipientsSheet({required this.recipients});

  final List<RecentRecipient> recipients;

  @override
  Widget build(BuildContext context) => FractionallySizedBox(
    heightFactor: .72,
    child: Container(
      decoration: const BoxDecoration(
        color: Color(0xFF151D31),
        borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
        border: Border(top: BorderSide(color: Color(0x337C83FD))),
      ),
      child: SafeArea(
        top: false,
        child: Column(
          children: [
            const SizedBox(height: 9),
            Container(
              width: 40,
              height: 3,
              decoration: BoxDecoration(
                color: Colors.white24,
                borderRadius: BorderRadius.circular(10),
              ),
            ),
            Padding(
              padding: const EdgeInsets.fromLTRB(18, 12, 8, 8),
              child: Row(
                children: [
                  const Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          'Người nhận gần đây',
                          style: TextStyle(
                            fontSize: 18,
                            fontWeight: FontWeight.w900,
                          ),
                        ),
                        SizedBox(height: 2),
                        Text(
                          'Chọn tài khoản để chuyển nhanh',
                          style: TextStyle(
                            color: Color(0xFF8D98B4),
                            fontSize: 12,
                          ),
                        ),
                      ],
                    ),
                  ),
                  IconButton(
                    tooltip: 'Đóng',
                    onPressed: () => Navigator.pop(context),
                    icon: const Icon(Icons.close_rounded),
                  ),
                ],
              ),
            ),
            Divider(height: 1, color: Colors.white.withValues(alpha: .08)),
            Expanded(
              child: ListView.separated(
                padding: const EdgeInsets.fromLTRB(10, 8, 10, 16),
                itemCount: recipients.length,
                separatorBuilder: (_, _) => Divider(
                  height: 1,
                  indent: 54,
                  color: Colors.white.withValues(alpha: .06),
                ),
                itemBuilder: (context, index) {
                  final recipient = recipients[index];
                  return ListTile(
                    dense: true,
                    contentPadding: const EdgeInsets.symmetric(
                      horizontal: 8,
                      vertical: 3,
                    ),
                    leading: Container(
                      width: 38,
                      height: 38,
                      decoration: BoxDecoration(
                        color: const Color(0xFF7C83FD).withValues(alpha: .13),
                        borderRadius: BorderRadius.circular(11),
                      ),
                      child: const Icon(
                        Icons.person_outline_rounded,
                        size: 20,
                        color: Color(0xFF9EA4FF),
                      ),
                    ),
                    title: Text(
                      recipient.accountName,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        fontSize: 13,
                        fontWeight: FontWeight.w800,
                      ),
                    ),
                    subtitle: Text(
                      recipient.accountNumber,
                      style: const TextStyle(
                        color: Color(0xFF8D98B4),
                        fontSize: 12,
                      ),
                    ),
                    trailing: const Icon(
                      Icons.north_east_rounded,
                      size: 18,
                      color: Color(0xFF4FD1C5),
                    ),
                    onTap: () => Navigator.pop(context, recipient),
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

class _BankMark extends StatelessWidget {
  const _BankMark();

  @override
  Widget build(BuildContext context) => Container(
    width: 38,
    height: 38,
    decoration: BoxDecoration(
      gradient: const LinearGradient(
        colors: [Color(0xFF7C83FD), Color(0xFF4FD1C5)],
      ),
      borderRadius: BorderRadius.circular(11),
    ),
    child: const Icon(Icons.north_east_rounded, size: 20, color: Colors.white),
  );
}

class _PartyTile extends StatelessWidget {
  const _PartyTile({
    required this.name,
    required this.accountNumber,
    required this.bankName,
    required this.avatarURL,
  });

  final String name;
  final String accountNumber;
  final String bankName;
  final String avatarURL;

  @override
  Widget build(BuildContext context) => Row(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      _ProfileAvatar(name: name, avatarURL: avatarURL),
      const SizedBox(width: 11),
      Expanded(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              name.toUpperCase(),
              style: const TextStyle(fontWeight: FontWeight.w900, fontSize: 14),
            ),
            const SizedBox(height: 2),
            Text(
              accountNumber,
              style: const TextStyle(color: Colors.white70, fontSize: 13),
            ),
            Text(
              bankName,
              style: const TextStyle(color: Colors.white54, fontSize: 12),
            ),
          ],
        ),
      ),
    ],
  );
}

class _BottomActions extends StatelessWidget {
  const _BottomActions({
    required this.primaryLabel,
    required this.onPrimary,
    required this.onBack,
  });

  final String primaryLabel;
  final VoidCallback? onPrimary;
  final VoidCallback? onBack;

  @override
  Widget build(BuildContext context) => Center(
    child: ConstrainedBox(
      constraints: const BoxConstraints(maxWidth: 390),
      child: Row(
        children: [
          Expanded(
            flex: 2,
            child: OutlinedButton(
              onPressed: onBack,
              style: OutlinedButton.styleFrom(
                minimumSize: const Size.fromHeight(44),
                padding: const EdgeInsets.symmetric(horizontal: 10),
                side: const BorderSide(color: Color(0xFF7C83FD)),
                foregroundColor: const Color(0xFFB7BBFF),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(13),
                ),
              ),
              child: const Text(
                'Quay lại',
                style: TextStyle(fontSize: 13, fontWeight: FontWeight.w800),
              ),
            ),
          ),
          const SizedBox(width: 10),
          Expanded(
            flex: 3,
            child: FilledButton(
              onPressed: onPrimary,
              style: FilledButton.styleFrom(
                minimumSize: const Size.fromHeight(44),
                padding: const EdgeInsets.symmetric(horizontal: 10),
                backgroundColor: const Color(0xFF6D74F7),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(13),
                ),
              ),
              child: Text(
                primaryLabel,
                style: const TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w900,
                ),
              ),
            ),
          ),
        ],
      ),
    ),
  );
}

Future<TransactionPINInput?> showTransactionPINSheet(
  BuildContext context, {
  required bool createPIN,
  String actionLabel = 'chuyển tiền',
}) => showModalBottomSheet<TransactionPINInput>(
  context: context,
  useSafeArea: true,
  isScrollControlled: true,
  backgroundColor: Colors.transparent,
  isDismissible: false,
  enableDrag: false,
  builder: (_) =>
      TransactionPINSheet(createPIN: createPIN, actionLabel: actionLabel),
);

class TransactionPINInput {
  const TransactionPINInput(this.pin, this.confirmPIN);

  final String pin;
  final String? confirmPIN;
}

class TransactionPINSheet extends StatefulWidget {
  const TransactionPINSheet({
    super.key,
    required this.createPIN,
    required this.actionLabel,
  });

  final bool createPIN;
  final String actionLabel;

  @override
  State<TransactionPINSheet> createState() => _TransactionPINSheetState();
}

class _TransactionPINSheetState extends State<TransactionPINSheet> {
  String _pin = '';
  String _confirmPIN = '';
  String? _error;
  int _step = 1;

  bool get _isConfirmStep => widget.createPIN && _step == 2;
  String get _currentPIN => _isConfirmStep ? _confirmPIN : _pin;

  void _continue() {
    final error = _validatePIN(
      _currentPIN,
      rejectWeak: widget.createPIN && _step == 1,
    );
    if (error != null) {
      setState(() => _error = error);
      HapticFeedback.mediumImpact();
      return;
    }
    if (widget.createPIN && _step == 1) {
      setState(() {
        _step = 2;
        _error = null;
      });
      return;
    }
    if (_isConfirmStep && _confirmPIN != _pin) {
      setState(() => _error = 'Hai mã PIN chưa trùng khớp');
      HapticFeedback.mediumImpact();
      return;
    }
    HapticFeedback.lightImpact();
    Navigator.pop(
      context,
      TransactionPINInput(_pin, widget.createPIN ? _confirmPIN : null),
    );
  }

  void _typeDigit(String digit) {
    if (_currentPIN.length >= 6) return;
    HapticFeedback.selectionClick();
    setState(() {
      if (_isConfirmStep) {
        _confirmPIN += digit;
      } else {
        _pin += digit;
      }
      _error = null;
    });
  }

  void _deleteDigit() {
    if (_currentPIN.isEmpty) return;
    HapticFeedback.selectionClick();
    setState(() {
      if (_isConfirmStep) {
        _confirmPIN = _confirmPIN.substring(0, _confirmPIN.length - 1);
      } else {
        _pin = _pin.substring(0, _pin.length - 1);
      }
      _error = null;
    });
  }

  void _backToFirstStep() {
    setState(() {
      _step = 1;
      _confirmPIN = '';
      _error = null;
    });
  }

  @override
  Widget build(BuildContext context) => Container(
    decoration: const BoxDecoration(
      color: Color(0xFF151D31),
      borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
      border: Border(top: BorderSide(color: Color(0x337C83FD))),
    ),
    child: SafeArea(
      top: false,
      child: Center(
        heightFactor: 1,
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 460),
          child: SingleChildScrollView(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 14),
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
                const SizedBox(height: 7),
                _header(),
                const SizedBox(height: 7),
                if (widget.createPIN) _stepIndicator(),
                const SizedBox(height: 13),
                AnimatedSwitcher(
                  duration: const Duration(milliseconds: 220),
                  child: Column(
                    key: ValueKey(_step),
                    children: [
                      Text(
                        _isConfirmStep
                            ? 'Nhập lại mã PIN vừa tạo'
                            : widget.createPIN
                            ? 'Tạo mã PIN giao dịch'
                            : 'Xác thực giao dịch',
                        style: const TextStyle(
                          color: Colors.white,
                          fontSize: 16,
                          fontWeight: FontWeight.w800,
                        ),
                      ),
                      const SizedBox(height: 5),
                      Text(
                        _isConfirmStep
                            ? 'Hai mã PIN phải hoàn toàn giống nhau.'
                            : widget.createPIN
                            ? 'Nhập 6 số khó đoán để bảo vệ mọi giao dịch.'
                            : 'Vui lòng nhập mã PIN 6 số để xác nhận ${widget.actionLabel}.',
                        textAlign: TextAlign.center,
                        style: const TextStyle(
                          color: Color(0xFF9BA8C7),
                          fontSize: 12,
                          height: 1.3,
                        ),
                      ),
                      const SizedBox(height: 14),
                      _pinDots(),
                    ],
                  ),
                ),
                const SizedBox(height: 3),
                SizedBox(
                  height: 28,
                  child: Center(
                    child: AnimatedSwitcher(
                      duration: const Duration(milliseconds: 160),
                      child: _error == null
                          ? const SizedBox.shrink()
                          : Text(
                              _error!,
                              key: ValueKey(_error),
                              textAlign: TextAlign.center,
                              style: const TextStyle(
                                color: Color(0xFFFF8A9B),
                                fontSize: 12,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                    ),
                  ),
                ),
                _keypad(),
                const SizedBox(height: 8),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton(
                    style: FilledButton.styleFrom(
                      minimumSize: const Size.fromHeight(44),
                      backgroundColor: const Color(0xFF6D74F7),
                      foregroundColor: Colors.white,
                      disabledBackgroundColor: const Color(0xFF29334C),
                      disabledForegroundColor: const Color(0xFF77839E),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(12),
                      ),
                    ),
                    onPressed: _currentPIN.length == 6 ? _continue : null,
                    child: Text(
                      widget.createPIN && _step == 1
                          ? 'Tiếp tục'
                          : widget.createPIN
                          ? 'Tạo PIN và ${widget.actionLabel}'
                          : 'Xác nhận ${widget.actionLabel}',
                      style: const TextStyle(fontWeight: FontWeight.w800),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    ),
  );

  Widget _header() => Stack(
    alignment: Alignment.center,
    children: [
      Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          if (_isConfirmStep)
            IconButton(
              onPressed: _backToFirstStep,
              color: const Color(0xFF9EA4FF),
              visualDensity: VisualDensity.compact,
              icon: const Icon(Icons.arrow_back_rounded),
            )
          else
            const SizedBox(width: 48),
          IconButton(
            onPressed: () => Navigator.pop(context),
            color: const Color(0xFF9BA8C7),
            visualDensity: VisualDensity.compact,
            icon: const Icon(Icons.close_rounded),
          ),
        ],
      ),
      IgnorePointer(
        child: Text(
          widget.createPIN ? 'Thiết lập PIN' : 'Xác thực PIN',
          style: const TextStyle(
            color: Color(0xFF9EA4FF),
            fontSize: 18,
            fontWeight: FontWeight.w900,
          ),
        ),
      ),
    ],
  );

  Widget _stepIndicator() => Row(
    children: [
      Expanded(child: _StepPill(active: true, label: '1  Tạo PIN')),
      const SizedBox(width: 8),
      Expanded(
        child: _StepPill(active: _step == 2, label: '2  Xác nhận'),
      ),
    ],
  );

  Widget _pinDots() => Row(
    mainAxisAlignment: MainAxisAlignment.center,
    children: List.generate(6, (index) {
      final filled = index < _currentPIN.length;
      return AnimatedContainer(
        duration: const Duration(milliseconds: 140),
        width: 30,
        height: 30,
        margin: const EdgeInsets.symmetric(horizontal: 4),
        decoration: BoxDecoration(
          color: filled
              ? const Color(0xFF7C83FD).withValues(alpha: 0.12)
              : Colors.transparent,
          shape: BoxShape.circle,
          border: Border.all(
            width: 1.7,
            color: filled ? const Color(0xFF9EA4FF) : const Color(0xFF4A5874),
          ),
        ),
        child: Center(
          child: AnimatedScale(
            scale: filled ? 1 : 0,
            duration: const Duration(milliseconds: 140),
            child: Container(
              width: 23,
              height: 23,
              decoration: const BoxDecoration(
                color: Color(0xFF7C83FD),
                shape: BoxShape.circle,
              ),
            ),
          ),
        ),
      );
    }),
  );

  Widget _keypad() {
    const rows = [
      ['1', '2', '3'],
      ['4', '5', '6'],
      ['7', '8', '9'],
      ['', '0', 'delete'],
    ];
    return Column(
      children: [
        for (final row in rows) ...[
          Row(
            children: [
              for (final key in row)
                Expanded(
                  child: Padding(
                    padding: const EdgeInsets.all(4),
                    child: key.isEmpty
                        ? const SizedBox(height: 46)
                        : _KeypadButton(
                            label: key == 'delete' ? null : key,
                            icon: key == 'delete'
                                ? Icons.backspace_outlined
                                : null,
                            onTap: key == 'delete'
                                ? _deleteDigit
                                : () => _typeDigit(key),
                          ),
                  ),
                ),
            ],
          ),
        ],
      ],
    );
  }
}

class _StepPill extends StatelessWidget {
  const _StepPill({required this.active, required this.label});

  final bool active;
  final String label;

  @override
  Widget build(BuildContext context) => AnimatedContainer(
    duration: const Duration(milliseconds: 180),
    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
    decoration: BoxDecoration(
      color: active
          ? const Color(0xFF7C83FD).withValues(alpha: 0.14)
          : const Color(0xFF1D2740),
      borderRadius: BorderRadius.circular(10),
      border: Border.all(
        color: active ? const Color(0xFF7C83FD) : const Color(0xFF303C59),
      ),
    ),
    child: Text(
      label,
      textAlign: TextAlign.center,
      style: TextStyle(
        color: active ? const Color(0xFFB7BBFF) : const Color(0xFF77839E),
        fontWeight: FontWeight.w700,
        fontSize: 11,
      ),
    ),
  );
}

class _KeypadButton extends StatelessWidget {
  const _KeypadButton({required this.onTap, this.label, this.icon});

  final VoidCallback onTap;
  final String? label;
  final IconData? icon;

  @override
  Widget build(BuildContext context) => Material(
    color: const Color(0xFF202A43),
    borderRadius: BorderRadius.circular(12),
    child: InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(12),
      splashColor: const Color(0x337C83FD),
      child: SizedBox(
        height: 46,
        child: Center(
          child: icon != null
              ? Icon(icon, size: 20, color: const Color(0xFF9BA8C7))
              : Text(
                  label!,
                  style: const TextStyle(
                    color: Colors.white,
                    fontSize: 21,
                    fontWeight: FontWeight.w600,
                  ),
                ),
        ),
      ),
    ),
  );
}

String? _validatePIN(String? value, {required bool rejectWeak}) {
  final pin = value ?? '';
  if (!RegExp(r'^[0-9]{6}$').hasMatch(pin)) {
    return 'PIN phải gồm đúng 6 chữ số';
  }
  if (!rejectWeak) return null;
  const weakPINs = {
    '000000',
    '111111',
    '222222',
    '333333',
    '444444',
    '555555',
    '666666',
    '777777',
    '888888',
    '999999',
    '012345',
    '123456',
    '234567',
    '345678',
    '456789',
    '987654',
    '876543',
    '765432',
    '654321',
  };
  return weakPINs.contains(pin) ? 'PIN quá dễ đoán, hãy chọn mã khác' : null;
}

class _RecipientPreview extends StatelessWidget {
  const _RecipientPreview({required this.recipient});

  final AccountResolution recipient;

  @override
  Widget build(BuildContext context) => Container(
    width: double.infinity,
    padding: const EdgeInsets.fromLTRB(0, 8, 0, 4),
    decoration: BoxDecoration(
      border: Border(
        top: BorderSide(color: const Color(0xFF4FD1C5).withValues(alpha: 0.22)),
      ),
    ),
    child: Row(
      children: [
        Stack(
          clipBehavior: Clip.none,
          children: [
            _ProfileAvatar(
              name: recipient.accountName,
              avatarURL: recipient.avatarUrl,
            ),
            const Positioned(
              right: -3,
              bottom: -3,
              child: Icon(
                Icons.verified_rounded,
                size: 16,
                color: Color(0xFF4FD1C5),
              ),
            ),
          ],
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                recipient.accountName,
                style: const TextStyle(
                  fontWeight: FontWeight.w900,
                  letterSpacing: .2,
                ),
              ),
              const SizedBox(height: 3),
              Text(
                'Đã xác thực chủ tài khoản',
                style: const TextStyle(color: Color(0xFF6EDDD3), fontSize: 12),
              ),
            ],
          ),
        ),
      ],
    ),
  );
}

class _ProfileAvatar extends StatelessWidget {
  const _ProfileAvatar({required this.name, required this.avatarURL});

  final String name;
  final String avatarURL;

  @override
  Widget build(BuildContext context) {
    final normalizedURL = avatarURL.trim();
    final initial = name.trim().isEmpty ? 'N' : name.trim()[0].toUpperCase();
    return Container(
      padding: const EdgeInsets.all(1.5),
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        border: Border.all(color: const Color(0xFF8F95FF), width: 1.5),
      ),
      child: CircleAvatar(
        radius: 19.5,
        backgroundColor: const Color(0xFF293352),
        backgroundImage: normalizedURL.startsWith('https://')
            ? NetworkImage(normalizedURL)
            : null,
        child: normalizedURL.startsWith('https://')
            ? null
            : Text(
                initial,
                style: const TextStyle(
                  color: Color(0xFFB7BBFF),
                  fontWeight: FontWeight.w900,
                ),
              ),
      ),
    );
  }
}

class _InfoRow extends StatelessWidget {
  const _InfoRow(this.label, this.value);

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 7),
    child: Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 130,
          child: Text(label, style: const TextStyle(color: Colors.white54)),
        ),
        Expanded(
          child: SelectableText(
            value,
            textAlign: TextAlign.end,
            style: const TextStyle(fontWeight: FontWeight.w600),
          ),
        ),
      ],
    ),
  );
}
