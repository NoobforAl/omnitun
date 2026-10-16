import 'package:flutter_test/flutter_test.dart';
import 'package:omnitun/ui/screens/home_screen.dart';

void main() {
  test('an address is shown in part only', () {
    expect(maskIp('147.78.241.51'), '147.78.x.x');
    expect(maskIp('2a07:d883:703:2::e018'), '2a07:d883:…');
    expect(() => maskIp('::1'), returnsNormally); // odd input never throws
    expect(maskIp('not an ip'), 'not an ip');
  });
}
