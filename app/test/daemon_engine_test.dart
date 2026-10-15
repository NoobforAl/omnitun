import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:omnitun/engine/core_engine.dart';
import 'package:omnitun/engine/daemon_engine.dart';
import 'package:omnitun/engine/engine.dart';

/// Drives the real Go daemon. Skipped when bin/omnitun-svc has not been built.
void main() {
  final exe = CoreEngine.findDaemonExecutable();

  test('import, connect and disconnect through omnitun-svc', () async {
    final dir = await Directory.systemTemp.createTemp('omnitun-test');
    final engine = await DaemonEngine.launch(exe!, controlFile: '${dir.path}${Platform.pathSeparator}control.json');
    addTearDown(() async {
      engine.dispose();
      await Future<void>.delayed(const Duration(milliseconds: 300));
      await dir.delete(recursive: true).catchError((_) => dir);
    });

    final imported = await engine.importText('trojan://pw@example.com:443?sni=example.com#Remote\nbad line');
    expect(imported.profiles.single.name, 'Remote');
    expect(imported.profiles.single.raw['password'], 'pw');
    expect(imported.errors, hasLength(1));

    final socket = await ServerSocket.bind(InternetAddress.loopbackIPv4, 0);
    final port = socket.port;
    await socket.close();

    // A profile nothing listens on: the tunnel comes up, then the handshake
    // fails and the core reports that stage instead of claiming "connected".
    final dead = await engine.importText('ss://aes-128-gcm:x@127.0.0.1:$port#dead');
    final states = <EngineState>[];
    final sub = engine.states.listen(states.add);
    await engine.connect(
      dead.profiles.single,
      options: ConnectOptions(systemProxy: false, listenPort: port + 1, autoFailover: false),
    );
    final deadline = DateTime.now().add(const Duration(seconds: 20));
    while (states.isEmpty || states.last.status != ConnStatus.error) {
      if (DateTime.now().isAfter(deadline)) fail('no error reported: ${states.map((s) => s.status).toList()}');
      await Future<void>.delayed(const Duration(milliseconds: 100));
    }
    await sub.cancel();
    if (states.last.error == 'No internet connection' || states.any((s) => s.status == ConnStatus.noInternet)) {
      markTestSkipped('no network available for the pre-connect check');
      return;
    }
    expect(states.any((s) => s.status == ConnStatus.connected), isFalse);
    final stages = states.map((s) => s.progress?.stage).nonNulls.toSet();
    expect(stages, containsAll(['network_check', 'prepare', 'engine_start', 'tunnel_up', 'handshake']));
    expect(states.last.progress?.stage, 'handshake');
    expect(states.last.error, startsWith('Contacting server failed'));

    expect(await engine.testLatency(dead.profiles.single), -1);
  }, skip: exe == null ? 'omnitun-svc not built' : false, timeout: const Timeout(Duration(seconds: 60)));

  test('fetches an authenticated subscription through omnitun-svc', () async {
    final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    addTearDown(() => server.close(force: true));
    server.listen((req) {
      if (req.headers.value('authorization') != 'Bearer s3cret') {
        req.response.statusCode = HttpStatus.unauthorized;
      } else {
        req.response.headers.set('subscription-userinfo', 'upload=10; download=20; total=100; expire=1893456000');
        req.response.write(base64.encode(utf8.encode('trojan://pw@a.example.com:443#A\nhy2://pw@b.example.com:443#B')));
      }
      req.response.close();
    });

    final dir = await Directory.systemTemp.createTemp('omnitun-test');
    final engine = await DaemonEngine.launch(exe!, controlFile: '${dir.path}${Platform.pathSeparator}control.json');
    addTearDown(() async {
      engine.dispose();
      await Future<void>.delayed(const Duration(milliseconds: 300));
      await dir.delete(recursive: true).catchError((_) => dir);
    });

    final url = 'http://127.0.0.1:${server.port}/sub';
    final res = await engine.fetchSubscription({
      'url': url,
      'auth': {'type': 'bearer', 'token': 's3cret'},
    });
    expect(res.profiles.map((p) => p.name), ['A', 'B']);
    expect(res.info.used, 30);
    expect(res.info.total, 100);
    expect(res.info.expire?.year, 2030);

    await expectLater(
      engine.fetchSubscription({
        'url': url,
        'auth': {'type': 'bearer', 'token': 'wrong'},
      }),
      throwsA(isA<EngineException>().having((e) => e.code, 'code', 'AUTH_FAILED')),
    );
  }, skip: exe == null ? 'omnitun-svc not built' : false, timeout: const Timeout(Duration(seconds: 60)));
}
