import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:omnitun/engine/core_engine.dart';
import 'package:omnitun/engine/daemon_engine.dart';
import 'package:omnitun/engine/engine.dart';
import 'package:omnitun/l10n/errors.dart';
import 'package:omnitun/l10n/form_text.dart';
import 'package:omnitun/l10n/l10n.dart';
import 'package:omnitun/state/countries.dart';
import 'package:omnitun/state/country_names.dart';

const _languages = ['en', 'fa', 'ru', 'zh', 'ar'];

Future<AppLocalizations> _load(String code) => AppLocalizations.delegate.load(Locale(code));

void main() {
  const codes = {
    'FORM_REQUIRED': {'label': 'Password'},
    'FORM_NOT_NUMBER': {'label': 'Port'},
    'FORM_NOT_BOOL': {'label': 'Allow insecure'},
    'FORM_BAD_CHOICE': {'label': 'Network', 'value': 'x', 'choices': 'tcp, ws'},
    'FORM_REALITY_ONLY_VLESS': <String, String>{},
    'FORM_UNKNOWN_SECURITY': {'value': 'xyz'},
    'FORM_RESERVED_BYTES': <String, String>{},
    'FORM_REALITY_KEY_REQUIRED': <String, String>{},
    'IMPORT_UNSUPPORTED': <String, String>{},
    'IMPORT_NOTHING': <String, String>{},
    'IMPORT_XRAY_SERVER': <String, String>{},
    'AUTH_FAILED': <String, String>{},
    'SUB_HTTP': {'status': '502'},
    'SUB_TOO_LARGE': <String, String>{},
    'SUB_BAD_URL': <String, String>{},
  };

  test('every code the core reports has a translation in every language', () async {
    final english = <String, String>{};
    for (final lang in _languages) {
      final l = await _load(lang);
      for (final e in codes.entries) {
        final text = translateCode(l, e.key, e.value);
        expect(text, isNotNull, reason: '$lang ${e.key}');
        expect(text, isNotEmpty, reason: '$lang ${e.key}');
        expect(text, isNot(contains('{')), reason: '$lang ${e.key} left a placeholder');
        if (lang == 'en') {
          english[e.key] = text!;
        } else {
          expect(text, isNot(english[e.key]), reason: '$lang ${e.key} is still English');
        }
      }
    }
    expect(english['FORM_REQUIRED'], 'Password is required');
    expect(english['SUB_HTTP'], 'The subscription server answered HTTP 502');
  });

  test('the label of a field is translated inside the message', () async {
    final fa = await _load('fa');
    final text = translateCode(fa, 'FORM_REQUIRED', {'label': 'Password'})!;
    expect(text, contains(formLabel(fa, 'Password')));
    expect(text, isNot(contains('Password')));
  });

  test('codes with no translation fall back to the core message', () async {
    final l = await _load('ru');
    expect(translateCode(l, 'SOMETHING_NEW', {}), isNull);
    expect(describeError(l, EngineException('SOMETHING_NEW', 'it broke')), 'it broke');
    expect(
      describeError(l, EngineException('SUBSCRIPTION_FAILED', 'download subscription: refused')),
      'download subscription: refused',
    );
    expect(describeError(l, StateError('plain')), 'plain');
  });

  test('bypass list problems and Tor steps are translated', () async {
    for (final lang in _languages.skip(1)) {
      final l = await _load(lang);
      for (final code in [
        'EMPTY_EXPRESSION',
        'BAD_EXPRESSION',
        'BAD_NAME',
        'GEO_UNSUPPORTED',
        'BAD_ENTRY',
        'TOO_MANY',
      ]) {
        final issue = BypassIssue(3, 'x', 'English message', code: code, limit: 1000);
        expect(bypassIssueText(l, issue), isNot('English message'), reason: '$lang $code');
      }
      expect(
        bypassIssueText(l, const BypassIssue(1, 'x', 'English message')),
        'English message',
        reason: 'an older core has no code',
      );
      for (final p in [
        const ConnectProgress('engine_start', 5, detail: 'Starting Tor', detailCode: 'TOR_STARTING'),
        const ConnectProgress(
          'engine_start',
          5,
          detail: 'Connecting to Tor 40%',
          detailCode: 'TOR_BOOTSTRAP',
          detailN: 40,
        ),
        const ConnectProgress('engine_start', 5, detail: 'Building a Tor circuit', detailCode: 'TOR_CIRCUIT'),
        const ConnectProgress(
          'engine_start',
          5,
          detail: 'Retrying Tor (2/3)',
          detailCode: 'TOR_RETRY',
          detailN: 2,
          detailM: 3,
        ),
      ]) {
        final text = progressDetail(l, p)!;
        expect(text, isNot(p.detail), reason: '$lang ${p.detailCode}');
        if (p.detailCode == 'TOR_BOOTSTRAP') expect(text, contains('40'));
        if (p.detailCode == 'TOR_RETRY') expect(text, allOf(contains('2'), contains('3')));
      }
    }
    final en = await _load('en');
    expect(
      progressDetail(
        en,
        const ConnectProgress(
          'engine_start',
          5,
          detail: 'Connecting to Tor 40%',
          detailCode: 'TOR_BOOTSTRAP',
          detailN: 40,
        ),
      ),
      'Connecting to Tor 40%',
    );
    expect(progressDetail(en, const ConnectProgress('engine_start', 5, detail: 'Odd status')), 'Odd status');
    expect(progressDetail(en, const ConnectProgress('engine_start', 5)), isNull);
  });

  test('every country has a name in every language', () {
    for (final lang in countryNamesByLanguage.keys) {
      final names = countryNamesByLanguage[lang]!;
      expect(names.keys.toSet(), countryNames.keys.toSet(), reason: lang);
      expect(names['DE'], isNot('Germany'), reason: lang);
      expect(names.values.every((n) => n.isNotEmpty), isTrue, reason: lang);
    }
    expect(countryNamesByLanguage.keys, unorderedEquals(['fa', 'ru', 'zh', 'ar']));
  });

  testWidgets('a country is named in the language of the app', (tester) async {
    final seen = <String, String>{};
    for (final lang in _languages) {
      await tester.pumpWidget(
        MaterialApp(
          locale: Locale(lang),
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: Builder(
            builder: (context) {
              seen[lang] = context.countryName('DE');
              return const SizedBox();
            },
          ),
        ),
      );
    }
    expect(seen, {'en': 'Germany', 'fa': 'آلمان', 'ru': 'Германия', 'zh': '德国', 'ar': 'ألمانيا'});
    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('fa'),
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Builder(builder: (context) => Text(context.countryName('ZZ'))),
      ),
    );
    expect(find.text('ZZ'), findsOneWidget, reason: 'an unknown code is shown as it is');
  });

  test('a real core reports editor and import problems with codes that are translated', () async {
    final exe = CoreEngine.findDaemonExecutable();
    if (exe == null) {
      markTestSkipped('omnitun-svc not built');
      return;
    }
    final dir = await Directory.systemTemp.createTemp('omnitun-msg');
    final engine = await DaemonEngine.launch(exe, controlFile: '${dir.path}${Platform.pathSeparator}control.json');
    addTearDown(() async {
      engine.dispose();
      await Future<void>.delayed(const Duration(milliseconds: 300));
      await dir.delete(recursive: true).catchError((_) => dir);
    });
    final fa = await _load('fa');

    final missing = await engine
        .buildProfile({'server': 'example.com'}, protocol: 'trojan')
        .then<Object>((_) => 'built', onError: (Object e) => e);
    expect(missing, isA<EngineException>());
    final e = missing as EngineException;
    expect(e.code, 'FORM_REQUIRED');
    expect(e.args['label'], 'Password');
    expect(describeError(fa, e), translateCode(fa, 'FORM_REQUIRED', {'label': 'Password'}));
    expect(describeError(fa, e), isNot(e.message));

    final notNumber = await engine
        .buildProfile({'server': 'example.com', 'password': 'x', 'port': 'abc'}, protocol: 'trojan')
        .then<Object>((_) => 'built', onError: (Object e) => e);
    expect((notNumber as EngineException).code, 'FORM_NOT_NUMBER');

    final nothing = await engine.importText('vless://not-valid');
    expect(nothing.profiles, isEmpty);
    expect(nothing.failure?.code, 'IMPORT_NOTHING');
    expect(describeError(fa, nothing.failure!), isNot(nothing.failure!.message));
  });
}
