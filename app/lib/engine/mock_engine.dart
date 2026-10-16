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
  Future<SubscriptionResult> fetchSubscription(Map<String, dynamic> request) async =>
      throw StateError('Subscriptions need the Omnitun core (not available in preview mode)');

  @override
  Future<int> testLatency(ProfileInfo profile) async {
    await Future<void>.delayed(Duration(milliseconds: 200 + _rng.nextInt(400)));
    return _rng.nextInt(10) == 0 ? -1 : 40 + _rng.nextInt(260);
  }

  @override
  Future<void> setStopOnClose(bool value) async {}

  @override
  Future<List<ProtocolFormSpec>> forms() async => const [
    ProtocolFormSpec(
      protocol: 'trojan',
      label: 'Trojan',
      fields: [
        FormFieldSpec(key: 'name', label: 'Name', kind: 'text'),
        FormFieldSpec(key: 'server', label: 'Server', kind: 'text', required: true),
        FormFieldSpec(key: 'port', label: 'Port', kind: 'int'),
        FormFieldSpec(key: 'password', label: 'Password', kind: 'secret', required: true),
        FormFieldSpec(key: 'tls.fingerprint', label: 'Fingerprint', kind: 'choice', choices: ['', 'chrome', 'firefox'], advanced: true),
      ],
    ),
    ProtocolFormSpec(
      protocol: 'tor',
      label: 'Tor',
      fields: [
        FormFieldSpec(key: 'name', label: 'Name', kind: 'text'),
        FormFieldSpec(key: 'tor.builtin', label: 'Bridges', kind: 'choice', choices: ['', 'snowflake', 'obfs4']),
        FormFieldSpec(key: 'tor.bridges', label: 'Bridge lines', kind: 'multiline'),
      ],
    ),
  ];

  @override
  Future<ProfileInfo> buildProfile(Map<String, String> values, {String? protocol, ProfileInfo? profile}) async {
    final merged = {...?profile?.raw, ...values};
    final p = protocol ?? profile!.protocol;
    for (final f in (await forms()).firstWhere((f) => f.protocol == p).fields) {
      if (f.required && (merged[f.key] ?? '').toString().trim().isEmpty) {
        throw EngineException('BAD_REQUEST', '${f.label} is required');
      }
    }
    final server = (merged['server'] ?? '').toString();
    final port = int.tryParse((merged['port'] ?? '').toString()) ?? 443;
    final name = (merged['name'] ?? '').toString();
    return ProfileInfo.fromJson({
      ...?profile?.raw,
      ...values,
      'id': profile?.id ?? '$p-$server-$port',
      'name': name.isEmpty ? (server.isEmpty ? p : server) : name,
      'protocol': p,
      'server': server,
      'port': port,
    });
  }

  @override
  Future<Map<String, String>> profileValues(ProfileInfo profile) async => {
    'name': profile.name,
    'server': profile.server,
    'port': '${profile.port}',
    'password': profile.raw['password'] as String? ?? '',
  };

  @override
  Future<String> exportLink(ProfileInfo profile) async {
    if (profile.protocol != 'trojan') throw EngineException('BAD_REQUEST', 'no share link for ${profile.protocol}');
    return 'trojan://${Uri.encodeComponent(profile.raw['password'] as String? ?? '')}@${profile.server}:${profile.port}'
        '#${Uri.encodeComponent(profile.name)}';
  }

  @override
  void dispose() {
    _traffic?.cancel();
    _states.close();
    _logs.close();
  }
}
