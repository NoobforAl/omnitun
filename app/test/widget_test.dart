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
    // The app's own default configuration is Tor, there from the first start.
    expect(find.text('No profiles yet'), findsNothing);
    expect(find.text('Tor (snowflake)'), findsOneWidget);

    // The home screen has no footer to switch pages with.
    expect(find.byType(NavigationBar), findsNothing);
    expect(find.byType(NavigationRail), findsNothing);

    await tester.tap(find.text('Location'));
    await tester.pumpAndSettle();
    await tester.tap(find.byIcon(Icons.add));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Import a config, link or QR'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'trojan://pw@example.com:443#Test%20Server');
    await tester.tap(find.widgetWithText(FilledButton, 'Import'));
    await tester.pumpAndSettle();
    final container = ProviderScope.containerOf(tester.element(find.byType(OmnitunApp)));
    final imported = container.read(profilesProvider).firstWhere((p) => p.name == 'Test Server');
    // Importing does not move the user off the default; choosing the server does.
    expect(container.read(selectedProfileProvider)?.protocol, 'tor');
    container.read(selectedProfileIdProvider.notifier).select(imported.id);
    await tester.pageBack();
    await tester.pumpAndSettle();

    await tester.tap(find.text('Connect'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 10));
    expect(container.read(connectionProvider).status, ConnStatus.connected);
    expect(find.textContaining('Connected'), findsOneWidget); // "Connected to <country>" once the exit is known
    expect(find.text('Test Server'), findsNWidgets(2)); // where you exit, and the chosen location

    await tester.tap(find.text('Disconnect'));
    await tester.pump(const Duration(milliseconds: 400));
    expect(container.read(connectionProvider).status, ConnStatus.disconnected);
    await tester.pumpWidget(const SizedBox());
  });
}
