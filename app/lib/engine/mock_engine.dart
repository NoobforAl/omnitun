import 'dart:async';
import 'dart:math';

import 'engine.dart';

/// Simulated engine for UI development until libcore is wired in.
class MockEngine implements VpnEngine {
  MockEngine({this.connectDelay = const Duration(milliseconds: 1200)});

  final Duration connectDelay;
  final _states = StreamController<EngineState>.broadcast();
  final _logs = StreamController<String>.broadcast();
  final _rng = Random();
  EngineState _state = const EngineState();
  Timer? _traffic;

  @override
  Stream<EngineState> get states => _states.stream;

  @override
  Stream<String> get logs => _logs.stream;

  void _emit(EngineState s) {
    _state = s;
    _states.add(s);
  }

  void _log(String line) => _logs.add('${DateTime.now().toIso8601String().substring(11, 19)}  $line');

  @override
  String get name => 'preview (mock)';

  @override
  Future<void> connect(
    ProfileInfo profile, {
    List<ProfileInfo> fallbacks = const [],
    ConnectOptions options = const ConnectOptions(),
  }) async {
    _log('connecting to ${profile.name} (${profile.protocol})');
    const stages = [
      ConnectProgress('network_check', 15),
      ConnectProgress('prepare', 25),
      ConnectProgress('engine_start', 45),
      ConnectProgress('tunnel_up', 65),
    ];
    for (final p in [...stages, ConnectProgress('handshake', 85, profile: profile.name)]) {
      _emit(EngineState(status: ConnStatus.connecting, activeProfileId: profile.id, progress: p));
      await Future<void>.delayed(connectDelay ~/ 5);
      if (_state.status != ConnStatus.connecting) return;
    }
    _emit(EngineState(
      status: ConnStatus.connected,
      activeProfileId: profile.id,
      connectedSince: DateTime.now(),
      progress: ConnectProgress('connected', 100, profile: profile.name),
    ));
    _log('connected, health probe ok');
    _traffic?.cancel();
    _traffic = Timer.periodic(const Duration(seconds: 1), (_) {
      _emit(_state.copyWith(upBps: _rng.nextInt(200000), downBps: _rng.nextInt(2000000)));
    });
  }

  @override
  Future<void> disconnect() async {
    _traffic?.cancel();
    _emit(_state.copyWith(status: ConnStatus.disconnecting));
    await Future<void>.delayed(const Duration(milliseconds: 300));
    _emit(const EngineState());
    _log('disconnected');
  }

  @override
  Future<ImportResult> importText(String text) async {
    // Minimal parse so the UI flow works; the Go core does the real parsing.
    final profiles = <ProfileInfo>[];
    final errors = <String>[];
    for (final line in text.split('\n').map((l) => l.trim()).where((l) => l.isNotEmpty)) {
      final uri = Uri.tryParse(line);
      if (uri == null || uri.host.isEmpty || !uri.hasPort) {
        errors.add(line);
        continue;
      }
      final protocol = switch (uri.scheme) {
        'hy2' => 'hysteria2',
        'ss' => 'shadowsocks',
        'wg' => 'wireguard',
        final s => s,
      };
      profiles.add(ProfileInfo(
        id: '${uri.scheme}-${uri.host}-${uri.port}-${profiles.length}',
        name: uri.fragment.isEmpty ? '${uri.host}:${uri.port}' : Uri.decodeComponent(uri.fragment),
        protocol: protocol,
        server: uri.host,
        port: uri.port,
      ));
    }
    return ImportResult(profiles, errors);
  }

  @override
  Future<int> testLatency(ProfileInfo profile) async {
    await Future<void>.delayed(Duration(milliseconds: 200 + _rng.nextInt(400)));
    return _rng.nextInt(10) == 0 ? -1 : 40 + _rng.nextInt(260);
  }

  @override
  Future<void> setStopOnClose(bool value) async {}

  @override
  void dispose() {
    _traffic?.cancel();
    _states.close();
    _logs.close();
  }
}
