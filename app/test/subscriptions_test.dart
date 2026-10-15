import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:omnitun/engine/engine.dart';
import 'package:omnitun/engine/mock_engine.dart';
import 'package:omnitun/state/providers.dart';
import 'package:omnitun/state/store.dart';
import 'package:omnitun/state/subscriptions.dart';

ProfileInfo _p(String id) =>
    ProfileInfo.fromJson({'id': id, 'name': id.toUpperCase(), 'protocol': 'trojan', 'server': '$id.x', 'port': 443});

/// Mock engine whose subscription responses the test controls.
class SubEngine extends MockEngine {
  List<ProfileInfo> next = [];
  Object? failWith;
  Map<String, dynamic>? lastRequest;

  @override
  Future<SubscriptionResult> fetchSubscription(Map<String, dynamic> request) async {
    lastRequest = request;
    if (failWith != null) throw failWith!;
    return SubscriptionResult(next, const [], const SubscriptionInfo(total: 1000, download: 250, title: 'Prov'));
  }
}

void main() {
  late SubEngine engine;
  late AppStore store;
  late ProviderContainer c;

  ProviderContainer make() => ProviderContainer(overrides: [
        storeProvider.overrideWithValue(store),
        engineProvider.overrideWithValue(engine),
      ]);

  setUp(() {
    engine = SubEngine();
    store = AppStore.memory();
    c = make();
  });
  tearDown(() => c.dispose());

  test('adding a subscription stores it and groups its profiles', () async {
    c.read(profilesProvider.notifier).addAll([_p('manual')]);
    engine.next = [_p('a'), _p('b')];
    final n = await c.read(subscriptionsProvider.notifier).add(
          url: 'https://prov.example/sub',
          auth: const SubscriptionAuth(type: 'bearer', token: 't'),
        );
    expect(n, 2);
    expect(engine.lastRequest, {
      'url': 'https://prov.example/sub',
      'auth': {'type': 'bearer', 'token': 't'},
    });
    final sub = c.read(subscriptionsProvider).single;
    expect(sub.name, 'Prov'); // provider title when no name is given
    expect(sub.info.used, 250);
    final profiles = c.read(profilesProvider);
    expect(profiles.map((p) => p.id), ['manual', 'a', 'b']);
    expect(profiles.where((p) => p.group == sub.id).length, 2);

    // Survives a restart, including the group assignment.
    c.dispose();
    c = make();
    expect(c.read(subscriptionsProvider).single.url, 'https://prov.example/sub');
    expect(c.read(profilesProvider).where((p) => p.group == sub.id).map((p) => p.id), ['a', 'b']);
  });

  test('refresh replaces the group in place and keeps the selection', () async {
    engine.next = [_p('a'), _p('b')];
    await c.read(subscriptionsProvider.notifier).add(url: 'https://prov.example/sub');
    c.read(profilesProvider.notifier).addAll([_p('manual')]);
    c.read(selectedProfileIdProvider.notifier).select('b');
    final id = c.read(subscriptionsProvider).single.id;

    engine.next = [_p('b'), _p('c')];
    expect(await c.read(subscriptionsProvider.notifier).refresh(id), 2);
    expect(c.read(profilesProvider).map((p) => p.id), ['b', 'c', 'manual']);
    expect(c.read(selectedProfileProvider)?.id, 'b');
  });

  test('a failed refresh keeps profiles and records the error', () async {
    engine.next = [_p('a')];
    await c.read(subscriptionsProvider.notifier).add(url: 'https://prov.example/sub');
    final id = c.read(subscriptionsProvider).single.id;

    engine.failWith = StateError('subscription authentication failed (HTTP 401)');
    await expectLater(c.read(subscriptionsProvider.notifier).refresh(id), throwsStateError);
    expect(c.read(subscriptionsProvider).single.lastError, contains('HTTP 401'));
    expect(c.read(profilesProvider).map((p) => p.id), ['a']);

    engine.failWith = null;
    await c.read(subscriptionsProvider.notifier).refresh(id);
    expect(c.read(subscriptionsProvider).single.lastError, isNull);
  });

  test('a failed add does not create a subscription', () async {
    engine.failWith = StateError('subscription server returned HTTP 404');
    await expectLater(c.read(subscriptionsProvider.notifier).add(url: 'https://x.example/'), throwsStateError);
    expect(c.read(subscriptionsProvider), isEmpty);
  });

  test('removing a subscription removes only its profiles', () async {
    c.read(profilesProvider.notifier).addAll([_p('manual')]);
    engine.next = [_p('a')];
    await c.read(subscriptionsProvider.notifier).add(url: 'https://prov.example/sub');
    c.read(subscriptionsProvider.notifier).remove(c.read(subscriptionsProvider).single.id);
    expect(c.read(profilesProvider).map((p) => p.id), ['manual']);
  });

  test('reordering inside a group leaves other groups alone', () async {
    c.read(profilesProvider.notifier).addAll([_p('m1')]);
    engine.next = [_p('a'), _p('b'), _p('c')];
    await c.read(subscriptionsProvider.notifier).add(url: 'https://prov.example/sub');
    c.read(profilesProvider.notifier).addAll([_p('m2')]);
    final id = c.read(subscriptionsProvider).single.id;
    c.read(profilesProvider.notifier).reorderInGroup(id, 2, 0); // c, a, b
    expect(c.read(profilesProvider).map((p) => p.id), ['m1', 'c', 'a', 'b', 'm2']);
  });

  test('isDue follows the provider interval', () {
    final fresh = Subscription(id: 's', name: 'n', url: 'u', updatedAt: DateTime.now());
    expect(fresh.isDue, isFalse);
    final old = Subscription(
      id: 's',
      name: 'n',
      url: 'u',
      info: const SubscriptionInfo(updateIntervalHours: 2),
      updatedAt: DateTime.now().subtract(const Duration(hours: 3)),
    );
    expect(old.isDue, isTrue);
  });
}
