import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:omnitun/engine/engine.dart';
import 'package:omnitun/engine/mock_engine.dart';
import 'package:omnitun/main.dart';
import 'package:omnitun/state/providers.dart';

void main() {
  testWidgets('import a link, then connect and disconnect', (tester) async {
    await tester.pumpWidget(ProviderScope(
      overrides: [engineProvider.overrideWithValue(MockEngine(connectDelay: Duration.zero))],
      child: const OmnitunApp(),
    ));

    // The first start opens with a short introduction.
    await tester.pumpAndSettle();
    expect(find.text('Private and simple'), findsOneWidget);
    await tester.tap(find.text('Skip'));
    await tester.pumpAndSettle();

    expect(find.text('Not connected'), findsOneWidget);
    expect(find.text('No profiles yet'), findsOneWidget);

    // The home screen has no footer to switch pages with.
    expect(find.byType(NavigationBar), findsNothing);
    expect(find.byType(NavigationRail), findsNothing);

    await tester.tap(find.text('Import config'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'trojan://pw@example.com:443#Test%20Server');
    await tester.tap(find.widgetWithText(FilledButton, 'Import'));
    await tester.pumpAndSettle();
    expect(find.text('Automatic'), findsOneWidget); // the location row of the home screen
    final container = ProviderScope.containerOf(tester.element(find.byType(OmnitunApp)));
    expect(container.read(profilesProvider).single.name, 'Test Server');

    await tester.tap(find.text('Connect'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 10));
    expect(container.read(connectionProvider).status, ConnStatus.connected);
    expect(find.textContaining('Connected'), findsOneWidget); // "Connected to <country>" once the exit is known
    expect(find.text('Test Server'), findsOneWidget); // where you exit

    await tester.tap(find.text('Disconnect'));
    await tester.pump(const Duration(milliseconds: 400));
    expect(container.read(connectionProvider).status, ConnStatus.disconnected);
    await tester.pumpWidget(const SizedBox());
  });
}
