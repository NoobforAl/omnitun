import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'engine.dart';

/// Thrown for typed failures reported by the core.
class EngineException implements Exception {
  EngineException(this.code, this.message);

  final String code;
  final String message;

  @override
  String toString() => '$code: $message';
}

/// Talks to the omnitun-svc daemon over its loopback control protocol
/// (newline-delimited JSON, token-authenticated). See core/control.
///
/// The daemon is started as a child process with --parent-pid, so it stops the
/// tunnel and exits if this app dies; while connected, a ping keeps its lease.
class DaemonEngine implements VpnEngine {
  DaemonEngine._(this._socket, this._process);

  final Socket _socket;
  final Process? _process;
  final _states = StreamController<EngineState>.broadcast();
  final _logs = StreamController<String>.broadcast();
  final _pending = <int, Completer<dynamic>>{};
  int _nextId = 0;
  Timer? _ping;
  EngineState _state = const EngineState();

  /// Launches [executable] and connects to it.
  static Future<DaemonEngine> launch(String executable, {String? controlFile}) async {
    final process = await Process.start(executable, [
      '--parent-pid',
      '$pid',
      if (controlFile != null) ...['--control-file', controlFile],
    ]);
    final ready = Completer<String>();
    process.stdout.transform(utf8.decoder).transform(const LineSplitter()).listen((line) {
      if (line.startsWith('READY ') && !ready.isCompleted) ready.complete(line.substring(6).trim());
    });
    process.stderr.transform(utf8.decoder).listen((_) {});
    unawaited(process.exitCode.then((code) {
      if (!ready.isCompleted) ready.completeError(StateError('omnitun-svc exited with code $code'));
    }));
    await ready.future.timeout(const Duration(seconds: 15));
    final file = controlFile ?? defaultControlFile();
    final control = jsonDecode(await File(file).readAsString()) as Map<String, dynamic>;
    return _connect(control['addr'] as String, control['token'] as String, process);
  }

  /// Connects to an already running daemon (e.g. the Windows service).
  static Future<DaemonEngine> attach({String? controlFile}) async {
    final control =
        jsonDecode(await File(controlFile ?? defaultControlFile()).readAsString()) as Map<String, dynamic>;
    return _connect(control['addr'] as String, control['token'] as String, null);
  }

  /// Connects to a core whose address and token are already known (Android in-process core).
  static Future<DaemonEngine> attachTo(String addr, String token) => _connect(addr, token, null);

  static Future<DaemonEngine> _connect(String addr, String token, Process? process) async {
    final i = addr.lastIndexOf(':');
    final socket = await Socket.connect(addr.substring(0, i), int.parse(addr.substring(i + 1)),
        timeout: const Duration(seconds: 5));
    final engine = DaemonEngine._(socket, process);
    engine._listen();
    final hello = await engine._call('hello', {'token': token}) as Map<String, dynamic>;
    engine._applyState(hello['state'] as Map<String, dynamic>);
    engine._ping = Timer.periodic(const Duration(seconds: 4), (_) => engine._call('ping').ignore());
    return engine;
  }

  static String defaultControlFile() {
    final base = Platform.isWindows
        ? Platform.environment['APPDATA']!
        : Platform.isMacOS
            ? '${Platform.environment['HOME']}/Library/Application Support'
            : Platform.environment['XDG_CONFIG_HOME'] ?? '${Platform.environment['HOME']}/.config';
    return '$base${Platform.pathSeparator}omnitun${Platform.pathSeparator}control.json';
  }

  void _listen() {
    _socket.cast<List<int>>().transform(utf8.decoder).transform(const LineSplitter()).listen(
      _onLine,
      onDone: _onClosed,
      onError: (_) => _onClosed(),
    );
  }

  void _onLine(String line) {
    final msg = jsonDecode(line) as Map<String, dynamic>;
    final event = msg['event'] as String?;
    if (event == 'state') {
      _applyState(msg['data'] as Map<String, dynamic>);
    } else if (event == 'log') {
      final d = msg['data'] as Map<String, dynamic>;
      final time = (d['time'] as String).substring(11, 19);
      _logs.add('$time  ${(d['level'] as String).toUpperCase().padRight(5)} ${d['message']}');
    } else if (msg.containsKey('id')) {
      final c = _pending.remove(msg['id']);
      if (c == null) return;
      final err = msg['error'] as Map<String, dynamic>?;
      if (err != null) {
        c.completeError(EngineException(err['code'] as String, err['message'] as String));
      } else {
        c.complete(msg['result']);
      }
    }
  }

  void _onClosed() {
    for (final c in _pending.values) {
      c.completeError(EngineException('DISCONNECTED', 'lost connection to the core'));
    }
    _pending.clear();
    _ping?.cancel();
    _emit(const EngineState(status: ConnStatus.error, error: 'Lost connection to the core service'));
  }

  Future<dynamic> _call(String method, [Object? params]) {
    final id = ++_nextId;
    final c = Completer<dynamic>();
    _pending[id] = c;
    _socket.write('${jsonEncode({'id': id, 'method': method, 'params': ?params})}\n');
    return c.future;
  }

  void _emit(EngineState s) {
    _state = s;
    _states.add(s);
  }

  void _applyState(Map<String, dynamic> s) {
    final status = switch (s['status']) {
      'connected' => ConnStatus.connected,
      'connecting' => ConnStatus.connecting,
      'error' => ConnStatus.error,
      _ => ConnStatus.disconnected,
    };
    final since = s['since'] as String?;
    final selected = s['selected'] as String?;
    _emit(EngineState(
      status: status,
      activeProfileId: selected ?? _state.activeProfileId,
      connectedSince: since == null ? null : DateTime.parse(since).toLocal(),
      error: s['error'] as String?,
      upBps: (s['up_bps'] as num?)?.toInt() ?? 0,
      downBps: (s['down_bps'] as num?)?.toInt() ?? 0,
    ));
  }

  @override
  String get name => 'omnitun core (sing-box)';

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
    _emit(EngineState(status: ConnStatus.connecting, activeProfileId: profile.id));
    try {
      await _call('start', {
        'profiles': [profile.raw, for (final f in fallbacks) f.raw],
        'selected': profile.id,
        'mode': options.tunMode ? 'tun' : 'proxy',
        'listen_port': options.listenPort,
        'system_proxy': options.systemProxy,
      });
    } on EngineException catch (e) {
      _emit(EngineState(
        status: e.code == 'NO_NETWORK' ? ConnStatus.noInternet : ConnStatus.error,
        activeProfileId: profile.id,
        error: e.message,
      ));
    }
  }

  @override
  Future<void> disconnect() async {
    _emit(_state.copyWith(status: ConnStatus.disconnecting));
    await _call('stop');
  }

  @override
  Future<ImportResult> importText(String text) async {
    try {
      final res = await _call('import', {'text': text}) as Map<String, dynamic>;
      final profiles = [
        for (final p in (res['profiles'] as List? ?? const [])) ProfileInfo.fromJson(p as Map<String, dynamic>),
      ];
      final errors = [for (final e in (res['errors'] as List? ?? const [])) e as String];
      return ImportResult(profiles, errors);
    } on EngineException catch (e) {
      return ImportResult(const [], [e.message]);
    }
  }

  @override
  Future<int> testLatency(ProfileInfo profile) async {
    final res = await _call('test', {
      'profiles': [profile.raw],
    }) as List;
    return ((res.first as Map<String, dynamic>)['ms'] as num).toInt();
  }

  @override
  void dispose() {
    _ping?.cancel();
    _socket.destroy();
    _process?.kill();
    _states.close();
    _logs.close();
  }
}
