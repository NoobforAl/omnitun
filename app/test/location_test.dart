import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:omnitun/engine/engine.dart';
import 'package:omnitun/engine/mock_engine.dart';
import 'package:omnitun/main.dart';
import 'package:omnitun/state/countries.dart';
import 'package:omnitun/state/location.dart';
import 'package:omnitun/state/providers.dart';
import 'package:omnitun/state/store.dart';

ProfileInfo _p(
  String name, {
  String server = '203.0.113.1',
  String protocol = 'trojan',
  int? ms,
  String? id,
  String group = 'default',
}) => ProfileInfo.fromJson({
  'id': id ?? name,
  'name': name,
  'protocol': protocol,
  'server': server,
  'port': 443,
  'group': group,
}).copyWith(latencyMs: ms);

String? _country(String name, {String server = '203.0.113.1', String protocol = 'trojan'}) =>
    guessCountry(_p(name, server: server, protocol: protocol));

void main() {
  group('guessCountry', () {
    test('flag emoji, codes and names in the server name', () {
      expect(_country('🇩🇪 Berlin 01'), 'DE');
      expect(_country('DE-Berlin-01'), 'DE');
      expect(_country('[us] fast'), 'US');
      expect(_country('(NL) Amsterdam'), 'NL');
      expect(_country('Germany #3'), 'DE');
      expect(_country('United Kingdom London'), 'GB');
      expect(_country('my UK vps'), 'GB');
      expect(_country('Frankfurt'), 'DE');
      expect(_country('Sweden'), 'SE');
    });

    test('the longest name wins and only whole words count', () {
      expect(_country('DR Congo 1'), 'CD');
      expect(_country('Nigeria'), 'NG'); // not Niger
      expect(_country('Singapore-2'), 'SG');
      expect(_country('Portugal'), 'PT');
      expect(_country('Hungary'), 'HU'); // contains "hun" but no other name
    });

    test('the address tells when the name does not', () {
      expect(_country('fast', server: 'de1.example.com'), 'DE');
      expect(_country('fast', server: 'us-nyc.example.com'), 'US');
      expect(_country('fast', server: 'vpn.example.nl'), 'NL');
      expect(_country('fast', server: 'vpn.example.co.uk'), 'GB');
    });

    test('generic domains and plain addresses say nothing', () {
      expect(_country('Home VPN', server: 'vpn.example.io'), isNull);
      expect(_country('Home VPN', server: 'vpn.example.com'), isNull);
      expect(_country('Home VPN', server: '203.0.113.5'), isNull);
      expect(_country('my-server', server: 'host.example.me'), isNull);
      expect(_country('PC-dev'), isNull);
    });

    test('Tor has no country', () {
      expect(_country('Tor Germany', protocol: 'tor'), isNull);
    });

    test('flags convert both ways', () {
      expect(codeOfFlag('x 🇯🇵 y'), 'JP');
      expect(flagEmoji('JP'), '🇯🇵');
      expect(countryNames['IR'], 'Iran');
      expect(countryNames.length, greaterThan(190));
    });
  });

  group('planConnection', () {
    final de1 = _p('de1', ms: 200);
    final de2 = _p('de2', ms: 80);
    final nl = _p('nl', ms: 50);
    final dead = _p('dead', ms: -1);
    final untested = _p('untested');
    final tor = _p('tor', protocol: 'tor');
    final all = [de1, de2, nl, dead, untested, tor];
    String? countryOf(ProfileInfo p) => switch (p.id) {
      'de1' || 'de2' => 'DE',
      'nl' => 'NL',
      _ => null,
    };

    test('Auto starts on the fastest and keeps unreachable and Tor for last or never', () {
      final plan = planConnection(all, null, countryOf)!;
      expect(plan.primary.id, 'nl');
      expect(plan.fallbacks.map((p) => p.id), ['de2', 'de1', 'untested', 'dead']);
    });

    test('a country starts on its fastest server, then the others', () {
      final plan = planConnection(all, countryChoice('DE'), countryOf)!;
      expect(plan.primary.id, 'de2');
      expect(plan.fallbacks.map((p) => p.id), ['de1', 'nl', 'untested', 'dead']);
    });

    test('one server keeps the order of its group, and Tor runs only when picked', () {
      final plan = planConnection(all, 'de1', countryOf)!;
      expect(plan.primary.id, 'de1');
      expect(plan.fallbacks.map((p) => p.id), ['de2', 'nl', 'dead', 'untested', 'tor']);
      expect(planConnection(all, 'tor', countryOf)!.primary.id, 'tor');
    });

    test('a profile that needs a password does not start on its own', () {
      final needs = ProfileInfo.fromJson({
        'id': 'ovpn',
        'name': 'ovpn',
        'protocol': 'openvpn',
        'server': 'x.example.com',
        'port': 1194,
        'openvpn': {'needs_auth': true},
      }).copyWith(latencyMs: 5);
      expect(planConnection([needs, nl], null, countryOf)!.primary.id, 'nl');
      expect(planConnection([needs], null, countryOf)!.primary.id, 'ovpn'); // nothing else to try
    });

    test('a choice that no longer exists falls back to Auto; no profiles, no plan', () {
      expect(planConnection(all, 'gone', countryOf)!.primary.id, 'nl');
      expect(planConnection(all, countryChoice('FR'), countryOf)!.primary.id, 'nl');
      expect(planConnection(const [], null, countryOf), isNull);
    });
  });

  test('groupByCountry sorts countries by name, servers by speed, unknown last', () {
    final groups = groupByCountry(
      [_p('a', ms: 90), _p('b', ms: 20), _p('c'), _p('t', protocol: 'tor'), _p('d', ms: -1)],
      (p) => switch (p.id) {
        'a' || 'b' => 'NL',
        'd' => 'AT',
        _ => null,
      },
      (c) => countryNames[c] ?? c,
    );
    expect(groups.map((g) => g.code), ['AT', 'NL', null]);
    expect(groups[1].profiles.map((p) => p.id), ['b', 'a']);
    expect(groups[1].bestLatency, 20);
    expect(groups[0].unreachable, isTrue);
    expect(groups[2].profiles.map((p) => p.id), ['c']);
  });

  testWidgets('the location list picks Auto, a country or a server; Back returns home', (tester) async {
    final store = AppStore.memory({
      // These tests are about the choice of server, not about lookups of the exit.
      'settings': {'show_exit': false},
      'profiles': [
        {'id': 'd1', 'name': 'DE-Berlin-01', 'protocol': 'trojan', 'server': 'a.example.com', 'port': 443},
        {'id': 'd2', 'name': 'DE-Frankfurt-02', 'protocol': 'vless', 'server': 'b.example.com', 'port': 443},
        {'id': 'n1', 'name': 'Netherlands 1', 'protocol': 'trojan', 'server': 'c.example.com', 'port': 443},
        {'id': 'x1', 'name': 'Home VPN', 'protocol': 'trojan', 'server': 'd.example.com', 'port': 443},
        {'id': 't', 'name': 'Tor', 'protocol': 'tor', 'server': '', 'port': 0},
      ],
    });
    await tester.pumpWidget(
      ProviderScope(
        overrides: [storeProvider.overrideWithValue(store), engineProvider.overrideWithValue(MockEngine())],
        child: const OmnitunApp(),
      ),
    );
    await tester.pump();
    final container = ProviderScope.containerOf(tester.element(find.byType(OmnitunApp)));

    expect(find.text('Automatic'), findsOneWidget);
    await tester.tap(find.text('Location'));
    await tester.pumpAndSettle();
    expect(find.text('Select location'), findsOneWidget);
    expect(find.text('Germany'), findsOneWidget);
    expect(find.text('2 servers'), findsOneWidget);
    expect(find.text('Netherlands'), findsOneWidget);
    expect(find.text('Other'), findsOneWidget);
    expect(find.text('Tor'), findsWidgets);
    expect(find.text('DE-Berlin-01'), findsNothing); // servers are hidden until a country opens

    // Open Germany and pick one server.
    await tester.tap(find.byTooltip('Show servers').first);
    await tester.pumpAndSettle();
    await tester.tap(find.text('DE-Frankfurt-02'));
    await tester.pumpAndSettle();
    expect(container.read(selectedProfileIdProvider), 'd2');
    expect(find.text('Select location'), findsNothing); // back on the home screen
    expect(find.text('DE-Frankfurt-02'), findsOneWidget);

    // Pick a whole country.
    await tester.tap(find.text('Location'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Netherlands'));
    await tester.pumpAndSettle();
    expect(container.read(selectedProfileIdProvider), countryChoice('NL'));
    expect(find.text('Any server in this country'), findsOneWidget);
    expect(container.read(selectedProfileProvider)?.id, 'n1');

    // And back to Automatic.
    await tester.tap(find.text('Location'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Automatic'));
    await tester.pumpAndSettle();
    expect(container.read(selectedProfileIdProvider), isNull);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('choosing another country while connected reconnects there', (tester) async {
    final store = AppStore.memory({
      // These tests are about the choice of server, not about lookups of the exit.
      'settings': {'show_exit': false},
      'profiles': [
        {'id': 'd1', 'name': 'Germany 1', 'protocol': 'trojan', 'server': 'a.example.com', 'port': 443},
        {'id': 'n1', 'name': 'Netherlands 1', 'protocol': 'trojan', 'server': 'c.example.com', 'port': 443},
      ],
      'selected': 'd1',
    });
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          storeProvider.overrideWithValue(store),
          engineProvider.overrideWithValue(MockEngine(connectDelay: Duration.zero)),
        ],
        child: const OmnitunApp(),
      ),
    );
    await tester.pump();
    final container = ProviderScope.containerOf(tester.element(find.byType(OmnitunApp)));

    await tester.tap(find.text('Connect'));
    await tester.pump(const Duration(milliseconds: 50));
    expect(container.read(connectionProvider).status, ConnStatus.connected);
    expect(container.read(connectionProvider).activeProfileId, 'd1');

    await tester.tap(find.text('Location'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Netherlands'));
    await tester.pump(const Duration(seconds: 1));
    await tester.pump(const Duration(seconds: 1));
    expect(container.read(connectionProvider).status, ConnStatus.connected);
    expect(container.read(connectionProvider).activeProfileId, 'n1');
    await tester.tap(find.text('Disconnect'));
    // Also lets the store save (it waits 300 ms after a change).
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pumpWidget(const SizedBox());
  });

  test('the guess can be corrected by hand and is remembered', () {
    final store = AppStore.memory({
      // These tests are about the choice of server, not about lookups of the exit.
      'settings': {'show_exit': false},
      'profiles': [
        {'id': 'x1', 'name': 'Home VPN', 'protocol': 'trojan', 'server': 'd.example.com', 'port': 443},
      ],
    });
    final c1 = ProviderContainer(
      overrides: [storeProvider.overrideWithValue(store), engineProvider.overrideWithValue(MockEngine())],
    );
    final p = c1.read(profilesProvider).single;
    expect(c1.read(profileCountryProvider)(p), isNull);
    c1.read(countryOverridesProvider.notifier).set('x1', 'FR');
    expect(c1.read(profileCountryProvider)(p), 'FR');
    c1.dispose();

    final c2 = ProviderContainer(
      overrides: [storeProvider.overrideWithValue(store), engineProvider.overrideWithValue(MockEngine())],
    );
    addTearDown(c2.dispose);
    expect(c2.read(profileCountryProvider)(c2.read(profilesProvider).single), 'FR');
    c2.read(countryOverridesProvider.notifier).set('x1', null);
    expect(c2.read(profileCountryProvider)(c2.read(profilesProvider).single), isNull);
  });
}
