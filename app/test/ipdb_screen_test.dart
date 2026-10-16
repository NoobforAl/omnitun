import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:omnitun/l10n/l10n.dart';
import 'package:omnitun/state/ipdb.dart';
import 'package:omnitun/state/ipdb_state.dart';
import 'package:omnitun/ui/screens/ipdb_screen.dart';
import 'package:omnitun/ui/theme.dart';

void main() {
  group('the offline database screen', () {
    Future<void> show(WidgetTester tester, IpDbState state) async {
      tester.view.physicalSize = const Size(720, 1400);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      await tester.pumpWidget(
        ProviderScope(
          overrides: [ipDbProvider.overrideWith(() => _Fixed(state))],
          child: MaterialApp(
            theme: buildTheme(Brightness.dark),
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
            home: const IpDbScreen(),
          ),
        ),
      );
      await tester.pump(const Duration(milliseconds: 500));
    }

    const mid = IpDbProgress(
      part: 0,
      parts: 2,
      file: 'v4',
      received: 4000000,
      total: 10000000,
      overallReceived: 10000000,
      overallTotal: 23000000,
      bytesPerSecond: 1800000,
    );

    testWidgets('not downloaded: explains and offers Download', (tester) async {
      await show(tester, const IpDbState());
      expect(find.text('Download'), findsOneWidget);
      expect(find.textContaining('without asking a website'), findsWidgets);
      await tester.pumpWidget(const SizedBox());
    });

    testWidgets('downloading: percent, bytes, speed, time left and Pause', (tester) async {
      await show(tester, const IpDbState(phase: IpDbPhase.downloading, progress: mid));
      expect(find.text('43%'), findsOneWidget);
      expect(find.textContaining('10.0 MB / 23.0 MB'), findsOneWidget);
      expect(find.textContaining('1.8 MB/s'), findsOneWidget);
      expect(find.textContaining('left'), findsOneWidget);
      expect(find.text('Pause'), findsOneWidget);
      await tester.pumpWidget(const SizedBox());
    });

    testWidgets('failed: says where it stopped and offers Continue', (tester) async {
      await show(tester, const IpDbState(phase: IpDbPhase.failed, progress: mid, error: 'network'));
      expect(find.textContaining('Stopped at 43%'), findsOneWidget);
      expect(find.textContaining('no connection to the data servers'), findsOneWidget);
      expect(find.text('Continue'), findsOneWidget);
      await tester.pumpWidget(const SizedBox());
    });
  });
}

/// A controller that holds a given state and does nothing.
class _Fixed extends IpDbController {
  _Fixed(this._state);

  final IpDbState _state;

  @override
  IpDbState build() => _state;
}
