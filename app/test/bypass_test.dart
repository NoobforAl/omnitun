import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:omnitun/engine/engine.dart';
import 'package:omnitun/engine/mock_engine.dart';
import 'package:omnitun/main.dart';
import 'package:omnitun/state/providers.dart';
import 'package:omnitun/state/store.dart';

/// Remembers how it was asked to connect.
class RecordingEngine extends MockEngine {
  RecordingEngine() : super(connectDelay: Duration.zero);
  ConnectOptions? options;

  @override
  Future<void> connect(
    ProfileInfo profile, {
    List<ProfileInfo> fallbacks = const [],
    ConnectOptions options = const ConnectOptions(),
  }) {
    this.options = options;
    return super.connect(profile, fallbacks: fallbacks, options: options);
  }
}

void main() {
  testWidgets('add a list by typing, check it, switch it on and off, and connect with it', (tester) async {
    tester.view.physicalSize = const Size(800, 1600);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final engine = RecordingEngine();
    final store = AppStore.memory({
      'onboarded': true,
      'profiles': [
        {'id': 'a', 'name': 'A', 'protocol': 'trojan', 'server': 'a.example.com', 'port': 443},
      ],
    });
    await tester.pumpWidget(
      ProviderScope(
        overrides: [storeProvider.overrideWithValue(store), engineProvider.overrideWithValue(engine)],
        child: const OmnitunApp(),
      ),
    );
    await tester.pumpAndSettle();
    final c = ProviderScope.containerOf(tester.element(find.byType(OmnitunApp)));

    await tester.tap(find.byTooltip('Settings'));
    await tester.pumpAndSettle();
    expect(find.text('Websites and addresses that skip the VPN'), findsOneWidget);
    await tester.tap(find.text('Bypass lists'));
    await tester.pumpAndSettle();
    expect(find.text('No lists yet.'), findsOneWidget);

    await tester.tap(find.byTooltip('Add a list'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Type it'));
    await tester.pumpAndSettle();
    expect(find.text('New list'), findsOneWidget);
    expect(find.text('Save'), findsOneWidget);

    await tester.enterText(
      find.widgetWithText(TextField, 'Entries, one per line'),
      'bank.example\n10.0.0.0/8\nnot a name\n',
    );
    await tester.pump(const Duration(milliseconds: 600)); // the check waits for typing to stop
    await tester.pumpAndSettle();
    expect(find.textContaining('2 entries'), findsOneWidget);
    expect(find.textContaining('1 line will be ignored'), findsOneWidget);
    expect(find.textContaining('not a name'), findsWidgets);
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    expect(find.text('My list'), findsOneWidget);
    expect(find.text('2 entries'), findsOneWidget);
    expect(c.read(bypassListsProvider).single.text, contains('bank.example'));

    // Only listed use the VPN.
    await tester.tap(find.text('Only listed use the VPN'));
    await tester.pumpAndSettle();
    expect(c.read(settingsProvider).bypassOnly, isTrue);
    await tester.pageBack();
    await tester.pumpAndSettle();
    expect(find.text('2 entries in use'), findsOneWidget);
    await tester.pageBack();
    await tester.pumpAndSettle();

    await tester.tap(find.text('Connect'));
    await tester.pump(const Duration(milliseconds: 50));
    expect(engine.options!.bypass, contains('bank.example'));
    expect(engine.options!.bypass, contains('10.0.0.0/8'));
    expect(engine.options!.bypassOnly, isTrue);
    await tester.tap(find.text('Disconnect'));
    await tester.pump(const Duration(milliseconds: 400));

    // Switched off, nothing is sent.
    c.read(bypassListsProvider.notifier).update(c.read(bypassListsProvider).single.copyWith(enabled: false));
    expect(c.read(bypassListsProvider.notifier).enabledText, isEmpty);
    await tester.tap(find.text('Connect'));
    await tester.pump(const Duration(milliseconds: 50));
    expect(engine.options!.bypass, isEmpty);
    await tester.tap(find.text('Disconnect'));
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pumpWidget(const SizedBox());
  });

  test('lists survive a restart and the connect options carry them', () {
    final store = AppStore.memory();
    final first = ProviderContainer(
      overrides: [storeProvider.overrideWithValue(store), engineProvider.overrideWithValue(MockEngine())],
    );
    first.read(bypassListsProvider.notifier).add(name: 'Banks', text: 'bank.example\n', entries: 1);
    first.read(bypassListsProvider.notifier).add(name: 'Office', text: '203.0.113.0/24', entries: 1);
    first.read(settingsProvider.notifier).update((s) => s.copyWith(bypassOnly: true));
    first.dispose();

    final second = ProviderContainer(
      overrides: [storeProvider.overrideWithValue(store), engineProvider.overrideWithValue(MockEngine())],
    );
    addTearDown(second.dispose);
    expect(second.read(bypassListsProvider).map((l) => l.name), ['Banks', 'Office']);
    expect(second.read(bypassListsProvider.notifier).enabledText, 'bank.example\n203.0.113.0/24');
    expect(second.read(settingsProvider).bypassOnly, isTrue);
    expect(const SettingsState().bypassOnly, isFalse);
  });
}
