import 'package:flutter_test/flutter_test.dart';
import 'package:omnitun/state/countries.dart';
import 'package:omnitun/state/geo_data.dart';

void main() {
  test('every country the app names can be found on the globe', () {
    final missing = countryNames.keys.where((c) => !countryCenters.containsKey(c)).toList()..sort();
    // Places Natural Earth's 110 m map has no polygon for.
    const known = <String>{};
    expect(missing.toSet().difference(known), isEmpty, reason: 'no globe position for $missing');
  });

  test('the centres are real places and the land grid has the size it says', () {
    for (final e in countryCenters.entries) {
      expect(e.value.$1.abs() <= 90 && e.value.$2.abs() <= 180, isTrue, reason: e.key);
    }
    expect(countryCenters['DE']!.$1, closeTo(51, 3));
    expect(countryCenters['DE']!.$2, closeTo(10, 3));
    expect(countryCenters['US']!.$2, lessThan(-90)); // the contiguous states, not Alaska or Hawaii
    expect(landGridWidth * landGridHeight / 8, lessThanOrEqualTo(landGridBase64.length * 3 / 4));
  });
}
