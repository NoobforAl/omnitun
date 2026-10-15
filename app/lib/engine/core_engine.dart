import 'dart:async';
import 'dart:io';

import 'daemon_engine.dart';
import 'engine.dart';
import 'mock_engine.dart';

/// Engine used by the app: starts the real core in the background and
/// forwards to it. Falls back to [MockEngine] when the core is unavailable,
/// logging why, so the UI stays usable during development.
class CoreEngine implements VpnEngine {
  CoreEngine() {
    _ready = _start();
  }

  late final Future<VpnEngine> _ready;
  VpnEngine? _inner;
  final _states = StreamController<EngineState>.broadcast();
  final _logs = StreamController<String>.broadcast();
  final _subs = <StreamSubscription<Object?>>[];

  Future<VpnEngine> _start() async {
    VpnEngine engine;
    try {
      if (Platform.isAndroid || Platform.isIOS) {
        throw UnsupportedError('mobile core bridge is not wired yet');
      }
      final exe = findDaemonExecutable();
      if (exe == null) throw StateError('omnitun-svc not found (build it with scripts/go.ps1 build -o bin/ ./cmd/...)');
      engine = await DaemonEngine.launch(exe);
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
  }) async =>
      (await _ready).connect(profile, fallbacks: fallbacks, options: options);

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
