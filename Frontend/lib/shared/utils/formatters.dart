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
