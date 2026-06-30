import 'package:flutter_test/flutter_test.dart';
import 'package:nf_bank/main.dart';

void main() {
  testWidgets('shows login screen for signed-out users', (tester) async {
    await tester.pumpWidget(const NfBankApp(isLoggedIn: false));
    expect(find.text('Chào mừng trở lại'), findsOneWidget);
    expect(find.text('Đăng nhập'), findsOneWidget);
  });
}
