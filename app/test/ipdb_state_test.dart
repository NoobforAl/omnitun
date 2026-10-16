import 'dart:convert';
import 'dart:io';

import 'package:crypto/crypto.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:omnitun/engine/engine.dart';
import 'package:omnitun/engine/mock_engine.dart';
import 'package:omnitun/state/ipdb.dart';
import 'package:omnitun/state/ipdb_state.dart';
import 'package:omnitun/state/providers.dart';
import 'package:omnitun/state/store.dart';

const _v4 = '1.0.0.0,1.0.0.255,AU\n203.0.113.0,203.0.113.255,DE\n';
const _v6 = '2a07:d880::,2a07:d883:ffff:ffff:ffff:ffff:ffff:ffff,DE\n';

ProfileInfo _profile(String server) => ProfileInfo(
  id: 'p-$server',
  name: 'Server',
  protocol: 'trojan',
  server: server,
  port: 443,
  raw: {'id': 'p-$server', 'name': 'Server', 'protocol': 'trojan', 'server': server, 'port': 443},
);

void main() {
  test('downloading the database makes a server known by its address, and removing it takes that away', () async {
    final tmp = Directory.systemTemp.createTempSync('ipdb_state');
    addTearDown(() => tmp.deleteSync(recursive: true));
    final server = await HttpServer.bind('127.0.0.1', 0);
    addTearDown(() => server.close(force: true));
    final files = {'a4.csv': _v4, 'a6.csv': _v6};
    server.listen((req) async {
      final data = files[req.uri.pathSegments.last]!.codeUnits;
      final range = req.headers.value('range');
      var start = 0, end = data.length - 1;
      if (range != null) {
        final m = RegExp(r'bytes=(\d+)-(\d*)').firstMatch(range)!;
        start = int.parse(m.group(1)!);
        if (m.group(2)!.isNotEmpty) end = int.parse(m.group(2)!);
        req.response.statusCode = 206;
        req.response.headers.set('content-range', 'bytes $start-$end/${data.length}');
      }
      req.response.headers.contentLength = end - start + 1;
      req.response.add(data.sublist(start, end + 1));
      await req.response.close();
    });

    final c = ProviderContainer(
      overrides: [
        storeProvider.overrideWithValue(AppStore.memory({'settings': {'show_exit': false}})),
        engineProvider.overrideWithValue(MockEngine(connectDelay: Duration.zero)),
        ipDbDirProvider.overrideWithValue(() async => Directory('${tmp.path}/ipdb')),
        ipDbDownloaderProvider.overrideWithValue(
          (dir) => IpDbDownloader(
            dir: dir,
            urls: (f, v) => ['http://127.0.0.1:${server.port}/${f == ipDbFiles[0] ? 'a4.csv' : 'a6.csv'}'],
            resolve: () async => IpDbSource('1.2.3', {
              ipDbFiles[0]: IpDbFile(_v4.length, base64.encode(sha256.convert(_v4.codeUnits).bytes)),
              ipDbFiles[1]: IpDbFile(_v6.length, base64.encode(sha256.convert(_v6.codeUnits).bytes)),
            }),
            retryDelay: Duration.zero,
          ),
        ),
      ],
    );
    addTearDown(c.dispose);

    final raw = _profile('203.0.113.9');
    c.read(profilesProvider.notifier).addAll([raw]);
    final profile = c.read(profilesProvider).single;
    expect(c.read(profileCountryProvider)(profile), isNull); // the name says nothing; no database yet

    await c.read(ipDbProvider.notifier).start();
    final s = c.read(ipDbProvider);
    expect(s.phase, IpDbPhase.ready);
    expect(s.info!.version, '1.2.3');
    expect(s.info!.v4, 2);
    expect(c.read(ipDbActiveProvider)!.lookup('203.0.113.77'), 'DE');
    // The files are gone, only the compiled database stays.
    expect(Directory('${tmp.path}/ipdb/staging').listSync(), isEmpty);
    expect(File('${tmp.path}/ipdb/ipcountry.bin').existsSync(), isTrue);

    // The server's country now comes from the database (resolved in the background).
    await Future<void>.delayed(const Duration(milliseconds: 100));
    expect(c.read(profileCountryProvider)(profile), 'DE');

    // Switched off, it says nothing; switched on again it does.
    c.read(ipDbProvider.notifier).setEnabled(false);
    expect(c.read(ipDbActiveProvider), isNull);
    expect(c.read(profileCountryProvider)(profile), isNull);
    c.read(ipDbProvider.notifier).setEnabled(true);
    expect(c.read(profileCountryProvider)(profile), 'DE');

    // An address a lookup service answered without a country gets one from the database.
    expect(withDbCountry(const ExitInfo('203.0.113.5'), c.read(ipDbActiveProvider)).country, 'DE');
    expect(withDbCountry(const ExitInfo('9.9.9.9'), c.read(ipDbActiveProvider)).country, isNull);

    await c.read(ipDbProvider.notifier).delete();
    expect(c.read(ipDbProvider).phase, IpDbPhase.none);
    expect(c.read(ipDbActiveProvider), isNull);
    expect(Directory('${tmp.path}/ipdb').existsSync(), isFalse);
  });
}
