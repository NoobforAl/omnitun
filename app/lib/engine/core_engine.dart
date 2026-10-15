import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/services.dart';

import 'daemon_engine.dart';
import 'engine.dart';
import 'mock_engine.dart';

/// Engine used by the app: starts the real core in the background and
/// forwards to it. Desktop launches the omnitun-svc daemon; Android starts the
/// core inside the app process (core/mobile) via the "omnitun/core" channel.
/// Falls back to [MockEngine] when the core is unavailable, logging why.
class CoreEngine implements VpnEngine {
  CoreEngine() {
    _ready = _start();
  }

  static const _channel = MethodChannel('omnitun/core');

  late final Future<VpnEngine> _ready;
  VpnEngine? _inner;
  final _states = StreamController<EngineState>.broadcast();
  final _logs = StreamController<String>.broadcast();
  final _subs = <StreamSubscription<Object?>>[];

  Future<VpnEngine> _start() async {
    VpnEngine engine;
    try {
      if (Platform.isAndroid) {
        final info = jsonDecode(await _channel.invokeMethod<String>('start') ?? '{}') as Map<String, dynamic>;
        engine = await DaemonEngine.attachTo(info['addr'] as String, info['token'] as String);
        unawaited(_channel.invokeMethod<void>('requestNotifications'));
      } else if (Platform.isIOS) {
        throw UnsupportedError('iOS core bridge is not implemented yet');
      } else {
        final exe = findDaemonExecutable();
        if (exe == null) {
          throw StateError('omnitun-svc not found (build it with scripts/go.ps1 build -o bin/ ./cmd/...)');
        }
        engine = await DaemonEngine.launch(exe);
      }
    } catch (e) {
      _logs.add('core unavailable, using preview engine: $e');
      engine = MockEngine();
    }
    _subs
      ..add(engine.states.listen(_states.add))
      ..add(engine.logs.listen(_logs.add));
    _inner = engine;
    return engine;
  }

  /// Looks for the daemon next to the app, via OMNITUN_SVC, or in the repo's bin/ (development).
  static String? findDaemonExecutable() {
    final name = Platform.isWindows ? 'omnitun-svc.exe' : 'omnitun-svc';
    final sep = Platform.pathSeparator;
    final candidates = <String>[
      ?Platform.environment['OMNITUN_SVC'],
      '${File(Platform.resolvedExecutable).parent.path}$sep$name',
    ];
    var dir = Directory.current;
    for (var i = 0; i < 4; i++) {
      candidates.add('${dir.path}${sep}bin$sep$name');
      dir = dir.parent;
    }
    for (final c in candidates) {
      if (File(c).existsSync()) return c;
    }
    return null;
  }

  @override
  String get name => _inner?.name ?? 'starting…';

  @override
  Stream<EngineState> get states => _states.stream;

  @override
  Stream<String> get logs => _logs.stream;

  @override
  Future<void> connect(
    ProfileInfo profile, {
    List<ProfileInfo> fallbacks = const [],
    ConnectOptions options = const ConnectOptions(),
  }) async {
    final engine = await _ready;
    if (Platform.isAndroid && options.tunMode && engine is DaemonEngine) {
      // The system consent dialog must be accepted before the core can open a TUN.
      final granted = await _channel.invokeMethod<bool>('prepareVpn') ?? false;
      if (!granted) {
        _states.add(EngineState(
          status: ConnStatus.error,
          activeProfileId: profile.id,
          error: 'VPN permission was not granted',
        ));
        return;
      }
    }
    return engine.connect(profile, fallbacks: fallbacks, options: options);
  }

  @override
  Future<void> disconnect() async => (await _ready).disconnect();

  @override
  Future<ImportResult> importText(String text) async => (await _ready).importText(text);

  @override
  Future<int> testLatency(ProfileInfo profile) async => (await _ready).testLatency(profile);

  @override
  void dispose() {
    for (final s in _subs) {
      s.cancel();
    }
    _inner?.dispose();
    _states.close();
    _logs.close();
  }
}
