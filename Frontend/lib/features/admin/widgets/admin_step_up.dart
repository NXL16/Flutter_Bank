import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../../core/network/api_service.dart';
import '../../../shared/widgets/common.dart';
import '../services/admin_repository.dart';

abstract final class AdminSensitiveAction {
  static const deposit = 'DEPOSIT';
  static const createAdmin = 'CREATE_ADMIN';
  static const lockUser = 'LOCK_USER';
  static const unlockUser = 'UNLOCK_USER';
}

Future<String?> showAdminStepUp(
  BuildContext context, {
  required String action,
  required String binding,
  required String title,
  required String transactionSummary,
}) => showModalBottomSheet<String>(
  context: context,
  useSafeArea: true,
  isScrollControlled: true,
  backgroundColor: Colors.transparent,
  builder: (_) => _AdminStepUpSheet(
    action: action,
    binding: binding,
    title: title,
    transactionSummary: transactionSummary,
  ),
);

class _AdminStepUpSheet extends StatefulWidget {
  const _AdminStepUpSheet({
    required this.action,
    required this.binding,
    required this.title,
    required this.transactionSummary,
  });

  final String action;
  final String binding;
  final String title;
  final String transactionSummary;

  @override
  State<_AdminStepUpSheet> createState() => _AdminStepUpSheetState();
}

class _AdminStepUpSheetState extends State<_AdminStepUpSheet> {
  final _code = TextEditingController();
  final _repository = const AdminRepository();
  bool _loading = false;
  String? _error;

  @override
  void dispose() {
    _code.dispose();
    super.dispose();
  }

  Future<void> _verify() async {
    final code = _code.text.trim();
    if (!RegExp(r'^\d{6}$').hasMatch(code) || _loading) {
      setState(() => _error = 'Mã TOTP phải gồm đúng 6 số');
      return;
    }
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final token = await _repository.createStepUp(
        action: widget.action,
        totpCode: code,
        binding: widget.binding,
      );
      if (mounted) Navigator.pop(context, token);
    } on ApiException catch (error) {
      if (mounted) {
        setState(() {
          _error = error.message;
          _code.clear();
        });
      }
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) => AnimatedPadding(
    duration: const Duration(milliseconds: 180),
    padding: EdgeInsets.only(bottom: MediaQuery.viewInsetsOf(context).bottom),
    child: Container(
      padding: const EdgeInsets.fromLTRB(20, 10, 20, 20),
      decoration: const BoxDecoration(
        color: Color(0xFF11192B),
        borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
        border: Border(top: BorderSide(color: Color(0xFF344263))),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 42,
            height: 4,
            decoration: BoxDecoration(
              color: Colors.white24,
              borderRadius: BorderRadius.circular(10),
            ),
          ),
          const SizedBox(height: 16),
          Container(
            width: 44,
            height: 44,
            decoration: BoxDecoration(
              color: const Color(0x22FFB566),
              borderRadius: BorderRadius.circular(14),
            ),
            child: const Icon(Icons.security_rounded, color: Color(0xFFFFB566)),
          ),
          const SizedBox(height: 12),
          Text(
            widget.title,
            textAlign: TextAlign.center,
            style: const TextStyle(fontSize: 19, fontWeight: FontWeight.w900),
          ),
          const SizedBox(height: 6),
          const Text(
            'Xác thực nâng cao cho thao tác nhạy cảm',
            style: TextStyle(color: Color(0xFF8792AD), fontSize: 12),
          ),
          const SizedBox(height: 14),
          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: const Color(0xFF0D1424),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: const Color(0xFF283550)),
            ),
            child: Text(
              widget.transactionSummary,
              textAlign: TextAlign.center,
              style: const TextStyle(
                color: Color(0xFFC1C9DC),
                fontSize: 12,
                height: 1.4,
              ),
            ),
          ),
          const SizedBox(height: 14),
          TextField(
            controller: _code,
            autofocus: true,
            keyboardType: TextInputType.number,
            maxLength: 6,
            textAlign: TextAlign.center,
            style: const TextStyle(
              fontSize: 24,
              letterSpacing: 10,
              fontWeight: FontWeight.w800,
            ),
            inputFormatters: [FilteringTextInputFormatter.digitsOnly],
            decoration: fieldDecoration('Mã từ ứng dụng Authenticator')
                .copyWith(
                  counterText: '',
                  errorText: _error,
                  prefixIcon: const Icon(Icons.password_rounded),
                ),
            onSubmitted: (_) => _verify(),
          ),
          const SizedBox(height: 14),
          Row(
            children: [
              Expanded(
                child: OutlinedButton(
                  onPressed: _loading ? null : () => Navigator.pop(context),
                  child: const Text('Hủy'),
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                flex: 2,
                child: FilledButton.icon(
                  onPressed: _loading ? null : _verify,
                  icon: _loading
                      ? const SizedBox.square(
                          dimension: 17,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : const Icon(Icons.verified_user_outlined),
                  label: Text(_loading ? 'Đang xác thực...' : 'Xác thực'),
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
          const Text(
            'Mã chỉ được dùng một lần và quyền phê duyệt hết hạn sau 2 phút.',
            textAlign: TextAlign.center,
            style: TextStyle(color: Color(0xFF66728E), fontSize: 10),
          ),
        ],
      ),
    ),
  );
}
