/// Contract between the UI and the native core (Go libcore).
///
/// The real implementation talks to libcore over FFI (desktop) or a platform
/// channel to the Android VpnService. [MockEngine] lets the UI run without it.
library;

/// Thrown for typed failures reported by the core; [message] is fit to show.
class EngineException implements Exception {
  EngineException(this.code, this.message);

  final String code;
  final String message;

  @override
  String toString() => '$code: $message';
}

enum ConnStatus { disconnected, connecting, connected, disconnecting, error, noInternet }

class ProfileInfo {
  const ProfileInfo({
    required this.id,
    required this.name,
    required this.protocol,
    required this.server,
    required this.port,
    this.group = defaultGroup,
    this.latencyMs,
    this.raw = const {},
  });

  /// Group of manually imported profiles; subscriptions use their own ID.
  static const defaultGroup = 'default';

  final String id;
  final String name;
  final String protocol;
  final String server;
  final int port;

  /// What to show as the profile's address (Tor has no server).
  String get address => protocol == 'tor' ? 'Tor network' : '$server:$port';

  /// [defaultGroup] or the ID of the subscription the profile came from.
  final String group;

  /// Last measured latency; null = not tested, -1 = unreachable.
  final int? latencyMs;

  /// The full profile as produced by the core; sent back verbatim on connect.
  final Map<String, dynamic> raw;

  ProfileInfo copyWith({int? latencyMs}) => ProfileInfo(
        id: id,
        name: name,
        protocol: protocol,
        server: server,
        port: port,
        group: group,
        latencyMs: latencyMs ?? this.latencyMs,
        raw: raw,
      );

  /// OpenVPN profiles with `auth-user-pass` and IKEv2 EAP profiles need a username and password.
  bool get needsCredentials =>
      (raw['openvpn'] as Map?)?['needs_auth'] == true || (protocol == 'ikev2' && (raw['ikev2'] as Map?)?['auth'] == 'eap');

  bool get hasCredentials =>
      (raw['username'] as String? ?? '').isNotEmpty && (raw['password'] as String? ?? '').isNotEmpty;

  /// The same profile with sign-in credentials (kept in [raw] so they are persisted).
  ProfileInfo withCredentials(String username, String password) =>
      ProfileInfo.fromJson({...raw, 'username': username, 'password': password}).copyWith(latencyMs: latencyMs);

  /// The same profile assigned to [group] (kept in [raw] so it is persisted).
  ProfileInfo inGroup(String group) => ProfileInfo.fromJson({...raw, 'group': group});

  factory ProfileInfo.fromJson(Map<String, dynamic> j) => ProfileInfo(
        id: j['id'] as String,
        name: (j['name'] as String?)?.isNotEmpty == true ? j['name'] as String : '${j['server']}:${j['port']}',
        protocol: j['protocol'] as String,
        server: j['server'] as String,
        port: j['port'] as int,
        group: j['group'] as String? ?? defaultGroup,
        raw: j,
      );
}

/// How a connection should be made.
class ConnectOptions {
  const ConnectOptions({
    this.tunMode = false,
    this.systemProxy = true,
    this.listenPort = 2080,
    this.dns = '',
    this.autoFailover = true,
    this.bypassLan = true,
    this.includePackages = const [],
    this.excludePackages = const [],
    this.bypass = '',
    this.bypassOnly = false,
  });

  /// Capture all traffic (needs VPN permission / admin). Otherwise proxy mode.
  final bool tunMode;

  /// In proxy mode, point the OS proxy settings at the local port.
  final bool systemProxy;
  final int listenPort;

  /// Remote resolver URL (https://, tls://, tcp://, udp://). Empty = core default.
  final String dns;

  /// Switch to the next working profile when the selected one stops responding.
  final bool autoFailover;

  /// Keep local networks (LAN, link-local, multicast) out of the VPN.
  final bool bypassLan;

  /// Android split tunneling: only these apps use the VPN (wins over [excludePackages]).
  final List<String> includePackages;

  /// Android split tunneling: these apps bypass the VPN.
  final List<String> excludePackages;

  /// Text of the enabled bypass lists (one entry per line): these destinations go around the VPN.
  final String bypass;

  /// With [bypass]: only the listed destinations use the VPN, everything else goes direct.
  final bool bypassOnly;
}

/// One line of a bypass list that cannot be used.
class BypassIssue {
  const BypassIssue(this.line, this.text, this.message);

  final int line;
  final String text;
  final String message;
}

/// What a bypass list holds, as counted by the core.
class BypassReport {
  const BypassReport({this.entries = 0, this.domains = 0, this.ranges = 0, this.issues = const []});

  final int entries;
  final int domains;
  final int ranges;
  final List<BypassIssue> issues;

  factory BypassReport.fromJson(Map<String, dynamic> j) => BypassReport(
    entries: (j['entries'] as num?)?.toInt() ?? 0,
    domains: (j['domains'] as num?)?.toInt() ?? 0,
    ranges: (j['ranges'] as num?)?.toInt() ?? 0,
    issues: [
      for (final i in (j['issues'] as List? ?? const []))
        BypassIssue((i['line'] as num).toInt(), i['text'] as String, i['message'] as String),
    ],
  );
}

/// One connection stage (see PLAN.md 3.3 #6). [percent] runs 0-100.
class ConnectProgress {
  const ConnectProgress(this.stage, this.percent, {this.profile, this.detail});

  final String stage;
  final int percent;

  /// Display name of the profile being tried, when relevant.
  final String? profile;

  /// Finer-grained status that replaces the stage label, e.g. "Connecting to Tor 40%".
  final String? detail;

  factory ConnectProgress.fromJson(Map<String, dynamic> j) => ConnectProgress(
        j['stage'] as String,
        (j['percent'] as num).toInt(),
        profile: j['profile'] as String?,
        detail: j['detail'] as String?,
      );
}

/// An automatic switch made by the core.
class FailoverInfo {
  const FailoverInfo({required this.from, required this.to, required this.reason, required this.at});

  final String from;
  final String to;
  final String reason;
  final DateTime at;
}

class EngineState {
  const EngineState({
    this.status = ConnStatus.disconnected,
    this.activeProfileId,
    this.connectedSince,
    this.error,
    this.upBps = 0,
    this.downBps = 0,
    this.failover,
    this.progress,
  });

  final ConnStatus status;
  final String? activeProfileId;
  final DateTime? connectedSince;
  final String? error;
  final int upBps;
  final int downBps;

  /// The most recent automatic switch during this connection, if any.
  final FailoverInfo? failover;

  /// Current connection stage; on error, the stage that failed.
  final ConnectProgress? progress;

  EngineState copyWith({
    ConnStatus? status,
    String? activeProfileId,
    DateTime? connectedSince,
    String? error,
    int? upBps,
    int? downBps,
  }) =>
      EngineState(
        status: status ?? this.status,
        activeProfileId: activeProfileId ?? this.activeProfileId,
        connectedSince: connectedSince ?? this.connectedSince,
        error: error ?? this.error,
        upBps: upBps ?? this.upBps,
        downBps: downBps ?? this.downBps,
        failover: failover,
        progress: progress,
      );
}

class ImportResult {
  const ImportResult(this.profiles, this.errors);
  final List<ProfileInfo> profiles;
  final List<String> errors;
}

/// Provider metadata from a subscription response (all optional).
class SubscriptionInfo {
  const SubscriptionInfo({
    this.upload = 0,
    this.download = 0,
    this.total = 0,
    this.expire,
    this.updateIntervalHours = 0,
    this.title = '',
  });

  final int upload;
  final int download;

  /// Data cap in bytes; 0 = unlimited/unknown.
  final int total;
  final DateTime? expire;
  final int updateIntervalHours;
  final String title;

  int get used => upload + download;

  factory SubscriptionInfo.fromJson(Map<String, dynamic> j) {
    final expire = (j['expire'] as num?)?.toInt() ?? 0;
    return SubscriptionInfo(
      upload: (j['upload'] as num?)?.toInt() ?? 0,
      download: (j['download'] as num?)?.toInt() ?? 0,
      total: (j['total'] as num?)?.toInt() ?? 0,
      expire: expire > 0 ? DateTime.fromMillisecondsSinceEpoch(expire * 1000) : null,
      updateIntervalHours: (j['update_interval_hours'] as num?)?.toInt() ?? 0,
      title: j['title'] as String? ?? '',
    );
  }

  Map<String, dynamic> toJson() => {
        'upload': upload,
        'download': download,
        'total': total,
        if (expire != null) 'expire': expire!.millisecondsSinceEpoch ~/ 1000,
        'update_interval_hours': updateIntervalHours,
        'title': title,
      };
}

class SubscriptionResult {
  const SubscriptionResult(this.profiles, this.errors, this.info);
  final List<ProfileInfo> profiles;
  final List<String> errors;
  final SubscriptionInfo info;
}

/// One input of a protocol's profile editor (see core/profile/form.go).
class FormFieldSpec {
  const FormFieldSpec({
    required this.key,
    required this.label,
    required this.kind,
    this.help = '',
    this.required = false,
    this.choices = const [],
    this.defaultValue = '',
    this.advanced = false,
  });

  final String key;
  final String label;

  /// text, secret, int, bool, choice, multiline or list.
  final String kind;
  final String help;
  final bool required;
  final List<String> choices;
  final String defaultValue;

  /// Shown only after "More options".
  final bool advanced;

  factory FormFieldSpec.fromJson(Map<String, dynamic> j) => FormFieldSpec(
    key: j['key'] as String,
    label: j['label'] as String,
    kind: j['kind'] as String,
    help: j['help'] as String? ?? '',
    required: j['required'] as bool? ?? false,
    choices: [for (final c in (j['choices'] as List? ?? const [])) c as String],
    defaultValue: j['default'] as String? ?? '',
    advanced: j['advanced'] as bool? ?? false,
  );
}

/// The editor of one protocol.
class ProtocolFormSpec {
  const ProtocolFormSpec({required this.protocol, required this.label, required this.fields});

  final String protocol;
  final String label;
  final List<FormFieldSpec> fields;

  factory ProtocolFormSpec.fromJson(Map<String, dynamic> j) => ProtocolFormSpec(
    protocol: j['protocol'] as String,
    label: j['label'] as String,
    fields: [for (final f in (j['fields'] as List? ?? const [])) FormFieldSpec.fromJson(f as Map<String, dynamic>)],
  );
}

abstract class VpnEngine {
  /// Status, traffic and failover updates.
  Stream<EngineState> get states;

  /// Log lines from the core.
  Stream<String> get logs;

  /// Human-readable engine name for the settings screen.
  String get name;

  Future<void> connect(
    ProfileInfo profile, {
    List<ProfileInfo> fallbacks = const [],
    ConnectOptions options = const ConnectOptions(),
  });
  Future<void> disconnect();
  Future<ImportResult> importText(String text);

  /// Returns latency in ms, or -1 if unreachable.
  Future<int> testLatency(ProfileInfo profile);

  /// Downloads a subscription. [request] is {"url", "auth": {...}, "user_agent"}
  /// (see core/subscription). Throws an exception with a user-facing message on failure.
  Future<SubscriptionResult> fetchSubscription(Map<String, dynamic> request);

  /// "Stop VPN when app is closed". Used where the OS reports app closure (Android).
  Future<void> setStopOnClose(bool value);

  /// The profile editor of every protocol the core can build by hand.
  Future<List<ProtocolFormSpec>> forms();

  /// Builds a profile from the editor's [values] ([protocol] for a new one), or
  /// applies them to a copy of [profile] when editing. Throws [EngineException]
  /// with a message for the user when a value is wrong.
  Future<ProfileInfo> buildProfile(Map<String, String> values, {String? protocol, ProfileInfo? profile});

  /// Checks the text of a bypass list: what it holds and which lines cannot be used.
  Future<BypassReport> parseBypass(String text);

  /// What the editor shows for [profile].
  Future<Map<String, String>> profileValues(ProfileInfo profile);

  /// The share link of [profile]. Throws [EngineException] when the protocol has none.
  Future<String> exportLink(ProfileInfo profile);

  void dispose();
}
