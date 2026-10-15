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

    final states = <ConnStatus>[];
    final sub = engine.states.listen((s) => states.add(s.status));
    await engine.connect(
      imported.profiles.single,
      options: ConnectOptions(systemProxy: false, listenPort: port),
    );
    await Future<void>.delayed(const Duration(milliseconds: 300));
    if (states.contains(ConnStatus.noInternet)) {
      markTestSkipped('no network available for the pre-connect check');
      return;
    }
    expect(states.last, ConnStatus.connected);

    // The local proxy port is open while connected.
    final probe = await Socket.connect(InternetAddress.loopbackIPv4, port);
    probe.destroy();

    await engine.disconnect();
    await Future<void>.delayed(const Duration(milliseconds: 300));
    expect(states.last, ConnStatus.disconnected);
    await sub.cancel();

    final dead = await engine.importText('ss://aes-128-gcm:x@127.0.0.1:$port#dead');
    expect(await engine.testLatency(dead.profiles.single), -1);
  }, skip: exe == null ? 'omnitun-svc not built' : false, timeout: const Timeout(Duration(seconds: 60)));
}
