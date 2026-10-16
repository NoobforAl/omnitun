import 'dart:convert';
import 'dart:io';
import 'dart:math';

import 'package:crypto/crypto.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:omnitun/state/ipdb.dart';

const _v4 = '1.0.0.0,1.0.0.255,AU\n'
    '1.0.1.0,1.0.1.255,AU\n'
    '1.0.2.0,1.0.3.255,CN\n'
    '5.0.0.0,5.0.0.9,ZZ\n'
    '8.8.8.0,8.8.8.255,US\n'
    '203.0.113.0,203.0.113.255,DE\n'
    '255.255.255.0,255.255.255.255,FR\n';

const _v6 = '2001:200::,2001:200:ffff:ffff:ffff:ffff:ffff:ffff,JP\n'
    '2a07:d880::,2a07:d883:ffff:ffff:ffff:ffff:ffff:ffff,DE\n'
    'ff00::,ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff,NL\n';

void main() {
  late Directory tmp;
  setUp(() => tmp = Directory.systemTemp.createTempSync('ipdb_test'));
  tearDown(() => tmp.deleteSync(recursive: true));

  Future<IpCountryDb> build() async {
    File('${tmp.path}/v4.csv').writeAsStringSync(_v4);
    File('${tmp.path}/v6.csv').writeAsStringSync(_v6);
    final built = await compileIpDb('${tmp.path}/v4.csv', '${tmp.path}/v6.csv', '${tmp.path}/db.bin');
    // Neighbouring ranges of one country are one range; "ZZ" (unknown) is dropped.
    expect(built.v4Ranges, 5);
    expect(built.v6Ranges, 3);
    return IpCountryDb.fromBytes(File('${tmp.path}/db.bin').readAsBytesSync());
  }

  test('addresses are found in IPv4 and IPv6, and only where the data has them', () async {
    final db = await build();
    expect(db.lookup('1.0.0.1'), 'AU');
    expect(db.lookup('1.0.1.200'), 'AU'); // the merged range
    expect(db.lookup('1.0.3.255'), 'CN');
    expect(db.lookup('8.8.8.8'), 'US');
    expect(db.lookup('203.0.113.77'), 'DE');
    expect(db.lookup('255.255.255.255'), 'FR');
    expect(db.lookup('1.0.4.0'), isNull); // between ranges
    expect(db.lookup('5.0.0.1'), isNull); // "ZZ"
    expect(db.lookup('0.0.0.1'), isNull);
    expect(db.lookup('2001:200:1::1'), 'JP');
    expect(db.lookup('2a07:d883:703:2::e018'), 'DE');
    expect(db.lookup('ff02::1'), 'NL'); // the top bit of the number is set: unsigned comparison
    expect(db.lookup('2001:300::'), isNull);
    expect(db.lookup('::ffff:8.8.8.8'), 'US'); // an IPv4 address in IPv6 clothes
    expect(db.lookup('not an address'), isNull);
  });

  test('a damaged or cut file is refused', () async {
    await build();
    final bytes = File('${tmp.path}/db.bin').readAsBytesSync();
    expect(() => IpCountryDb.fromBytes(bytes.sublist(0, bytes.length - 8)), throwsFormatException);
    expect(() => IpCountryDb.fromBytes(bytes.sublist(1)), throwsFormatException);
  });

  group('download', () {
    // Two CSV files served with Range support; one request can be made to die half-way.
    late HttpServer server;
    late Map<String, List<int>> files;
    final ranges = <String>[];
    var dieAfter = -1;

    List<int> make(bool v6, int lines) {
      final rnd = Random(3);
      final b = StringBuffer();
      var n = 0;
      String ip(int v) => '${(v >> 24) & 255}.${(v >> 16) & 255}.${(v >> 8) & 255}.${v & 255}';
      for (var i = 0; i < lines; i++) {
        final cc = String.fromCharCodes([65 + rnd.nextInt(3), 65 + rnd.nextInt(3)]);
        if (v6) {
          final h = (0x200 + i * 2).toRadixString(16);
          b.writeln('2001:$h::,2001:$h:ffff:ffff:ffff:ffff:ffff,$cc');
        } else {
          b.writeln('${ip(n)},${ip(n + 255)},$cc');
          n += 512;
        }
      }
      return b.toString().codeUnits;
    }

    setUp(() async {
      files = {'a4.csv': make(false, 4000), 'a6.csv': make(true, 3000)};
      ranges.clear();
      dieAfter = -1;
      server = await HttpServer.bind('127.0.0.1', 0);
      server.listen((req) async {
        final data = files[req.uri.pathSegments.last];
        if (data == null) {
          req.response.statusCode = 404;
          await req.response.close();
          return;
        }
        // A client that asks for compression gets a compressed body that ignores ranges (as some CDNs do):
        // sizes and offsets would then mean different things from one request to the next.
        if ((req.headers.value('accept-encoding') ?? '').contains('gzip')) {
          final zipped = gzip.encode(data);
          req.response.headers.set('content-encoding', 'gzip');
          req.response.headers.contentLength = zipped.length;
          req.response.add(zipped);
          await req.response.close();
          return;
        }
        final range = req.headers.value('range');
        ranges.add('${req.uri.pathSegments.last} $range');
        var start = 0, end = data.length - 1;
        if (range != null) {
          final m = RegExp(r'bytes=(\d+)-(\d*)').firstMatch(range)!;
          start = int.parse(m.group(1)!);
          if (m.group(2)!.isNotEmpty) end = int.parse(m.group(2)!);
          req.response.statusCode = 206;
          req.response.headers.set('content-range', 'bytes $start-$end/${data.length}');
        }
        req.response.headers.contentLength = end - start + 1;
        final body = data.sublist(start, end + 1);
        if (dieAfter >= 0 && range != 'bytes=0-0') {
          final cut = dieAfter;
          dieAfter = -1; // only once
          // Send the headers and part of the body, then drop the connection.
          final socket = await req.response.detachSocket(writeHeaders: true);
          socket.add(body.sublist(0, cut));
          await socket.flush();
          socket.destroy();
          return;
        }
        // In small slices, as a real connection delivers it.
        for (var i = 0; i < body.length; i += 4096) {
          req.response.add(body.sublist(i, min(i + 4096, body.length)));
          await req.response.flush();
          await Future<void>.delayed(const Duration(milliseconds: 5));
        }
        await req.response.close();
      });
    });
    tearDown(() => server.close(force: true));

    String path(String f) => f == ipDbFiles[0] ? 'a4.csv' : 'a6.csv';

    // What the data API would say about the two files.
    IpDbSource source() => IpDbSource('test', {
      for (final f in ipDbFiles)
        f: IpDbFile(files[path(f)]!.length, base64.encode(sha256.convert(files[path(f)]!).bytes)),
    });

    IpDbDownloader downloader(Directory dir, {List<String> Function(String, String)? urls, int rounds = 1}) =>
        IpDbDownloader(
          dir: dir,
          urls: urls ?? (f, v) => ['http://127.0.0.1:${server.port}/${path(f)}'],
          resolve: () async => source(),
          retryDelay: Duration.zero,
          rounds: rounds,
        );

    test('a download that fails half-way carries on from the last byte, not from the start', () async {
      final dir = Directory('${tmp.path}/dl');
      dieAfter = 20000;
      await expectLater(downloader(dir).download((_) {}, IpDbCancel()), throwsA(isA<IpDbDownloadException>()));

      // What was fetched is on disk and can be shown without a network.
      final partial = await downloader(dir).onDisk();
      expect(partial, isNotNull);
      expect(partial!.overallReceived, greaterThan(0));
      expect(partial.overallReceived, lessThan(partial.overallTotal));
      final kept = File('${dir.path}/${ipDbFiles[0]}.part').lengthSync();
      expect(kept, greaterThan(0));

      final seen = <double>[];
      await downloader(dir).download((p) => seen.add(p.fraction), IpDbCancel());
      expect(ranges.any((r) => r.endsWith('bytes=$kept-')), isTrue, reason: 'resumed at byte $kept: $ranges');
      expect(File('${dir.path}/${ipDbFiles[0]}').readAsBytesSync(), files['a4.csv']);
      expect(File('${dir.path}/${ipDbFiles[1]}').readAsBytesSync(), files['a6.csv']);
      expect(seen.last, 1.0);
      expect(seen.first, greaterThan(0)); // the first report is already past where it stopped
      for (var i = 1; i < seen.length; i++) {
        expect(seen[i], greaterThanOrEqualTo(seen[i - 1]));
      }
    });

    test('a dead mirror is skipped and the next one continues', () async {
      final dead = await HttpServer.bind('127.0.0.1', 0);
      final deadPort = dead.port;
      await dead.close(force: true);
      final dir = Directory('${tmp.path}/dl2');
      await downloader(
        dir,
        rounds: 2,
        urls: (f, v) => ['http://127.0.0.1:$deadPort/x', 'http://127.0.0.1:${server.port}/${path(f)}'],
      ).download((_) {}, IpDbCancel());
      expect(File('${dir.path}/${ipDbFiles[0]}').readAsBytesSync(), files['a4.csv']);
      expect(File('${dir.path}/${ipDbFiles[1]}').readAsBytesSync(), files['a6.csv']);
    });

    test('stopping keeps the bytes, and the next start continues', () async {
      final dir = Directory('${tmp.path}/dl3');
      final cancel = IpDbCancel();
      var stopped = false;
      await expectLater(
        downloader(dir).download((p) {
          if (!stopped && p.received > 5000) {
            stopped = true;
            cancel.cancel();
          }
        }, cancel),
        throwsA(isA<IpDbDownloadException>()),
      );
      expect(File('${dir.path}/${ipDbFiles[0]}.part').lengthSync(), greaterThan(5000));
      await downloader(dir).download((_) {}, IpDbCancel());
      expect(File('${dir.path}/${ipDbFiles[0]}').readAsBytesSync(), files['a4.csv']);
    });

    test('a mirror that reports a different size is skipped, and a damaged file is refused', () async {
      // The first "mirror" serves a smaller file under the same name (as the CDNs that answer ranges
      // against a compressed copy do); the second is right.
      final liar = await HttpServer.bind('127.0.0.1', 0);
      addTearDown(() => liar.close(force: true));
      var liarAsked = 0;
      liar.listen((req) async {
        liarAsked++;
        const total = 2098375;
        req.response.statusCode = 206;
        req.response.headers.set('content-range', 'bytes 0-0/$total');
        req.response.headers.contentLength = 1;
        req.response.add([65]);
        await req.response.close();
      });
      final dir = Directory('${tmp.path}/dl4');
      await downloader(
        dir,
        urls: (f, v) => ['http://127.0.0.1:${liar.port}/${path(f)}', 'http://127.0.0.1:${server.port}/${path(f)}'],
      ).download((_) {}, IpDbCancel());
      expect(File('${dir.path}/${ipDbFiles[0]}').readAsBytesSync(), files['a4.csv']);
      expect(liarAsked, 1, reason: 'a mirror that disagreed is not asked again for the next file');

      // Same size, wrong bytes: the checksum catches it, nothing is kept.
      final dir2 = Directory('${tmp.path}/dl5');
      final good = source(); // the hashes of the real files
      files['a4.csv'] = List<int>.from(files['a4.csv']!)..[100] ^= 0xFF; // one byte differs
      final wrong = IpDbDownloader(
        dir: dir2,
        urls: (f, v) => ['http://127.0.0.1:${server.port}/${path(f)}'],
        resolve: () async => good,
        retryDelay: Duration.zero,
        rounds: 1,
      );
      await expectLater(wrong.download((_) {}, IpDbCancel()), throwsA(isA<IpDbDownloadException>()));
      expect(File('${dir2.path}/${ipDbFiles[0]}').existsSync(), isFalse);
      expect(File('${dir2.path}/${ipDbFiles[0]}.part').existsSync(), isFalse);
    });
  });
}
