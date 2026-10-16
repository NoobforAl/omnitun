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

  test('the profile editor of the real core: forms, build, edit, read back and share', () async {
    final dir = await Directory.systemTemp.createTemp('omnitun-test');
    final engine = await DaemonEngine.launch(exe!, controlFile: '${dir.path}${Platform.pathSeparator}control.json');
    addTearDown(() async {
      engine.dispose();
      await Future<void>.delayed(const Duration(milliseconds: 300));
      await dir.delete(recursive: true).catchError((_) => dir);
    });

    final forms = await engine.forms();
    expect(forms.map((f) => f.protocol), containsAll(['vless', 'trojan', 'wireguard', 'amneziawg', 'tor']));
    final trojan = forms.firstWhere((f) => f.protocol == 'trojan');
    expect(trojan.fields.firstWhere((f) => f.key == 'password').kind, 'secret');
    expect(trojan.fields.firstWhere((f) => f.key == 'server').required, isTrue);
    expect(trojan.fields.where((f) => f.advanced), isNotEmpty);
    final tor = forms.firstWhere((f) => f.protocol == 'tor');
    expect(tor.fields.firstWhere((f) => f.key == 'tor.builtin').choices, containsAll(['', 'snowflake', 'obfs4']));

    // The core's own message reaches the app.
    await expectLater(
      engine.buildProfile({'server': 'de1.example.com'}, protocol: 'trojan'),
      throwsA(isA<EngineException>().having((e) => e.message, 'message', contains('Password'))),
    );

    final built = await engine.buildProfile({
      'name': 'Berlin',
      'server': 'de1.example.com',
      'password': 'pw',
      'tls.server_name': 'de1.example.com',
    }, protocol: 'trojan');
    expect(built.name, 'Berlin');
    expect(built.port, 443);
    expect(built.id, isNotEmpty);

    final edited = await engine.buildProfile({'name': 'Berlin 2', 'port': '8443'}, profile: built);
    expect(edited.id, built.id);
    expect(edited.port, 8443);
    expect(edited.raw['password'], 'pw'); // what the editor does not touch is kept

    final values = await engine.profileValues(edited);
    expect(values['name'], 'Berlin 2');
    expect(values['port'], '8443');
    expect(values['tls.server_name'], 'de1.example.com');

    final link = await engine.exportLink(edited);
    expect(link, startsWith('trojan://pw@de1.example.com:8443'));
    final again = await engine.importText(link);
    expect(again.profiles.single.server, 'de1.example.com');

    final torProfile = await engine.buildProfile({'tor.builtin': 'snowflake'}, protocol: 'tor');
    expect(torProfile.protocol, 'tor');
    expect(await engine.exportLink(torProfile), 'tor://snowflake');
  }, skip: exe == null ? 'omnitun-svc not built' : false, timeout: const Timeout(Duration(seconds: 60)));

  test('a subscription can ask to go through the VPN; without one the core says so', () async {
    final server = await HttpServer.bind(InternetAddress.loopbackIPv4, 0);
    addTearDown(() => server.close(force: true));
    server.listen((req) {
      req.response.write('trojan://pw@a.example.com:443#A');
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

    final auto = await engine.fetchSubscription({'url': url, 'auth': {'type': 'none'}, 'via': 'auto'});
    expect(auto.profiles.single.name, 'A');
    await expectLater(
      engine.fetchSubscription({'url': url, 'auth': {'type': 'none'}, 'via': 'tunnel'}),
      throwsA(isA<EngineException>().having((e) => e.message, 'message', contains('VPN'))),
    );
  }, skip: exe == null ? 'omnitun-svc not built' : false, timeout: const Timeout(Duration(seconds: 60)));

  test('the core checks a bypass list; the connect request carries it', () async {
    final dir = await Directory.systemTemp.createTemp('omnitun-test');
    final engine = await DaemonEngine.launch(exe!, controlFile: '${dir.path}${Platform.pathSeparator}control.json');
    addTearDown(() async {
      engine.dispose();
      await Future<void>.delayed(const Duration(milliseconds: 300));
      await dir.delete(recursive: true).catchError((_) => dir);
    });
    final report = await engine.parseBypass(
      '# banks\nbank.example\nfull:www.bank.example\n10.0.0.0/8\nnot a name\ngeosite:ads\n',
    );
    expect(report.entries, 3);
    expect(report.domains, 2);
    expect(report.ranges, 1);
    expect(report.issues.map((i) => i.line), [5, 6]);
    expect(report.issues.first.message, isNotEmpty);

    final profile = (await engine.importText('trojan://pw@example.com:443#t')).profiles.single;
    final params = DaemonEngine.startParams(
      profile,
      const [],
      const ConnectOptions(bypass: 'bank.example\n10.0.0.0/8', bypassOnly: true),
    );
    expect(params['bypass'], 'bank.example\n10.0.0.0/8');
    expect(params['bypass_only'], true);
    // No list: "only these" means nothing, so it is not sent.
    expect(DaemonEngine.startParams(profile, const [], const ConnectOptions(bypassOnly: true)).containsKey('bypass_only'), isFalse);
  }, skip: exe == null ? 'omnitun-svc not built' : false, timeout: const Timeout(Duration(seconds: 60)));
}
