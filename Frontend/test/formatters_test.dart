import 'package:flutter_test/flutter_test.dart';
import 'package:nf_bank/shared/utils/formatters.dart';

void main() {
  test('formats currency input with Vietnamese thousand separators', () {
    expect(formatCurrencyInput('10000'), '10.000');
    expect(formatCurrencyInput('12.345.678'), '12.345.678');
    expect(formatCurrencyInput('0005000000'), '5.000.000');
  });

  test('converts an amount to Vietnamese words', () {
    expect(moneyInVietnameseWords(10000), 'Mười nghìn đồng');
    expect(
      moneyInVietnameseWords(1250000),
      'Một triệu hai trăm năm mươi nghìn đồng',
    );
  });
}
