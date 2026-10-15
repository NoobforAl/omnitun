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
  });

  final String id;
  final String name;
  final String protocol;
  final String server;
  final int port;
  final String group;

  /// Last measured latency; null = not tested, -1 = unreachable.
  final int? latencyMs;

  ProfileInfo copyWith({int? latencyMs}) => ProfileInfo(
        id: id,
        name: name,
        protocol: protocol,
        server: server,
        port: port,
        group: group,
        latencyMs: latencyMs ?? this.latencyMs,
      );

  factory ProfileInfo.fromJson(Map<String, dynamic> j) => ProfileInfo(
        id: j['id'] as String,
        name: (j['name'] as String?)?.isNotEmpty == true ? j['name'] as String : '${j['server']}:${j['port']}',
        protocol: j['protocol'] as String,
        server: j['server'] as String,
        port: j['port'] as int,
      );
}

class EngineState {
  const EngineState({
    this.status = ConnStatus.disconnected,
    this.activeProfileId,
    this.connectedSince,
    this.error,
    this.upBps = 0,
    this.downBps = 0,
  });

  final ConnStatus status;
  final String? activeProfileId;
  final DateTime? connectedSince;
  final String? error;
  final int upBps;
  final int downBps;

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

  Future<void> connect(ProfileInfo profile, {List<ProfileInfo> fallbacks = const []});
  Future<void> disconnect();
  Future<ImportResult> importText(String text);

  /// Returns latency in ms, or -1 if unreachable.
  Future<int> testLatency(ProfileInfo profile);

  void dispose();
}
