import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:omnitun/engine/mock_engine.dart';
import 'package:omnitun/main.dart';
import 'package:omnitun/state/providers.dart';
import 'package:omnitun/state/store.dart';

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
  await tester.pumpAndSettle();
  return ProviderScope.containerOf(tester.element(find.byType(OmnitunApp)));
}

void main() {
  testWidgets('the first start shows the introduction, ends in the import sheet and is not shown again', (
    tester,
  ) async {
    final store = AppStore.memory();
    final c = await _start(tester, store);
    expect(find.text('Private and simple'), findsOneWidget);
    await tester.tap(find.text('Next'));
    await tester.pumpAndSettle();
    expect(find.text('Add your servers'), findsOneWidget);
    await tester.tap(find.text('Next'));
    await tester.pumpAndSettle();
    expect(find.text('Add my first server'), findsOneWidget);
    await tester.tap(find.text('Add my first server'));
    await tester.pumpAndSettle();
    expect(c.read(onboardedProvider), isTrue);
    expect(find.text('Import config'), findsWidgets); // the import sheet
    expect(find.text('Private and simple'), findsNothing);
    await tester.pump(const Duration(milliseconds: 400));
    expect(store.read<bool>('onboarded'), isTrue);
    await tester.pumpWidget(const SizedBox());

    // The next start goes straight to the home screen.
    await tester.pumpWidget(
      ProviderScope(
        overrides: [storeProvider.overrideWithValue(store), engineProvider.overrideWithValue(MockEngine())],
        child: const OmnitunApp(),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Private and simple'), findsNothing);
    expect(find.text('Not connected'), findsOneWidget);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('with servers already there, or after Skip, there is no introduction', (tester) async {
    final store = AppStore.memory({
      'profiles': [
        {'id': 'a', 'name': 'A', 'protocol': 'trojan', 'server': 'a.example.com', 'port': 443},
      ],
    });
    await _start(tester, store);
    expect(find.text('Private and simple'), findsNothing);
    await tester.pumpWidget(const SizedBox());

    final fresh = AppStore.memory();
    final c = await _start(tester, fresh);
    await tester.tap(find.text('Skip'));
    await tester.pumpAndSettle();
    expect(find.text('Private and simple'), findsNothing);
    expect(c.read(onboardedProvider), isTrue);
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pumpWidget(const SizedBox());
  });
}
