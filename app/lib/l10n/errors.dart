import '../engine/engine.dart';
import 'app_localizations.dart';
import 'form_text.dart';

/// The translation of a failure the core reported by [code], or null for a code
/// that has none (the core's English message is shown instead).
String? translateCode(AppLocalizations l, String code, Map<String, String> args) {
  String label() => formLabel(l, args['label'] ?? '');
  return switch (code) {
    'FORM_REQUIRED' => l.formRequired(label()),
    'FORM_NOT_NUMBER' => l.formNotNumber(label()),
    'FORM_NOT_BOOL' => l.formNotBool(label()),
    'FORM_BAD_CHOICE' => l.formBadChoice(label(), args['value'] ?? '', args['choices'] ?? ''),
    'FORM_REALITY_ONLY_VLESS' => l.formRealityOnlyVless,
    'FORM_UNKNOWN_SECURITY' => l.formUnknownSecurity(args['value'] ?? ''),
    'FORM_RESERVED_BYTES' => l.formReservedBytes,
    'FORM_REALITY_KEY_REQUIRED' => l.formRealityKeyRequired,
    'IMPORT_UNSUPPORTED' =>
      (args['scheme'] ?? '').isEmpty ? l.importUnsupported : l.importUnsupportedScheme(args['scheme']!),
    'IMPORT_NOTHING' => l.importNoServer,
    'IMPORT_XRAY_SERVER' => l.importXrayServer,
    'AUTH_FAILED' => l.subAuthFailed,
    'SUB_HTTP' => l.subHttp(args['status'] ?? ''),
    'SUB_TOO_LARGE' => l.subTooLarge,
    'SUB_BAD_URL' => l.subBadUrl,
    _ => null,
  };
}

/// What to tell the user about an error from the core, in their language.
String describeError(AppLocalizations l, Object e) {
  if (e is EngineException) return translateCode(l, e.code, e.args) ?? e.message;
  return e.toString().replaceFirst(RegExp(r'^(Bad state|[A-Za-z]*Exception|[A-Z_]+): '), '');
}

/// Why a line of a bypass list cannot be used, in the user's language.
String bypassIssueText(AppLocalizations l, BypassIssue i) => switch (i.code) {
  'EMPTY_EXPRESSION' => l.bypassIssueEmptyExpression,
  'BAD_EXPRESSION' => l.bypassIssueBadExpression,
  'BAD_NAME' => l.bypassIssueBadName,
  'GEO_UNSUPPORTED' => l.bypassIssueGeo,
  'BAD_ENTRY' => l.bypassIssueBadEntry,
  'TOO_MANY' => l.bypassIssueTooMany(i.limit),
  _ => i.message,
};

/// The finer-grained connection status (Tor's steps) in the user's language, or null when there is none.
String? progressDetail(AppLocalizations l, ConnectProgress p) => switch (p.detailCode) {
  'TOR_STARTING' => l.torStarting,
  'TOR_BOOTSTRAP' => l.torBootstrap(p.detailN),
  'TOR_CIRCUIT' => l.torCircuit,
  'TOR_RETRY' => l.torRetry(p.detailN, p.detailM),
  _ => p.detail,
};
