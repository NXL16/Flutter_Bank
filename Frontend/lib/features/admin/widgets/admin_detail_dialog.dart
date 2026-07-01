import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../../shared/utils/formatters.dart';
import '../../../shared/widgets/common.dart';
import '../models/admin_models.dart';
import 'admin_widgets.dart';

class AdminDetailData {
  const AdminDetailData(
    this.label,
    this.value, {
    this.emphasized = false,
    this.copyable = false,
  });

  final String label;
  final String value;
  final bool emphasized;
  final bool copyable;
}

Future<void> showAdminTransactionSummaryDetail(
  BuildContext context,
  AdminTransactionSummary item,
) => showAdminTransactionDetail(
  context,
  referenceCode: item.referenceCode,
  type: item.type,
  amount: item.amount,
  currency: item.currency,
  status: item.status,
  description: item.description,
  createdAt: item.createdAt,
);

Future<void> showAdminAccountTransactionDetail(
  BuildContext context,
  Map<String, dynamic> transaction, {
  required String customerName,
}) {
  final direction = transaction['direction']?.toString().toUpperCase() ?? 'IN';
  final type = transaction['type']?.toString().toUpperCase() ?? 'GIAO DỊCH';
  final counterparty = transaction['counterparty_name']?.toString() ?? '';

  String sender;
  String receiver;
  if (direction == 'IN') {
    receiver = customerName;
    sender = type == 'DEPOSIT'
        ? 'Hệ thống / Admin'
        : counterparty.isNotEmpty
        ? counterparty
        : 'Hệ thống / Chuyển khoản';
  } else {
    sender = customerName;
    receiver = type == 'WITHDRAWAL'
        ? 'Rút tiền mặt'
        : counterparty.isNotEmpty
        ? counterparty
        : 'Ngoại tuyến / Khác';
  }

  return showAdminTransactionDetail(
    context,
    referenceCode: transaction['reference_code']?.toString() ?? '—',
    type: type,
    amount: transaction['amount'],
    currency: transaction['currency']?.toString() ?? 'VND',
    status: transaction['status']?.toString() ?? 'SUCCESS',
    description: transaction['description']?.toString() ?? '',
    createdAt: transaction['created_at'],
    sender: sender,
    receiver: receiver,
  );
}

Future<void> showAdminTransactionDetail(
  BuildContext context, {
  required String referenceCode,
  required String type,
  required dynamic amount,
  required String currency,
  required String status,
  required String description,
  required dynamic createdAt,
  String? sender,
  String? receiver,
}) {
  final success = status.toUpperCase() == 'SUCCESS';
  return showDialog<void>(
    context: context,
    builder: (_) => AdminDetailDialog(
      icon: Icons.receipt_long_rounded,
      accent: success ? const Color(0xFF50D4A3) : const Color(0xFFFF7D8A),
      title: 'Chi tiết giao dịch',
      badge: status,
      rows: [
        AdminDetailData('Mã giao dịch', referenceCode, copyable: true),
        AdminDetailData('Loại giao dịch', type.replaceAll('_', ' ')),
        AdminDetailData('Số tiền', money(amount, currency), emphasized: true),
        if (sender != null) AdminDetailData('Người gửi', sender),
        if (receiver != null) AdminDetailData('Người nhận', receiver),
        AdminDetailData('Thời gian', dateTimeText(createdAt)),
        AdminDetailData(
          'Nội dung',
          description.trim().isEmpty ? 'Không có nội dung' : description,
        ),
      ],
    ),
  );
}

Future<void> showAdminAuditDetail(BuildContext context, AdminAuditLog item) {
  return showDialog<void>(
    context: context,
    builder: (_) => AdminDetailDialog(
      icon: Icons.verified_user_outlined,
      accent: const Color(0xFFFFB566),
      title: 'Chi tiết nhật ký',
      badge: item.action.replaceAll('_', ' '),
      rows: [
        AdminDetailData('Người thực hiện', item.actorName, emphasized: true),
        AdminDetailData('Hành động', item.action.replaceAll('_', ' ')),
        AdminDetailData('Đối tượng', item.targetType.replaceAll('_', ' ')),
        AdminDetailData('Mã đối tượng', item.targetId, copyable: true),
        AdminDetailData(
          'Địa chỉ IP',
          item.ipAddress.isEmpty ? 'Không ghi nhận' : item.ipAddress,
        ),
        AdminDetailData('Thời gian', dateTimeText(item.createdAt)),
        AdminDetailData(
          'Nội dung',
          item.summary.isEmpty ? 'Không có nội dung' : item.summary,
        ),
      ],
    ),
  );
}

class AdminDetailDialog extends StatelessWidget {
  const AdminDetailDialog({
    super.key,
    required this.icon,
    required this.accent,
    required this.title,
    required this.badge,
    required this.rows,
  });

  final IconData icon;
  final Color accent;
  final String title;
  final String badge;
  final List<AdminDetailData> rows;

  @override
  Widget build(BuildContext context) => Dialog(
    insetPadding: const EdgeInsets.symmetric(horizontal: 20, vertical: 28),
    shape: RoundedRectangleBorder(
      borderRadius: BorderRadius.circular(22),
      side: const BorderSide(color: Color(0xFF2A3652)),
    ),
    child: ConstrainedBox(
      constraints: const BoxConstraints(maxWidth: 500, maxHeight: 650),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(20, 20, 20, 16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              children: [
                Container(
                  width: 44,
                  height: 44,
                  decoration: BoxDecoration(
                    color: accent.withValues(alpha: .14),
                    borderRadius: BorderRadius.circular(13),
                  ),
                  child: Icon(icon, color: accent, size: 22),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        title,
                        style: const TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.w900,
                        ),
                      ),
                      const SizedBox(height: 4),
                      AdminStatusBadge(label: badge, color: accent),
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
            const SizedBox(height: 18),
            Flexible(
              child: SingleChildScrollView(
                child: Container(
                  padding: const EdgeInsets.symmetric(horizontal: 15),
                  decoration: BoxDecoration(
                    color: const Color(0xFF111A2D),
                    borderRadius: BorderRadius.circular(16),
                    border: Border.all(color: const Color(0xFF26334F)),
                  ),
                  child: Column(
                    children: [
                      for (var index = 0; index < rows.length; index++) ...[
                        _AdminDetailRow(data: rows[index]),
                        if (index < rows.length - 1) const Divider(height: 1),
                      ],
                    ],
                  ),
                ),
              ),
            ),
            const SizedBox(height: 16),
            Align(
              alignment: Alignment.centerRight,
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

class _AdminDetailRow extends StatelessWidget {
  const _AdminDetailRow({required this.data});

  final AdminDetailData data;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 13),
    child: Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SizedBox(
          width: 116,
          child: Text(
            data.label,
            style: const TextStyle(color: Color(0xFF8994AE), fontSize: 12),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: SelectableText(
            data.value,
            textAlign: TextAlign.right,
            style: TextStyle(
              fontSize: 13,
              height: 1.35,
              fontWeight: data.emphasized ? FontWeight.w900 : FontWeight.w600,
            ),
          ),
        ),
        if (data.copyable) ...[
          const SizedBox(width: 6),
          IconButton(
            visualDensity: VisualDensity.compact,
            tooltip: 'Sao chép',
            onPressed: () async {
              await Clipboard.setData(ClipboardData(text: data.value));
              if (context.mounted) showMessage(context, 'Đã sao chép');
            },
            icon: const Icon(Icons.copy_rounded, size: 16),
          ),
        ],
      ],
    ),
  );
}
