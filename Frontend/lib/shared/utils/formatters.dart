import 'package:flutter/services.dart';

String money(dynamic value, [String currency = 'VND']) {
  // Chuyển value về số
  final number = value is num
      ? value
      : num.tryParse(value?.toString() ?? '0') ?? 0;

  final negative = number < 0;

  // Lấy phần nguyên
  final digits = number.abs().toInt().toString();

  // Thêm dấu phân cách hàng nghìn
  final formatted = digits.replaceAllMapped(
    RegExp(r'\B(?=(\d{3})+(?!\d))'),
    (match) => '.',
  );

  return '${negative ? '-' : ''}$formatted $currency';
}

String formatCurrencyInput(dynamic value) {
  final digits = value.toString().replaceAll(RegExp(r'[^0-9]'), '');
  if (digits.isEmpty) return '';
  final normalized = digits.replaceFirst(RegExp(r'^0+(?=\d)'), '');
  return normalized.replaceAllMapped(
    RegExp(r'\B(?=(\d{3})+(?!\d))'),
    (_) => '.',
  );
}

class CurrencyInputFormatter extends TextInputFormatter {
  @override
  TextEditingValue formatEditUpdate(
    TextEditingValue oldValue,
    TextEditingValue newValue,
  ) {
    final formatted = formatCurrencyInput(newValue.text);
    return TextEditingValue(
      text: formatted,
      selection: TextSelection.collapsed(offset: formatted.length),
    );
  }
}

String moneyInVietnameseWords(int number) {
  if (number <= 0) return '';
  return '${numberToVietnameseWords(number)} đồng';
}

String numberToVietnameseWords(int number) {
  if (number == 0) return 'Không';
  const units = ['', 'nghìn', 'triệu', 'tỷ', 'nghìn tỷ', 'triệu tỷ', 'tỷ tỷ'];
  final groups = <int>[];
  var remaining = number;
  while (remaining > 0) {
    groups.add(remaining % 1000);
    remaining ~/= 1000;
  }

  final parts = <String>[];
  for (var index = groups.length - 1; index >= 0; index--) {
    final group = groups[index];
    if (group == 0) continue;
    final forceHundreds = index < groups.length - 1 && group < 100;
    final words = _readThreeDigits(group, forceHundreds: forceHundreds);
    parts.add(units[index].isEmpty ? words : '$words ${units[index]}');
  }
  final result = parts.join(' ').replaceAll(RegExp(r'\s+'), ' ').trim();
  return '${result[0].toUpperCase()}${result.substring(1)}';
}

String _readThreeDigits(int number, {required bool forceHundreds}) {
  const digits = [
    'không',
    'một',
    'hai',
    'ba',
    'bốn',
    'năm',
    'sáu',
    'bảy',
    'tám',
    'chín',
  ];
  final hundred = number ~/ 100;
  final ten = (number % 100) ~/ 10;
  final unit = number % 10;
  final words = <String>[];

  if (hundred > 0 || forceHundreds) {
    words.add('${digits[hundred]} trăm');
  }
  if (ten > 1) {
    words.add('${digits[ten]} mươi');
  } else if (ten == 1) {
    words.add('mười');
  } else if (unit > 0 && (hundred > 0 || forceHundreds)) {
    words.add('lẻ');
  }
  if (unit > 0) {
    if (unit == 1 && ten > 1) {
      words.add('mốt');
    } else if (unit == 5 && ten > 0) {
      words.add('lăm');
    } else if (unit == 4 && ten > 1) {
      words.add('tư');
    } else {
      words.add(digits[unit]);
    }
  }
  return words.join(' ');
}

String shortDate(dynamic value) {
  final date = DateTime.tryParse(value?.toString() ?? '')?.toLocal();
  if (date == null) return '—';

  String two(int value) => value.toString().padLeft(2, '0');

  return '${two(date.day)}/${two(date.month)}/${date.year}';
}

String dateTimeText(dynamic value) {
  final date = DateTime.tryParse(value?.toString() ?? '')?.toLocal();
  if (date == null) return '—';

  String two(int value) => value.toString().padLeft(2, '0');

  return '${two(date.hour)}:${two(date.minute)} · ${two(date.day)}/${two(date.month)}/${date.year}';
}

String removeVietnameseDiacritics(String value) {
  const accents = <String>[
    'àáạảãâầấậẩẫăằắặẳẵ',
    'èéẹẻẽêềếệểễ',
    'ìíịỉĩ',
    'òóọỏõôồốộổỗơờớợởỡ',
    'ùúụủũưừứựửữ',
    'ỳýỵỷỹ',
    'đ',
    'ÀÁẠẢÃÂẦẤẬẨẪĂẰẮẶẲẴ',
    'ÈÉẸẺẼÊỀẾỆỂỄ',
    'ÌÍỊỈĨ',
    'ÒÓỌỎÕÔỒỐỘỔỖƠỜỚỢỞỠ',
    'ÙÚỤỦŨƯỪỨỰỬỮ',
    'ỲÝỴỶỸ',
    'Đ',
  ];
  const replacements = <String>[
    'a',
    'e',
    'i',
    'o',
    'u',
    'y',
    'd',
    'A',
    'E',
    'I',
    'O',
    'U',
    'Y',
    'D',
  ];

  var result = value;
  for (var group = 0; group < accents.length; group++) {
    for (final character in accents[group].split('')) {
      result = result.replaceAll(character, replacements[group]);
    }
  }
  return result.trim().replaceAll(RegExp(r'\s+'), ' ');
}
