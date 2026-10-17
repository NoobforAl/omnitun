import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:omnitun/engine/engine.dart';
import 'package:omnitun/engine/mock_engine.dart';
import 'package:omnitun/state/providers.dart';
import 'package:omnitun/state/store.dart';

ProfileInfo _profile(String id) => ProfileInfo.fromJson({
      'id': id,
      'name': 'P-$id',
      'protocol': 'vless',
      'server': '$id.example.com',
      'port': 443,
    });

ProviderContainer _container(AppStore store) {
  final c = ProviderContainer(overrides: [
    storeProvider.overrideWithValue(store),
    engineProvider.overrideWithValue(MockEngine()),
  ]);
  addTearDown(c.dispose);
  return c;
}

void main() {
  test('a new install starts on Tor with Snowflake, not on anything the user imports', () async {
    final c = _container(AppStore.memory());
    expect(await c.read(defaultProfileIdProvider.notifier).ensureTorDefaults(), 3);
    expect(c.read(profilesProvider).every((p) => p.protocol == 'tor'), isTrue);
    expect(torKind(c.read(defaultProfileProvider)!), 'snowflake');
    expect(c.read(selectedProfileProvider)?.protocol, 'tor');
    expect(torKind(c.read(selectedProfileProvider)!), 'snowflake');
    // Only once: the user may delete them.
    c.read(profilesProvider.notifier).remove(c.read(profilesProvider).first.id);
    expect(await c.read(defaultProfileIdProvider.notifier).ensureTorDefaults(), 0);
  });

  test('an existing user keeps their list and choice; Restore adds Tor without changing the choice', () async {
    final c = _container(AppStore.memory());
    c.read(profilesProvider.notifier).addAll([_profile('a')]);
    c.read(selectedProfileIdProvider.notifier).select('a');
    expect(await c.read(defaultProfileIdProvider.notifier).ensureTorDefaults(), 0);
    expect(c.read(profilesProvider).length, 1);
    expect(await c.read(defaultProfileIdProvider.notifier).ensureTorDefaults(force: true), 3);
    expect(c.read(selectedProfileIdProvider), 'a');
  });

  test('deleting the chosen server falls back to the default', () async {
    final c = _container(AppStore.memory());
    await c.read(defaultProfileIdProvider.notifier).ensureTorDefaults();
    c.read(profilesProvider.notifier).addAll([_profile('a'), _profile('b')]);
    c.read(selectedProfileIdProvider.notifier).select('a');
    c.read(profilesProvider.notifier).remove('a');
    expect(torKind(c.read(selectedProfileProvider)!), 'snowflake');

    // The user's own default wins over the built-in one, and survives a restart.
    c.read(defaultProfileIdProvider.notifier).set('b');
    expect(c.read(defaultProfileProvider)?.id, 'b');
    c.read(selectedProfileIdProvider.notifier).select('b');
    c.read(profilesProvider.notifier).remove('b');
    expect(torKind(c.read(selectedProfileProvider)!), 'snowflake'); // its server is gone: back to Tor
  });

  test('delete all removes every server but keeps Tor, and Auto never lands on nothing', () async {
    final c = _container(AppStore.memory());
    await c.read(defaultProfileIdProvider.notifier).ensureTorDefaults();
    c.read(profilesProvider.notifier).addAll([_profile('a'), _profile('b')]);
    c.read(selectedProfileIdProvider.notifier).select(null); // Auto
    c.read(profilesProvider.notifier).clearAll();
    expect(c.read(profilesProvider).map((p) => p.protocol).toSet(), {'tor'});
    expect(c.read(profilesProvider).length, 3);
    // Auto would have nothing to start on, so the default is chosen.
    expect(torKind(c.read(selectedProfileProvider)!), 'snowflake');
    expect(c.read(selectedProfileIdProvider), isNotNull);
  });

  test('a country choice is not replaced when servers are deleted', () async {
    final c = _container(AppStore.memory());
    await c.read(defaultProfileIdProvider.notifier).ensureTorDefaults();
    c.read(profilesProvider.notifier).addAll([_profile('a')]);
    c.read(selectedProfileIdProvider.notifier).select('country:DE');
    c.read(profilesProvider.notifier).remove('a');
    expect(c.read(selectedProfileIdProvider), 'country:DE');
  });
}
