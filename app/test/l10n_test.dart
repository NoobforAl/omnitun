import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:omnitun/engine/core_engine.dart';
import 'package:omnitun/engine/daemon_engine.dart';
import 'package:omnitun/engine/engine.dart';
import 'package:omnitun/engine/mock_engine.dart';
import 'package:omnitun/l10n/form_text.dart';
import 'package:omnitun/l10n/l10n.dart';
import 'package:omnitun/main.dart';
import 'package:omnitun/state/providers.dart';
import 'package:omnitun/state/store.dart';

const _languages = ['en', 'fa', 'ru', 'zh', 'ar'];
const _rtl = {'fa', 'ar'};

Future<AppLocalizations> _load(String code) => AppLocalizations.delegate.load(Locale(code));

Map<String, dynamic> _arb(String code) =>
    jsonDecode(File('lib/l10n/app_$code.arb').readAsStringSync()) as Map<String, dynamic>;

/// The `{name}` placeholders of a message, not counting plural/select keywords.
Set<String> _holes(String s) => {for (final m in RegExp(r'\{(\w+)(?:,|\})').allMatches(s)) m.group(1)!}
  ..removeWhere(
    (k) =>
        RegExp(r'^[^\x00-\x7F]').hasMatch(k) ||
        {'plural', 'select', 'other', 'one', 'two', 'few', 'many', 'zero'}.contains(k),
  );

void main() {
  test('every language has every message, with the same placeholders', () {
    final en = _arb('en');
    final messages = [
      for (final k in en.keys)
        if (!k.startsWith('@')) k,
    ];
    expect(messages.length, greaterThan(300));
    for (final code in _languages.skip(1)) {
      final tr = _arb(code);
      expect(
        {
          for (final k in tr.keys)
            if (!k.startsWith('@')) k,
        },
        messages.toSet(),
        reason: code,
      );
      for (final k in messages) {
        expect(tr[k], isA<String>(), reason: '$code $k');
        expect((tr[k] as String).trim(), isNotEmpty, reason: '$code $k');
        // Placeholder names of ASCII words must match (the plural syntax is checked by the generator).
        final want = _holes(en[k] as String).where((h) => RegExp(r'^[A-Za-z]\w*$').hasMatch(h)).toSet();
        final got = _holes(tr[k] as String).where((h) => RegExp(r'^[A-Za-z]\w*$').hasMatch(h)).toSet();
        expect(got, want, reason: '$code $k');
      }
    }
  });

  test('plurals follow each language', () async {
    final ru = await _load('ru');
    expect(ru.serversCount(1), '1 сервер');
    expect(ru.serversCount(2), '2 сервера');
    expect(ru.serversCount(5), '5 серверов');
    expect(ru.serversCount(21), '21 сервер');
    expect(ru.serversCount(22), '22 сервера');
    expect(ru.serversCount(25), '25 серверов');
    final ar = await _load('ar');
    expect(ar.serversCount(1), 'خادم واحد');
    expect(ar.serversCount(2), 'خادمان');
    expect(ar.serversCount(5), '5 خوادم');
    expect(ar.serversCount(11), '11 خادمًا');
    expect(ar.serversCount(100), '100 خادم');
    final en = await _load('en');
    expect(en.serversCount(1), '1 server');
    expect(en.serversCount(3), '3 servers');
    expect(en.importedProfiles(2, 0), 'Imported 2 profiles');
    expect(en.importedProfiles(1, 3), 'Imported 1 profile, 3 already present');
    final zh = await _load('zh');
    expect(zh.serversCount(3), '3 台服务器');
    final fa = await _load('fa');
    expect(fa.importedProfiles(2, 1), contains('2'));
  });

  test('failures are worded from their code in every language', () async {
    for (final code in _languages) {
      final l = await _load(code);
      final timeout = localizedError(
        l,
        const EngineState(
          status: ConnStatus.error,
          error: 'Contacting server failed: the server did not answer in time',
          errorCode: 'TIMEOUT',
          progress: ConnectProgress('handshake', 85),
        ),
      );
      expect(timeout, contains(l.errTimeout), reason: code);
      expect(timeout, contains(l.stageHandshake), reason: code);
      // Things without a stage, and offline.
      expect(
        localizedError(
          l,
          const EngineState(
            status: ConnStatus.error,
            error: 'VPN permission was not granted',
            errorCode: 'NO_PERMISSION',
          ),
        ),
        l.errPermission,
      );
      expect(
        localizedError(l, const EngineState(status: ConnStatus.error, error: 'x', errorCode: 'NO_NETWORK')),
        l.statusNoInternet,
      );
      // Something unusual shows as the core wrote it.
      expect(localizedError(l, const EngineState(status: ConnStatus.error, error: 'odd failure')), 'odd failure');
      expect(localizedError(l, const EngineState()), isNull);
    }
  });

  testWidgets('each language shows its words, right to left where it should, and nothing overflows', (tester) async {
    tester.view.physicalSize = const Size(360, 760); // a small phone
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    for (final code in _languages) {
      final l = await _load(code);
      final store = AppStore.memory({
        'onboarded': true,
        'settings': {'language': code},
        'profiles': [
          {'id': 'd1', 'name': 'DE-Berlin-01', 'protocol': 'trojan', 'server': 'a.example.com', 'port': 443},
          {'id': 'd2', 'name': 'DE-Frankfurt-02', 'protocol': 'vless', 'server': 'b.example.com', 'port': 443},
          {'id': 't', 'name': 'Tor', 'protocol': 'tor', 'server': '', 'port': 0},
        ],
      });
      await tester.pumpWidget(
        ProviderScope(
          overrides: [storeProvider.overrideWithValue(store), engineProvider.overrideWithValue(MockEngine())],
          child: const OmnitunApp(),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text(l.buttonConnect), findsOneWidget, reason: code);
      expect(find.text(l.statusDisconnected), findsOneWidget, reason: code);
      expect(find.text(l.locationLabel), findsOneWidget, reason: code);
      final dir = Directionality.of(tester.element(find.byType(Scaffold).first));
      expect(dir, _rtl.contains(code) ? TextDirection.rtl : TextDirection.ltr, reason: code);

      // The location list, a country opened, the add menu.
      await tester.tap(find.text(l.locationLabel));
      await tester.pumpAndSettle();
      expect(find.text(l.selectLocation), findsOneWidget, reason: code);
      expect(find.text(l.automaticHint), findsOneWidget, reason: code);
      await tester.tap(find.byTooltip(l.showServers).first);
      await tester.pumpAndSettle();
      await tester.tap(find.byTooltip(l.addServersTooltip));
      await tester.pumpAndSettle();
      expect(find.text(l.menuNewServer), findsOneWidget, reason: code);
      await tester.tap(find.text(l.menuNewServer));
      await tester.pumpAndSettle();
      expect(find.text(l.addServerTitle), findsOneWidget, reason: code);
      await tester.tap(find.byType(BackButton).last); // pageBack looks for the English tooltip
      await tester.pumpAndSettle();
      await tester.tap(find.byType(BackButton).last); // pageBack looks for the English tooltip
      await tester.pumpAndSettle();

      // Settings, with its long explanations, and the screens it opens.
      await tester.tap(find.byTooltip(l.tooltipSettings));
      await tester.pumpAndSettle();
      expect(find.text(l.settingsConnection), findsOneWidget, reason: code);
      expect(find.text(l.vpnMode), findsOneWidget, reason: code);
      await tester.scrollUntilVisible(find.text(l.bypassTitle), 200, scrollable: find.byType(Scrollable).first);
      await tester.ensureVisible(find.text(l.bypassTitle));
      await tester.pumpAndSettle();
      await tester.tap(find.text(l.bypassTitle));
      await tester.pumpAndSettle();
      expect(find.text(l.bypassIntro), findsOneWidget, reason: code);
      await tester.tap(find.byType(BackButton).last); // pageBack looks for the English tooltip
      await tester.pumpAndSettle();
      await tester.tap(find.byType(BackButton).last); // pageBack looks for the English tooltip
      await tester.pumpAndSettle();

      // Connecting shows the translated state.
      await tester.tap(find.text(l.buttonConnect));
      await tester.pump(const Duration(milliseconds: 300));
      expect(find.text(l.statusConnecting), findsOneWidget, reason: code);
      await tester.tap(find.text(l.buttonCancel));
      await tester.pump(const Duration(milliseconds: 900));
      await tester.pump(const Duration(milliseconds: 400));
      await tester.pumpWidget(const SizedBox());
    }
  });

  testWidgets('the language chosen in Settings applies at once and is remembered', (tester) async {
    tester.view.physicalSize = const Size(800, 1600);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final store = AppStore.memory({'onboarded': true});
    await tester.pumpWidget(
      ProviderScope(
        overrides: [storeProvider.overrideWithValue(store), engineProvider.overrideWithValue(MockEngine())],
        child: const OmnitunApp(),
      ),
    );
    await tester.pumpAndSettle();
    final c = ProviderScope.containerOf(tester.element(find.byType(OmnitunApp)));
    expect(find.text('Not connected'), findsOneWidget);

    await tester.tap(find.byTooltip('Settings'));
    await tester.pumpAndSettle();
    await tester.scrollUntilVisible(find.text('Language'), 200, scrollable: find.byType(Scrollable).first);
    await tester.tap(find.text('Language'));
    await tester.pumpAndSettle();
    expect(find.text('فارسی'), findsOneWidget);
    expect(find.text('العربية'), findsOneWidget);
    await tester.tap(find.text('فارسی'));
    await tester.pumpAndSettle();
    expect(c.read(settingsProvider).language, 'fa');
    final fa = await _load('fa');
    expect(find.text(fa.settingsTitle), findsOneWidget); // the open screen changed
    await tester.pump(const Duration(milliseconds: 400));
    expect(store.read<Map<String, dynamic>>('settings')!['language'], 'fa');

    // Back to the system language.
    await tester.scrollUntilVisible(find.text(fa.language), 200, scrollable: find.byType(Scrollable).first);
    await tester.tap(find.text(fa.language));
    await tester.pumpAndSettle();
    await tester.tap(find.text(fa.languageSystem).last);
    await tester.pumpAndSettle();
    expect(c.read(settingsProvider).language, '');
    expect(find.text('Settings'), findsOneWidget);
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pumpWidget(const SizedBox());
  });

  test('every label and help text of the real core editors is translated', () async {
    final exe = CoreEngine.findDaemonExecutable();
    if (exe == null) {
      markTestSkipped('omnitun-svc not built');
      return;
    }
    final dir = await Directory.systemTemp.createTemp('omnitun-l10n');
    final engine = await DaemonEngine.launch(exe, controlFile: '${dir.path}${Platform.pathSeparator}control.json');
    addTearDown(() async {
      engine.dispose();
      await Future<void>.delayed(const Duration(milliseconds: 300));
      await dir.delete(recursive: true).catchError((_) => dir);
    });
    final forms = await engine.forms();
    final fa = await _load('fa');
    final en = await _load('en');
    // Technical names that are the same in every language.
    const same = {
      'UUID',
      'MTU',
      'ALPN',
      'Flow',
      'Alter id',
      'Jmin',
      'Jmax',
      'S1',
      'S2',
      'H1',
      'H2',
      'H3',
      'H4',
      'I2',
      'I3',
      'I4',
      'I5',
      'J2',
      'J3',
      'Itime',
    };
    final missing = <String>[];
    for (final f in forms) {
      for (final field in f.fields) {
        if (formLabel(en, field.label) != field.label) missing.add('label (English) ${field.label}');
        if (!same.contains(field.label) && formLabel(fa, field.label) == field.label) {
          missing.add('label ${field.label}');
        }
        if (field.help.isNotEmpty) {
          if (formHelp(en, field.help) != field.help) missing.add('help (English) ${field.help}');
          if (formHelp(fa, field.help) == field.help) missing.add('help ${field.help}');
        }
      }
    }
    expect(missing.toSet(), isEmpty, reason: 'texts of the core forms with no translation');
  }, timeout: const Timeout(Duration(seconds: 60)));
}
