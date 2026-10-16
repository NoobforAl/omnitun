import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:omnitun/engine/engine.dart';
import 'package:omnitun/engine/mock_engine.dart';
import 'package:omnitun/main.dart';
import 'package:omnitun/state/providers.dart';
import 'package:omnitun/state/store.dart';

/// Counts how often it is asked where the tunnel exits.
class ExitEngine extends MockEngine {
  ExitEngine() : super(connectDelay: Duration.zero);
  int asked = 0;
  Object? failWith;

  @override
  Future<ExitInfo> exitInfo() async {
    asked++;
    if (failWith != null) throw failWith!;
    return exit;
  }
}

Future<ProviderContainer> _start(WidgetTester tester, ExitEngine engine, Map<String, dynamic> data) async {
  tester.view.physicalSize = const Size(800, 1600);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(
    ProviderScope(
      overrides: [storeProvider.overrideWithValue(AppStore.memory(data)), engineProvider.overrideWithValue(engine)],
      child: const OmnitunApp(),
    ),
  );
  await tester.pumpAndSettle();
  return ProviderScope.containerOf(tester.element(find.byType(OmnitunApp)));
}

Map<String, dynamic> _data({bool showExit = true, String protocol = 'trojan'}) => {
  'onboarded': true,
  'settings': {'show_exit': showExit},
  'profiles': [
    {'id': 'a', 'name': 'Home VPN', 'protocol': protocol, 'server': '203.0.113.9', 'port': 443},
  ],
};

void main() {
  testWidgets('after connecting, the exit address and country are shown and the country is remembered', (tester) async {
    final engine = ExitEngine();
    final c = await _start(tester, engine, _data());
    expect(c.read(profileCountryProvider)(c.read(profilesProvider).single), isNull); // the name says nothing

    await tester.tap(find.text('Connect'));
    await tester.pump(const Duration(milliseconds: 100));
    await tester.pump(const Duration(milliseconds: 100));
    expect(engine.asked, 1);
    expect(c.read(exitProvider)?.ip, '203.0.113.77');
    expect(find.text('IP 203.0.x.x'), findsOneWidget);
    expect(find.text('Connected to Netherlands'), findsOneWidget); // the country that was seen
    // Remembered for the profile, so the location list can use it.
    expect(c.read(learnedCountriesProvider)['a'], 'NL');
    expect(c.read(profileCountryProvider)(c.read(profilesProvider).single), 'NL');

    // It is not asked again while the same connection stays up.
    await tester.pump(const Duration(seconds: 3));
    expect(engine.asked, 1);

    await tester.tap(find.text('Disconnect'));
    await tester.pump(const Duration(milliseconds: 400));
    expect(c.read(exitProvider), isNull);
    expect(find.textContaining('IP 203'), findsNothing);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('switched off, nothing is asked (it is on by default)', (tester) async {
    final engine = ExitEngine();
    final c = await _start(tester, engine, _data(showExit: false));
    expect(c.read(settingsProvider).showExit, isFalse);
    expect(const SettingsState().showExit, isTrue); // the default, for the "you are here" globe
    await tester.tap(find.text('Connect'));
    await tester.pump(const Duration(milliseconds: 100));
    await tester.pump(const Duration(milliseconds: 100));
    expect(engine.asked, 0);
    expect(c.read(exitProvider), isNull);
    await tester.tap(find.text('Disconnect'));
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('a failed lookup shows nothing and does not disturb the connection', (tester) async {
    final engine = ExitEngine()..failWith = EngineException('LOOKUP_FAILED', 'no answer');
    final c = await _start(tester, engine, _data());
    await tester.tap(find.text('Connect'));
    await tester.pump(const Duration(milliseconds: 100));
    await tester.pump(const Duration(milliseconds: 100));
    expect(engine.asked, 1);
    expect(c.read(exitProvider), isNull);
    expect(c.read(connectionProvider).status, ConnStatus.connected);
    expect(c.read(learnedCountriesProvider), isEmpty);
    await tester.tap(find.text('Disconnect'));
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('Tor shows the exit address but never teaches its profile a country', (tester) async {
    final engine = ExitEngine();
    final c = await _start(tester, engine, _data(protocol: 'tor'));
    await tester.tap(find.text('Connect'));
    await tester.pump(const Duration(milliseconds: 100));
    await tester.pump(const Duration(milliseconds: 100));
    expect(find.text('IP 203.0.x.x'), findsOneWidget);
    expect(c.read(learnedCountriesProvider), isEmpty);
    await tester.tap(find.text('Disconnect'));
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('a country the user set by hand wins over the one that was seen', (tester) async {
    final engine = ExitEngine();
    final c = await _start(tester, engine, _data());
    c.read(countryOverridesProvider.notifier).set('a', 'FR');
    await tester.tap(find.text('Connect'));
    await tester.pump(const Duration(milliseconds: 100));
    await tester.pump(const Duration(milliseconds: 100));
    expect(c.read(profileCountryProvider)(c.read(profilesProvider).single), 'FR');
    await tester.tap(find.text('Disconnect'));
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pumpWidget(const SizedBox());
  });
}
