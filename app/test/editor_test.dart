import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:omnitun/engine/mock_engine.dart';
import 'package:omnitun/main.dart';
import 'package:omnitun/state/providers.dart';
import 'package:omnitun/state/store.dart';
import 'package:qr_flutter/qr_flutter.dart';

Future<ProviderContainer> _start(WidgetTester tester, AppStore store) async {
  tester.view.physicalSize = const Size(800, 1600);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(
    ProviderScope(
      overrides: [storeProvider.overrideWithValue(store), engineProvider.overrideWithValue(MockEngine())],
      child: const OmnitunApp(),
    ),
  );
  await tester.pump();
  return ProviderScope.containerOf(tester.element(find.byType(OmnitunApp)));
}

Future<void> _openLocations(WidgetTester tester) async {
  await tester.tap(find.text('Location'));
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('add a server by hand: the core checks it, the app saves it', (tester) async {
    final store = AppStore.memory({
      'profiles': [
        {'id': 'x', 'name': 'Old', 'protocol': 'trojan', 'server': 'old.example.com', 'port': 443},
      ],
    });
    final c = await _start(tester, store);
    await _openLocations(tester);
    await tester.tap(find.byTooltip('Add servers'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Add a server by hand'));
    await tester.pumpAndSettle();
    expect(find.text('Trojan'), findsOneWidget);
    expect(find.text('Tor'), findsOneWidget);
    await tester.tap(find.text('Trojan'));
    await tester.pumpAndSettle();
    expect(find.text('New Trojan server'), findsOneWidget);

    // A missing password is refused with the core's message, and nothing is added.
    await tester.enterText(find.widgetWithText(TextField, 'Server *'), 'de1.example.com');
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();
    expect(find.text('Password is required'), findsOneWidget);
    expect(c.read(profilesProvider), hasLength(1));

    await tester.enterText(find.widgetWithText(TextField, 'Password *'), 'secret');
    await tester.enterText(find.widgetWithText(TextField, 'Name'), 'Berlin 1');
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();
    expect(c.read(profilesProvider).map((p) => p.name), ['Old', 'Berlin 1']);
    expect(find.text('New Trojan server'), findsNothing);
    expect(find.text('Add a server'), findsNothing); // back on the location list
    await tester.pump(const Duration(milliseconds: 400));
    expect(store.read<List<dynamic>>('profiles'), hasLength(2)); // saved
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('edit keeps the place and group; the country can be set in the editor', (tester) async {
    final store = AppStore.memory({
      'profiles': [
        {'id': 'a', 'name': 'A', 'protocol': 'trojan', 'server': 'a.example.com', 'port': 443, 'password': 'pw'},
        {
          'id': 'b',
          'name': 'B',
          'protocol': 'trojan',
          'server': 'b.example.com',
          'port': 443,
          'password': 'pw',
          'group': 'sub1',
        },
      ],
    });
    final c = await _start(tester, store);
    await _openLocations(tester);
    await tester.tap(find.byTooltip('Show servers').first);
    await tester.pumpAndSettle();
    // Both servers have no known country, so they are under "Other": edit the second.
    await tester.tap(find.byTooltip('More').last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Edit'));
    await tester.pumpAndSettle();
    expect(find.text('Edit server'), findsOneWidget);
    expect(find.textContaining('replaced when it refreshes'), findsOneWidget);

    await tester.enterText(find.widgetWithText(TextField, 'Name'), 'B renamed');
    await tester.tap(find.text('Country'));
    await tester.pumpAndSettle();
    await tester.enterText(find.widgetWithText(TextField, 'Search'), 'france');
    await tester.pumpAndSettle();
    await tester.tap(find.text('France'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    final profiles = c.read(profilesProvider);
    expect(profiles.map((p) => p.name), ['A', 'B renamed']); // same place
    expect(profiles.last.group, 'sub1'); // same group
    expect(c.read(countryOverridesProvider)['b'], 'FR');
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('share shows the link as a QR code; a server without a link says so', (tester) async {
    final store = AppStore.memory({
      'profiles': [
        {'id': 'a', 'name': 'A', 'protocol': 'trojan', 'server': 'a.example.com', 'port': 443, 'password': 'pw'},
        {'id': 't', 'name': 'Tor', 'protocol': 'tor', 'server': '', 'port': 0},
      ],
    });
    await _start(tester, store);
    await _openLocations(tester);
    await tester.tap(find.byTooltip('Show servers').first);
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('More').first);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Share link or QR'));
    await tester.pumpAndSettle();
    expect(find.byType(QrImageView), findsOneWidget);
    expect(find.textContaining('trojan://pw@a.example.com:443'), findsOneWidget);
    await tester.pageBack();
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('More').last);
    await tester.pumpAndSettle();
    expect(find.text('Bridge settings'), findsOneWidget);
    await tester.tap(find.text('Share link or QR'));
    await tester.pumpAndSettle();
    expect(find.textContaining('cannot be shared as a link'), findsOneWidget);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('the Tor editor has the bridge choice and bridge lines', (tester) async {
    final store = AppStore.memory({
      'profiles': [
        {'id': 't', 'name': 'Tor', 'protocol': 'tor', 'server': '', 'port': 0},
      ],
    });
    await _start(tester, store);
    await _openLocations(tester);
    await tester.tap(find.byTooltip('More'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Bridge settings'));
    await tester.pumpAndSettle();
    expect(find.text('Tor bridges'), findsOneWidget);
    expect(find.text('Bridges'), findsOneWidget);
    expect(find.text('Bridge lines'), findsOneWidget);
    expect(find.text('Paste'), findsOneWidget);
    expect(find.text('File'), findsOneWidget);
    expect(find.text('Country'), findsNothing); // Tor has no country
    await tester.pumpWidget(const SizedBox());
  });
}
