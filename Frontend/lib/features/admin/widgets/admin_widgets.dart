import 'package:flutter/material.dart';

import '../models/admin_models.dart';

class AdminSectionHeader extends StatelessWidget {
  const AdminSectionHeader({
    super.key,
    required this.title,
    this.subtitle,
    this.trailing,
  });

  final String title;
  final String? subtitle;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) => Row(
    crossAxisAlignment: CrossAxisAlignment.center,
    children: [
      Expanded(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              title,
              style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w900),
            ),
            if (subtitle != null) ...[
              const SizedBox(height: 3),
              Text(
                subtitle!,
                style: const TextStyle(color: Color(0xFF8792AD), fontSize: 12),
              ),
            ],
          ],
        ),
      ),
      ?trailing,
    ],
  );
}

class AdminPanel extends StatelessWidget {
  const AdminPanel({
    super.key,
    required this.child,
    this.padding = const EdgeInsets.all(18),
  });

  final Widget child;
  final EdgeInsetsGeometry padding;

  @override
  Widget build(BuildContext context) => Container(
    padding: padding,
    decoration: BoxDecoration(
      color: const Color(0xFF151D31),
      borderRadius: BorderRadius.circular(18),
      border: Border.all(color: const Color(0xFF27334F)),
      boxShadow: const [
        BoxShadow(
          color: Color(0x24000000),
          blurRadius: 20,
          offset: Offset(0, 8),
        ),
      ],
    ),
    child: child,
  );
}

class AdminMetricCard extends StatelessWidget {
  const AdminMetricCard({
    super.key,
    required this.label,
    required this.value,
    required this.caption,
    required this.icon,
    required this.accent,
    this.onTap,
  });

  final String label;
  final String value;
  final String caption;
  final IconData icon;
  final Color accent;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) => AdminPanel(
    padding: EdgeInsets.zero,
    child: Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(18),
        child: Padding(
          padding: const EdgeInsets.all(18),
          child: Row(
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
              const SizedBox(width: 13),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      label,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        color: Color(0xFF8E99B5),
                        fontSize: 11,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                    const SizedBox(height: 3),
                    Text(
                      value,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        fontSize: 19,
                        fontWeight: FontWeight.w900,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      caption,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        color: Color(0xFF66728E),
                        fontSize: 10,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    ),
  );
}

class AdminUserAvatar extends StatelessWidget {
  const AdminUserAvatar({super.key, required this.user, this.radius = 19});

  final AdminUserSummary user;
  final double radius;

  @override
  Widget build(BuildContext context) {
    final initial = user.fullName.trim().isEmpty
        ? '?'
        : user.fullName.trim().characters.first.toUpperCase();
    final hasAvatar = user.avatarUrl != null && user.avatarUrl!.isNotEmpty;

    return CircleAvatar(
      radius: radius,
      backgroundColor: user.role == 'user'
          ? const Color(0xFF273457)
          : const Color(0xFF463461),
      foregroundColor: user.role == 'user'
          ? const Color(0xFFAEB5FF)
          : const Color(0xFFD7B8FF),
      backgroundImage: hasAvatar ? NetworkImage(user.avatarUrl!) : null,
      child: hasAvatar
          ? null
          : Text(initial, style: const TextStyle(fontWeight: FontWeight.w900)),
    );
  }
}

class AdminStatusBadge extends StatelessWidget {
  const AdminStatusBadge({super.key, required this.label, required this.color});

  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 5),
    decoration: BoxDecoration(
      color: color.withValues(alpha: .12),
      borderRadius: BorderRadius.circular(20),
      border: Border.all(color: color.withValues(alpha: .28)),
    ),
    child: Text(
      label,
      style: TextStyle(color: color, fontSize: 10, fontWeight: FontWeight.w800),
    ),
  );
}

class AdminInfoRow extends StatelessWidget {
  const AdminInfoRow(this.label, this.value, {super.key});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.symmetric(vertical: 7),
    child: Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Expanded(
          child: Text(
            label,
            style: const TextStyle(color: Color(0xFF8792AD), fontSize: 12),
          ),
        ),
        const SizedBox(width: 16),
        Expanded(
          flex: 2,
          child: SelectableText(
            value,
            textAlign: TextAlign.end,
            style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w700),
          ),
        ),
      ],
    ),
  );
}

String adminRoleLabel(String role) => switch (role) {
  'super_admin' => 'Super Admin',
  'admin' => 'Admin',
  _ => 'Khách hàng',
};
