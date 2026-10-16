import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:omnitun/l10n/app_localizations.dart';
import 'package:omnitun/ui/screens/notices_screen.dart';

const _sample = '''# Third-party notices

## Packages

| Go | a | v1 | MIT |

## License texts

### Used by: Go example.com/a v1.0.0, Dart b 2.0.0, Go example.com/c v3

MIT License

Permission is hereby granted.

### Used by: Rust d 0.1.0

Apache License 2.0 text
''';

Widget _app(Widget home, {Locale locale = const Locale('en')}) => MaterialApp(
  locale: locale,
  localizationsDelegates: const [
    AppLocalizations.delegate,
    GlobalMaterialLocalizations.delegate,
    GlobalWidgetsLocalizations.delegate,
    GlobalCupertinoLocalizations.delegate,
  ],
  supportedLocales: AppLocalizations.supportedLocales,
  home: home,
);

void main() {
  test('the notices are split into license texts with their packages', () {
    final groups = parseNotices(_sample);
    expect(groups, hasLength(2));
    expect(groups[0].packages, ['Go example.com/a v1.0.0', 'Dart b 2.0.0', 'Go example.com/c v3']);
    expect(groups[0].text, startsWith('MIT License'));
    expect(groups[0].text, contains('Permission is hereby granted.'));
    expect(groups[0].text, isNot(contains('Apache')));
    expect(groups[1].packages, ['Rust d 0.1.0']);
    expect(parseNotices('nothing here'), isEmpty);
  });

  test('the bundled notices are there and cover Go, Dart and Rust', () async {
    TestWidgetsFlutterBinding.ensureInitialized();
    final groups = parseNotices(await rootBundle.loadString('assets/third_party_notices.md'));
    expect(groups.length, greaterThan(50));
    final all = groups.expand((g) => g.packages).toList();
    expect(all.any((p) => p.startsWith('Go github.com/sagernet/sing-box')), isTrue);
    expect(all.any((p) => p.startsWith('Dart flutter_riverpod')), isTrue);
    expect(all.any((p) => p.startsWith('Rust ')), isTrue);
    expect(groups.every((g) => g.text.length > 40), isTrue, reason: 'a license without text');
  });

  testWidgets('the screen lists the licenses and opens one', (tester) async {
    await tester.pumpWidget(_app(NoticesScreen(load: () async => _sample)));
    await tester.pumpAndSettle();
    expect(find.text('Open-source licenses'), findsOneWidget);
    expect(find.textContaining('4 packages'), findsOneWidget);
    expect(find.text('Go example.com/a v1.0.0, Dart b 2.0.0'), findsOneWidget);
    expect(find.text('+1 more'), findsOneWidget);
    expect(find.textContaining('Permission is hereby granted.'), findsNothing);
    await tester.tap(find.text('Go example.com/a v1.0.0, Dart b 2.0.0'));
    await tester.pumpAndSettle();
    expect(find.textContaining('Permission is hereby granted.'), findsOneWidget);
  });

  testWidgets('a list that cannot be read says so, in the language of the app', (tester) async {
    await tester.pumpWidget(_app(NoticesScreen(load: () async => ''), locale: const Locale('ru')));
    await tester.pumpAndSettle();
    expect(find.text('Не удалось прочитать список лицензий.'), findsOneWidget);
    expect(find.text('Лицензии с открытым кодом'), findsOneWidget);
  });
}
