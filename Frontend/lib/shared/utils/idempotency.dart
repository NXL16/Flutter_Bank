import 'dart:math';

String createIdempotencyKey() {
  final random = Random.secure();
  final bytes = List<int>.generate(16, (_) => random.nextInt(256));
  final suffix = bytes
      .map((value) => value.toRadixString(16).padLeft(2, '0'))
      .join();
  return '${DateTime.now().microsecondsSinceEpoch}:$suffix';
}
