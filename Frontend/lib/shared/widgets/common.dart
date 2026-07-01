import 'package:flutter/material.dart';

class PageTitle extends StatelessWidget {
  const PageTitle(this.title, {super.key, this.subtitle, this.trailing});

  final String title;
  final String? subtitle;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) => Row(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      Expanded(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(title, style: Theme.of(context).textTheme.headlineSmall),
            if (subtitle != null) ...[
              const SizedBox(height: 4),
              Text(
                subtitle!,
                style: Theme.of(
                  context,
                ).textTheme.bodyMedium?.copyWith(color: Colors.white60),
              ),
            ],
          ],
        ),
      ),
      ?trailing,
    ],
  );
}

class SurfaceCard extends StatelessWidget {
  const SurfaceCard({super.key, required this.child, this.padding});

  final Widget child;
  final EdgeInsetsGeometry? padding;

  @override
  Widget build(BuildContext context) => Container(
    padding: padding ?? const EdgeInsets.all(20),
    decoration: BoxDecoration(
      color: Theme.of(context).colorScheme.surface,
      borderRadius: BorderRadius.circular(20),
      border: Border.all(color: Colors.white.withValues(alpha: .07)),
    ),
    child: child,
  );
}

class EmptyState extends StatelessWidget {
  const EmptyState({
    super.key,
    required this.icon,
    required this.title,
    required this.message,
    this.actionLabel,
    this.onAction,
  });

  final IconData icon;
  final String title;
  final String message;
  final String? actionLabel;
  final VoidCallback? onAction;

  @override
  Widget build(BuildContext context) => Center(
    child: Padding(
      padding: const EdgeInsets.all(32),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 48, color: Colors.white24),
          const SizedBox(height: 12),
          Text(title, style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 6),
          Text(
            message,
            textAlign: TextAlign.center,
            style: const TextStyle(color: Colors.white54),
          ),
          if (actionLabel != null && onAction != null) ...[
            const SizedBox(height: 16),
            FilledButton.tonalIcon(
              onPressed: onAction,
              icon: const Icon(Icons.refresh_rounded, size: 18),
              label: Text(actionLabel!),
            ),
          ],
        ],
      ),
    ),
  );
}

class AsyncPage extends StatelessWidget {
  const AsyncPage({
    super.key,
    required this.loading,
    required this.child,
    this.error,
    this.onRetry,
  });

  final bool loading;
  final String? error;
  final VoidCallback? onRetry;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    if (loading) return const Center(child: CircularProgressIndicator());
    if (error != null) {
      return EmptyState(
        icon: Icons.cloud_off_rounded,
        title: 'Không tải được dữ liệu',
        message: error!,
        actionLabel: onRetry == null ? null : 'Thử lại',
        onAction: onRetry,
      );
    }
    return child;
  }
}

OverlayEntry? _activeNotice;

void dismissActiveNotice() {
  _activeNotice?.remove();
  _activeNotice = null;
}

void showMessage(
  BuildContext context,
  String message, {
  bool error = false,
  bool transaction = false,
}) {
  final overlay = Overlay.of(context, rootOverlay: true);
  _activeNotice?.remove();

  late final OverlayEntry entry;
  entry = OverlayEntry(
    builder: (_) => _AppNotice(
      message: message,
      error: error,
      transaction: transaction,
      onDismiss: () {
        if (entry.mounted) entry.remove();
        if (identical(_activeNotice, entry)) _activeNotice = null;
      },
    ),
  );
  _activeNotice = entry;
  overlay.insert(entry);
}

class _AppNotice extends StatefulWidget {
  const _AppNotice({
    required this.message,
    required this.error,
    required this.transaction,
    required this.onDismiss,
  });

  final String message;
  final bool error;
  final bool transaction;
  final VoidCallback onDismiss;

  @override
  State<_AppNotice> createState() => _AppNoticeState();
}

class _AppNoticeState extends State<_AppNotice>
    with SingleTickerProviderStateMixin {
  late final AnimationController _controller;
  late final Animation<Offset> _slide;
  bool _closing = false;

  @override
  void initState() {
    super.initState();
    _controller = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 260),
      reverseDuration: const Duration(milliseconds: 180),
    );
    _slide = Tween<Offset>(
      begin: widget.transaction ? const Offset(0, -1.15) : const Offset(.22, 0),
      end: Offset.zero,
    ).animate(CurvedAnimation(parent: _controller, curve: Curves.easeOutCubic));
    _controller.forward();
    Future<void>.delayed(
      Duration(seconds: widget.transaction ? 6 : 4),
      _dismiss,
    );
  }

  Future<void> _dismiss() async {
    if (!mounted || _closing) return;
    _closing = true;
    await _controller.reverse();
    if (mounted) widget.onDismiss();
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final screenWidth = MediaQuery.sizeOf(context).width;
    final width = screenWidth < 404 ? screenWidth - 24 : 380.0;
    final accent = widget.error
        ? const Color(0xFFFF6B7A)
        : widget.transaction
        ? const Color(0xFF4FD1C5)
        : const Color(0xFF7C83FD);
    final title = widget.error
        ? 'Có lỗi xảy ra'
        : widget.transaction
        ? 'Thông báo giao dịch'
        : 'Thành công';
    final icon = widget.error
        ? Icons.error_outline_rounded
        : widget.transaction
        ? Icons.receipt_long_rounded
        : Icons.check_circle_outline_rounded;

    return Positioned(
      top: MediaQuery.paddingOf(context).top + 12,
      right: 12,
      width: width,
      child: SlideTransition(
        position: _slide,
        child: FadeTransition(
          opacity: _controller,
          child: Material(
            color: Colors.transparent,
            child: Container(
              padding: const EdgeInsets.fromLTRB(14, 12, 8, 12),
              decoration: BoxDecoration(
                color: const Color(0xFF182139),
                borderRadius: BorderRadius.circular(14),
                border: Border.all(color: accent.withValues(alpha: .42)),
                boxShadow: const [
                  BoxShadow(
                    color: Color(0x66000000),
                    blurRadius: 24,
                    offset: Offset(0, 10),
                  ),
                ],
              ),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Container(
                    width: 34,
                    height: 34,
                    decoration: BoxDecoration(
                      color: accent.withValues(alpha: .13),
                      borderRadius: BorderRadius.circular(10),
                    ),
                    child: Icon(icon, size: 19, color: accent),
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          title,
                          style: const TextStyle(
                            color: Colors.white,
                            fontSize: 13,
                            fontWeight: FontWeight.w800,
                          ),
                        ),
                        const SizedBox(height: 3),
                        Text(
                          widget.message,
                          maxLines: widget.transaction ? 4 : 3,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(
                            color: Color(0xFFB8C2D9),
                            fontSize: 12,
                            height: 1.35,
                          ),
                        ),
                      ],
                    ),
                  ),
                  IconButton(
                    tooltip: 'Đóng',
                    onPressed: _dismiss,
                    visualDensity: VisualDensity.compact,
                    iconSize: 18,
                    color: const Color(0xFF8D98B4),
                    icon: const Icon(Icons.close_rounded),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

InputDecoration fieldDecoration(String label, {String? hint}) =>
    InputDecoration(labelText: label, hintText: hint);
