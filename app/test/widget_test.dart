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

    expect(find.text('Not connected'), findsOneWidget);
    expect(find.text('No profiles yet'), findsOneWidget);

    await tester.tap(find.text('Import config'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Import').last);
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'trojan://pw@example.com:443#Test%20Server');
    await tester.tap(find.widgetWithText(FilledButton, 'Import'));
    await tester.pumpAndSettle();
    expect(find.text('Test Server'), findsOneWidget);

    await tester.tap(find.text('Home'));
    await tester.pumpAndSettle();
    final container = ProviderScope.containerOf(tester.element(find.byType(OmnitunApp)));

    await tester.tap(find.text('CONNECT'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 10));
    expect(container.read(connectionProvider).status, ConnStatus.connected);
    expect(find.text('Connected'), findsOneWidget);

    await tester.tap(find.text('DISCONNECT'));
    await tester.pump(const Duration(milliseconds: 400));
    expect(container.read(connectionProvider).status, ConnStatus.disconnected);
    await tester.pumpWidget(const SizedBox());
  });
}
