import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:omnitun/engine/mock_engine.dart';
import 'package:omnitun/l10n/l10n.dart';
import 'package:omnitun/main.dart';
import 'package:omnitun/state/providers.dart';
import 'package:omnitun/state/store.dart';
import 'package:omnitun/ui/screens/location_screen.dart';
import 'package:omnitun/ui/screens/profiles_screen.dart';
import 'package:omnitun/ui/screens/settings_screen.dart';
import 'package:omnitun/ui/theme.dart';

/// Logical size, text scale and whether the screen is expected to keep the status in view.
const _screens = <(String, Size, double)>[
  ('phone', Size(412, 915), 1.0),
  ('small phone', Size(320, 568), 1.0),
  ('small phone, large text', Size(320, 568), 1.3),
  ('short', Size(360, 427), 1.3),
  ('tablet', Size(800, 1280), 1.0),
];

void main() {
  for (final (name, size, scale) in _screens) {
    testWidgets('home screen on a $name: nothing overflows, status and button are in view', (tester) async {
      tester.view.physicalSize = size;
      tester.view.devicePixelRatio = 1;
      tester.platformDispatcher.textScaleFactorTestValue = scale;
      addTearDown(tester.view.reset);
      addTearDown(tester.platformDispatcher.clearAllTestValues);

      final store = AppStore.memory({
        'onboarded': true,
        'profiles': [
          {'id': 'a', 'name': 'A server with a rather long name', 'protocol': 'trojan', 'server': 'a.example.com', 'port': 443, 'password': 'pw'},
        ],
      });
      await tester.pumpWidget(ProviderScope(
        overrides: [storeProvider.overrideWithValue(store), engineProvider.overrideWithValue(MockEngine(connectDelay: Duration.zero))],
        child: const OmnitunApp(),
      ));
      await tester.pumpAndSettle();

      void check(String state) {
        expect(tester.takeException(), isNull, reason: '$state: layout overflow');
        final screen = Offset.zero & size;
        final status = tester.getRect(find.textContaining(RegExp('Not connected|Connected')).first);
        final button = tester.getRect(find.byType(InkWell).last);
        expect(screen.contains(button.center), isTrue, reason: '$state: the button is off screen');
        expect(status.bottom <= button.top, isTrue, reason: '$state: the status is behind the panel ($status vs $button)');
        expect(status.top >= 0, isTrue, reason: '$state: the status is above the screen');
      }

      check('not connected');
      await tester.tap(find.text('Connect'));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 50));
      await tester.pumpAndSettle();
      check('connected');
      await tester.tap(find.text('Disconnect'));
      await tester.pump(const Duration(milliseconds: 400));
      await tester.pumpWidget(const SizedBox());
    });
  }

  // The other screens, on the screens most likely to overflow.
  for (final (name, size, scale) in _screens.where((s) => s.$2.width < 400)) {
    for (final (screenName, screen) in <(String, Widget)>[
      ('settings', const SettingsScreen()),
      ('manage servers', const ProfilesScreen()),
      ('locations', const LocationScreen()),
    ]) {
      testWidgets('$screenName on a $name: nothing overflows', (tester) async {
        tester.view.physicalSize = size;
        tester.view.devicePixelRatio = 1;
        tester.platformDispatcher.textScaleFactorTestValue = scale;
        addTearDown(tester.view.reset);
        addTearDown(tester.platformDispatcher.clearAllTestValues);
        final store = AppStore.memory({
          'onboarded': true,
          'profiles': [
            {'id': 'a', 'name': 'A server with a rather long name', 'protocol': 'trojan', 'server': 'a.example.com', 'port': 443, 'password': 'pw'},
            {'id': 't', 'name': 'Tor (snowflake)', 'protocol': 'tor', 'server': '', 'port': 0, 'tor': {'builtin': 'snowflake'}},
          ],
        });
        await tester.pumpWidget(ProviderScope(
          overrides: [storeProvider.overrideWithValue(store), engineProvider.overrideWithValue(MockEngine())],
          child: MaterialApp(
            theme: buildTheme(Brightness.dark),
            supportedLocales: AppLocalizations.supportedLocales,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            home: screen,
          ),
        ));
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox());
      });
    }

    testWidgets('import sheet on a $name: nothing overflows, the button can be reached', (tester) async {
      tester.view.physicalSize = size;
      tester.view.devicePixelRatio = 1;
      tester.platformDispatcher.textScaleFactorTestValue = scale;
      addTearDown(tester.view.reset);
      addTearDown(tester.platformDispatcher.clearAllTestValues);
      await tester.pumpWidget(ProviderScope(
        overrides: [storeProvider.overrideWithValue(AppStore.memory()), engineProvider.overrideWithValue(MockEngine())],
        child: MaterialApp(
          theme: buildTheme(Brightness.dark),
          supportedLocales: AppLocalizations.supportedLocales,
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          home: Consumer(
            builder: (context, ref, _) => TextButton(onPressed: () => showImportSheet(context, ref), child: const Text('open')),
          ),
        ),
      ));
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();
      expect(tester.takeException(), isNull);
      // Scrolls inside the sheet instead of overflowing.
      await tester.scrollUntilVisible(find.widgetWithText(FilledButton, 'Import'), 100, scrollable: find.byType(Scrollable).last);
      expect(tester.takeException(), isNull);
      await tester.pumpWidget(const SizedBox());
    });
  }
}
