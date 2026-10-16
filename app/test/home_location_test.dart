import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:omnitun/engine/engine.dart';
import 'package:omnitun/engine/mock_engine.dart';
import 'package:omnitun/state/providers.dart';
import 'package:omnitun/state/store.dart';

ProviderContainer _container(Map<String, dynamic> data, List<int> calls) {
  final c = ProviderContainer(
    overrides: [
      storeProvider.overrideWithValue(AppStore.memory(data)),
      engineProvider.overrideWithValue(MockEngine(connectDelay: Duration.zero)),
      directLookupProvider.overrideWithValue(() async {
        calls.add(1);
        return const ExitInfo('203.0.113.5', 'IR');
      }),
    ],
  );
  addTearDown(c.dispose);
  return c;
}

void main() {
  test('where the device is comes from a direct lookup while not connected, and is remembered', () async {
    final calls = <int>[];
    final c = _container({'home_country': 'DE'}, calls);
    expect(c.read(homeProvider)?.country, 'DE'); // the last answer, at once
    await Future<void>.delayed(Duration.zero);
    await Future<void>.delayed(Duration.zero);
    expect(calls, hasLength(1));
    expect(c.read(homeProvider)?.country, 'IR');
    expect(c.read(storeProvider).read<String>('home_country'), 'IR');
  });

  test('nothing is looked up when the user switched address lookups off', () async {
    final calls = <int>[];
    final c = _container({
      'settings': {'show_exit': false},
    }, calls);
    expect(c.read(homeProvider), isNull);
    await Future<void>.delayed(Duration.zero);
    await Future<void>.delayed(Duration.zero);
    expect(calls, isEmpty);
    expect(c.read(homeProvider), isNull);
  });
}
