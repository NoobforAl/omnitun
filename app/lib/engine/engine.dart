/// Contract between the UI and the native core (Go libcore).
///
/// The real implementation talks to libcore over FFI (desktop) or a platform
/// channel to the Android VpnService. [MockEngine] lets the UI run without it.
library;

enum ConnStatus { disconnected, connecting, connected, disconnecting, error, noInternet }

class ProfileInfo {
  const ProfileInfo({
    required this.id,
    required this.name,
    required this.protocol,
    required this.server,
    required this.port,
    this.group = 'Default',
    this.latencyMs,
    this.raw = const {},
  });

  final String id;
  final String name;
  final String protocol;
  final String server;
  final int port;
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

  factory ProfileInfo.fromJson(Map<String, dynamic> j) => ProfileInfo(
        id: j['id'] as String,
        name: (j['name'] as String?)?.isNotEmpty == true ? j['name'] as String : '${j['server']}:${j['port']}',
        protocol: j['protocol'] as String,
        server: j['server'] as String,
        port: j['port'] as int,
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
}

/// One connection stage (see PLAN.md 3.3 #6). [percent] runs 0-100.
class ConnectProgress {
  const ConnectProgress(this.stage, this.percent, {this.profile});

  final String stage;
  final int percent;

  /// Display name of the profile being tried, when relevant.
  final String? profile;

  factory ConnectProgress.fromJson(Map<String, dynamic> j) => ConnectProgress(
        j['stage'] as String,
        (j['percent'] as num).toInt(),
        profile: j['profile'] as String?,
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

  /// "Stop VPN when app is closed". Used where the OS reports app closure (Android).
  Future<void> setStopOnClose(bool value);

  void dispose();
}
