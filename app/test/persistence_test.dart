import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:omnitun/engine/engine.dart';
import 'package:omnitun/engine/mock_engine.dart';
import 'package:omnitun/state/providers.dart';
import 'package:omnitun/state/store.dart';

ProfileInfo _profile(String id) => ProfileInfo.fromJson({
      'id': id,
      'name': 'P-$id',
      'protocol': 'trojan',
      'server': '$id.example.com',
      'port': 443,
      'password': 'secret-$id',
    });

ProviderContainer _container(AppStore store) => ProviderContainer(overrides: [
      storeProvider.overrideWithValue(store),
      engineProvider.overrideWithValue(MockEngine()),
    ]);

void main() {
  test('profiles, selection, order and settings survive a restart', () {
    final store = AppStore.memory();
    final first = _container(store);
    first.read(profilesProvider.notifier).addAll([_profile('a'), _profile('b'), _profile('c')]);
    first.read(profilesProvider.notifier).reorder(2, 0); // c, a, b
    first.read(selectedProfileIdProvider.notifier).select('b');
    first.read(settingsProvider.notifier).update((s) => s.copyWith(remoteDns: 'tls://1.1.1.1', autoFailover: false));
    first.dispose();

    final second = _container(store);
    addTearDown(second.dispose);
    final profiles = second.read(profilesProvider);
    expect(profiles.map((p) => p.id), ['c', 'a', 'b']);
    // The full core profile is kept, not just what the UI displays.
    expect(profiles.first.raw['password'], 'secret-c');
    expect(second.read(selectedProfileProvider)?.id, 'b');
    expect(second.read(settingsProvider).remoteDns, 'tls://1.1.1.1');
    expect(second.read(settingsProvider).autoFailover, isFalse);
  });

  test('duplicate imports are skipped and corrupt entries ignored', () {
    final store = AppStore.memory({
      'profiles': [
        {'id': 'x', 'name': 'X', 'protocol': 'vless', 'server': 'x.example.com', 'port': 443},
        {'garbage': true},
      ],
    });
    final c = _container(store);
    addTearDown(c.dispose);
    expect(c.read(profilesProvider).map((p) => p.id), ['x']);
    expect(c.read(profilesProvider.notifier).addAll([_profile('x'), _profile('y')]), 1);
  });

  test('settings loaded from an older version keep defaults for new keys', () {
    final s = SettingsState.fromJson({'tun_mode': true}, const SettingsState());
    expect(s.tunMode, isTrue);
    expect(s.remoteDns, SettingsState.defaultRemoteDns);
    expect(s.autoFailover, isTrue);
  });
}
