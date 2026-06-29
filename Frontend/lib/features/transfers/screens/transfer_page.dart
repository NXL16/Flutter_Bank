import 'dart:async';

import 'package:flutter/material.dart';

import '../../../core/network/api_service.dart';
import '../../../core/storage/token_storage.dart';
import '../../../shared/utils/formatters.dart';
import '../../../shared/widgets/common.dart';
import '../models/account_resolution.dart';
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

  _TransferStep _step = _TransferStep.details;
  AccountResolution? _recipient;
  TransferReceipt? _receipt;
  String? _idempotencyKey;
  String? _resolutionError;
  String _defaultDescription = 'Chuyen khoan';
  Timer? _resolveDebounce;
  int _resolutionRequest = 0;
  bool _resolving = false;
  bool _loading = false;

  int get _parsedAmount =>
      int.tryParse(_amount.text.replaceAll(RegExp(r'[^0-9]'), '')) ?? 0;

  @override
  void initState() {
    super.initState();
    _loadDefaultDescription();
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
          showMessage(context, error.message, error: true);
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
      if (mounted) showMessage(context, error.message, error: true);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _submit() async {
    final recipient = _recipient;
    final requestID = _idempotencyKey;
    if (recipient == null || requestID == null) return;

    setState(() => _loading = true);
    try {
      final receipt = await _repository.transfer(
        accountNumber: recipient.accountNumber,
        amount: _parsedAmount,
        description: _description.text.trim(),
        idempotencyKey: requestID,
      );
      if (!mounted) return;
      setState(() {
        _receipt = receipt;
        _step = _TransferStep.receipt;
      });
    } on ApiException catch (error) {
      if (mounted) showMessage(context, error.message, error: true);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
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
    children: [
      PageTitle(
        _step == _TransferStep.receipt ? 'Giao dịch hoàn tất' : 'Chuyển tiền',
        subtitle: _subtitle,
      ),
      const SizedBox(height: 20),
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
    _TransferStep.details => 'Chuyển khoản nội bộ an toàn theo số tài khoản.',
    _TransferStep.confirmation => 'Kiểm tra kỹ thông tin trước khi xác nhận.',
    _TransferStep.receipt => 'Tiền đã được ghi nhận trong hệ thống NF Bank.',
  };

  Widget _detailsCard() => SurfaceCard(
    key: const ValueKey('details'),
    child: Form(
      key: _form,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const _StepHeader(index: 1, title: 'Thông tin giao dịch'),
          const SizedBox(height: 20),
          TextFormField(
            controller: _account,
            keyboardType: TextInputType.number,
            maxLength: 12,
            decoration: fieldDecoration('Số tài khoản người nhận'),
            onChanged: _onAccountChanged,
            validator: (value) =>
                RegExp(r'^[0-9]{12}$').hasMatch(value?.trim() ?? '')
                ? null
                : 'Số tài khoản phải gồm 12 chữ số',
          ),
          if (_resolving) ...[
            const LinearProgressIndicator(minHeight: 2),
            const SizedBox(height: 12),
          ] else if (_recipient != null) ...[
            _RecipientPreview(recipient: _recipient!),
            const SizedBox(height: 12),
          ] else if (_resolutionError != null) ...[
            Text(
              _resolutionError!,
              style: const TextStyle(color: Color(0xFFFF8A80)),
            ),
            const SizedBox(height: 12),
          ],
          const SizedBox(height: 12),
          TextFormField(
            controller: _amount,
            keyboardType: TextInputType.number,
            decoration: fieldDecoration('Số tiền (VND)'),
            validator: (_) => _parsedAmount >= 10000
                ? null
                : 'Số tiền tối thiểu là 10.000 VND',
          ),
          const SizedBox(height: 12),
          TextFormField(
            controller: _description,
            maxLength: 140,
            decoration: fieldDecoration('Nội dung', hint: _defaultDescription),
          ),
          const SizedBox(height: 8),
          FilledButton.icon(
            onPressed: _loading ? null : _review,
            icon: const Icon(Icons.arrow_forward_rounded),
            label: Text(_loading ? 'Đang xác minh...' : 'Tiếp tục'),
          ),
        ],
      ),
    ),
  );

  Widget _confirmationCard() {
    final recipient = _recipient!;
    return SurfaceCard(
      key: const ValueKey('confirmation'),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const _StepHeader(index: 2, title: 'Xác nhận giao dịch'),
          const SizedBox(height: 22),
          _InfoRow('Người nhận', recipient.accountName),
          _InfoRow('Ngân hàng', recipient.bankName),
          _InfoRow('Số tài khoản', recipient.accountNumber),
          const Divider(height: 28),
          _InfoRow(
            'Số tiền',
            money(_parsedAmount, recipient.currency),
            strong: true,
          ),
          const _InfoRow('Phí giao dịch', 'Miễn phí'),
          _InfoRow(
            'Nội dung',
            _description.text.trim().isEmpty
                ? 'Chuyển tiền'
                : _description.text.trim(),
          ),
          const SizedBox(height: 22),
          FilledButton.icon(
            onPressed: _loading ? null : _submit,
            icon: const Icon(Icons.lock_outline_rounded),
            label: Text(_loading ? 'Đang xử lý...' : 'Xác nhận chuyển tiền'),
          ),
          TextButton(
            onPressed: _loading
                ? null
                : () => setState(() => _step = _TransferStep.details),
            child: const Text('Chỉnh sửa thông tin'),
          ),
        ],
      ),
    );
  }

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

class _StepHeader extends StatelessWidget {
  const _StepHeader({required this.index, required this.title});

  final int index;
  final String title;

  @override
  Widget build(BuildContext context) => Row(
    children: [
      CircleAvatar(radius: 17, child: Text('$index')),
      const SizedBox(width: 12),
      Text(title, style: Theme.of(context).textTheme.titleLarge),
    ],
  );
}

class _RecipientPreview extends StatelessWidget {
  const _RecipientPreview({required this.recipient});

  final AccountResolution recipient;

  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.all(14),
    decoration: BoxDecoration(
      color: const Color(0xFF68D391).withValues(alpha: 0.09),
      borderRadius: BorderRadius.circular(14),
      border: Border.all(
        color: const Color(0xFF68D391).withValues(alpha: 0.35),
      ),
    ),
    child: Row(
      children: [
        const Icon(Icons.verified_rounded, color: Color(0xFF68D391)),
        const SizedBox(width: 12),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                recipient.accountName,
                style: const TextStyle(fontWeight: FontWeight.w800),
              ),
              const SizedBox(height: 3),
              Text(
                '${recipient.bankName} · ${recipient.accountNumber}',
                style: const TextStyle(color: Colors.white60),
              ),
            ],
          ),
        ),
      ],
    ),
  );
}

class _InfoRow extends StatelessWidget {
  const _InfoRow(this.label, this.value, {this.strong = false});

  final String label;
  final String value;
  final bool strong;

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
            style: TextStyle(
              fontWeight: strong ? FontWeight.w900 : FontWeight.w600,
            ),
          ),
        ),
      ],
    ),
  );
}
