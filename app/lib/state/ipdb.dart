/// An offline IP-address → country database: the file format, the lookup, the compiler that builds it
/// from the public CSV data, and a downloader that can stop and carry on where it stopped.
///
/// Data: "geo-whois-asn-country" of the ip-location-db project (CC0, built from the regional internet
/// registries' whois data and ASN announcements), published on npm and served by several CDNs.
library;

import 'dart:async';
import 'dart:convert';
import 'dart:io';
import 'dart:isolate';
import 'dart:math' as math;
import 'dart:typed_data';

import 'package:crypto/crypto.dart';

// ---------------------------------------------------------------------------------------------------
// The database in memory

/// Country by address, for IPv4 (exact) and IPv6 (by the upper 64 bits, which is all the registries
/// allocate). Ranges are sorted and never overlap, so a lookup is a binary search.
class IpCountryDb {
  IpCountryDb._(this.v4Start, this.v4End, this.v4Cc, this.v6Start, this.v6End, this.v6Cc);

  final Uint32List v4Start, v4End;
  final Uint16List v4Cc;
  final Uint64List v6Start, v6End;
  final Uint16List v6Cc;

  int get v4Count => v4Start.length;
  int get v6Count => v6Start.length;

  // The file: "OIPC", version 1, the two counts, then v4 starts, ends, countries and the same for v6,
  // each block padded to 8 bytes, little endian (every platform the app runs on).
  static const _magic = [0x4F, 0x49, 0x50, 0x43];

  static int _pad8(int n) => (n + 7) & ~7;

  static ({int v4s, int v4e, int v4c, int v6s, int v6e, int v6c, int end}) _layout(int n4, int n6) {
    var o = 16;
    final v4s = o;
    o += _pad8(n4 * 4);
    final v4e = o;
    o += _pad8(n4 * 4);
    final v4c = o;
    o += _pad8(n4 * 2);
    final v6s = o;
    o += n6 * 8;
    final v6e = o;
    o += n6 * 8;
    final v6c = o;
    o += _pad8(n6 * 2);
    return (v4s: v4s, v4e: v4e, v4c: v4c, v6s: v6s, v6e: v6e, v6c: v6c, end: o);
  }

  /// Reads a database file; throws [FormatException] when it is not one (or is cut short).
  static IpCountryDb fromBytes(Uint8List bytes) {
    if (bytes.length < 16 || [for (var i = 0; i < 4; i++) bytes[i]].toString() != _magic.toString() || bytes[4] != 1) {
      throw const FormatException('not an Omnitun IP database');
    }
    final head = ByteData.sublistView(bytes);
    final n4 = head.getUint32(8, Endian.little), n6 = head.getUint32(12, Endian.little);
    final l = _layout(n4, n6);
    if (bytes.length < l.end) throw const FormatException('the IP database is incomplete');
    // Typed views need an aligned start, which a Uint8List cut from a larger buffer may not have.
    final aligned = (bytes.offsetInBytes & 7) == 0 ? bytes : Uint8List.fromList(bytes);
    final buf = aligned.buffer;
    final base = aligned.offsetInBytes;
    return IpCountryDb._(
      buf.asUint32List(base + l.v4s, n4),
      buf.asUint32List(base + l.v4e, n4),
      buf.asUint16List(base + l.v4c, n4),
      buf.asUint64List(base + l.v6s, n6),
      buf.asUint64List(base + l.v6e, n6),
      buf.asUint16List(base + l.v6c, n6),
    );
  }

  /// The two-letter country of [ip], or null when it is not in the database or not an address.
  String? lookup(String ip) {
    final a = InternetAddress.tryParse(ip.trim());
    if (a == null) return null;
    final raw = a.rawAddress;
    if (raw.length == 4) return _v4(raw[0] << 24 | raw[1] << 16 | raw[2] << 8 | raw[3]);
    // ::ffff:a.b.c.d is an IPv4 address in IPv6 clothes.
    var mapped = true;
    for (var i = 0; i < 10; i++) {
      if (raw[i] != 0) mapped = false;
    }
    if (mapped && raw[10] == 0xff && raw[11] == 0xff) {
      return _v4(raw[12] << 24 | raw[13] << 16 | raw[14] << 8 | raw[15]);
    }
    var hi = 0;
    for (var i = 0; i < 8; i++) {
      hi = (hi << 8) | raw[i];
    }
    return _v6(hi);
  }

  String? _v4(int ip) {
    var lo = 0, hi = v4Start.length - 1, at = -1;
    while (lo <= hi) {
      final mid = (lo + hi) >> 1;
      if (v4Start[mid] <= ip) {
        at = mid;
        lo = mid + 1;
      } else {
        hi = mid - 1;
      }
    }
    return at >= 0 && ip <= v4End[at] ? _code(v4Cc[at]) : null;
  }

  // IPv6 values are unsigned 64-bit numbers held in signed ints: flip the top bit to compare them.
  static const _flip = -0x8000000000000000;

  String? _v6(int ip) {
    final x = ip ^ _flip;
    var lo = 0, hi = v6Start.length - 1, at = -1;
    while (lo <= hi) {
      final mid = (lo + hi) >> 1;
      if ((v6Start[mid] ^ _flip) <= x) {
        at = mid;
        lo = mid + 1;
      } else {
        hi = mid - 1;
      }
    }
    return at >= 0 && x <= (v6End[at] ^ _flip) ? _code(v6Cc[at]) : null;
  }

  static String _code(int cc) => String.fromCharCodes([cc >> 8, cc & 0xFF]);
}

// ---------------------------------------------------------------------------------------------------
// Building it from the CSV files

/// What [compileIpDb] made.
class IpDbBuilt {
  const IpDbBuilt(this.v4Ranges, this.v6Ranges, this.bytes);

  final int v4Ranges, v6Ranges, bytes;
}

/// Reads the two CSV files (`first,last,CC` per line) and writes the database file. Runs in its own
/// isolate: a few hundred thousand lines would otherwise stall the screen.
Future<IpDbBuilt> compileIpDb(String v4Csv, String v6Csv, String outPath) =>
    Isolate.run(() => _compile(v4Csv, v6Csv, outPath));

IpDbBuilt _compile(String v4Csv, String v6Csv, String outPath) {
  final v4s = <int>[], v4e = <int>[], v4c = <int>[];
  final v6s = <int>[], v6e = <int>[], v6c = <int>[];

  int? ip4(String s) {
    final p = s.split('.');
    if (p.length != 4) return null;
    var v = 0;
    for (final part in p) {
      final n = int.tryParse(part);
      if (n == null || n < 0 || n > 255) return null;
      v = (v << 8) | n;
    }
    return v;
  }

  int? ip6hi(String s) {
    final a = InternetAddress.tryParse(s);
    if (a == null || a.rawAddress.length != 16) return null;
    var hi = 0;
    for (var i = 0; i < 8; i++) {
      hi = (hi << 8) | a.rawAddress[i];
    }
    return hi;
  }

  void read(String path, int? Function(String) parse, List<int> s, List<int> e, List<int> c) {
    final lines = File(path).readAsLinesSync();
    for (final line in lines) {
      final f = line.split(',');
      if (f.length < 3) continue;
      final cc = f[2].trim();
      if (cc.length != 2 || cc == 'ZZ' || cc == 'XX') continue;
      final a = parse(f[0].trim()), b = parse(f[1].trim());
      if (a == null || b == null) continue;
      final code = cc.codeUnitAt(0) << 8 | cc.codeUnitAt(1);
      // Neighbouring ranges of one country become one range: the file shrinks a lot.
      if (s.isNotEmpty && c.last == code && e.last + 1 == a) {
        e[e.length - 1] = b;
      } else {
        s.add(a);
        e.add(b);
        c.add(code);
      }
    }
  }

  read(v4Csv, ip4, v4s, v4e, v4c);
  read(v6Csv, ip6hi, v6s, v6e, v6c);
  if (v4s.isEmpty) throw const FormatException('the IPv4 data has no ranges');

  final l = IpCountryDb._layout(v4s.length, v6s.length);
  final out = Uint8List(l.end);
  final head = ByteData.sublistView(out);
  out.setAll(0, IpCountryDb._magic);
  out[4] = 1;
  head.setUint32(8, v4s.length, Endian.little);
  head.setUint32(12, v6s.length, Endian.little);
  out.buffer.asUint32List(l.v4s, v4s.length).setAll(0, v4s);
  out.buffer.asUint32List(l.v4e, v4e.length).setAll(0, v4e);
  out.buffer.asUint16List(l.v4c, v4c.length).setAll(0, v4c);
  out.buffer.asUint64List(l.v6s, v6s.length).setAll(0, v6s);
  out.buffer.asUint64List(l.v6e, v6e.length).setAll(0, v6e);
  out.buffer.asUint16List(l.v6c, v6c.length).setAll(0, v6c);

  // Atomically: a half-written database must never be picked up.
  final tmp = File('$outPath.tmp')..writeAsBytesSync(out, flush: true);
  tmp.renameSync(outPath);
  return IpDbBuilt(v4s.length, v6s.length, out.length);
}

// ---------------------------------------------------------------------------------------------------
// Downloading it

/// The data package on npm; the files are plain CSV.
const ipDbPackage = '@ip-location-db/geo-whois-asn-country';

const ipDbFiles = ['geo-whois-asn-country-ipv4.csv', 'geo-whois-asn-country-ipv6.csv'];

/// What a file must be: its size and SHA-256 (base64, as the jsDelivr data API publishes it).
class IpDbFile {
  const IpDbFile(this.size, this.sha256);

  final int size;
  final String sha256;

  Map<String, Object> toJson() => {'size': size, 'sha256': sha256};

  factory IpDbFile.fromJson(Map<String, dynamic> j) => IpDbFile(j['size'] as int, j['sha256'] as String);
}

/// A version of the data and what each of its files must be.
class IpDbSource {
  const IpDbSource(this.version, this.files);

  final String version;
  final Map<String, IpDbFile> files;
}

/// Used when the data API cannot be reached: a version whose files were checked by hand.
const ipDbFallbackSource = IpDbSource('2.3.2026061719', {
  'geo-whois-asn-country-ipv4.csv': IpDbFile(10275222, '8IQ4JHzjFefyqv5ytIetEDIEyV2R6GbmxTBmWeVVSW8='),
  'geo-whois-asn-country-ipv6.csv': IpDbFile(12616700, '8Gqn1RsdOzjGfSIWvXuwA3RK7XtRc9htcoykZvtXcos='),
});

/// The same file on several CDNs, in the order to try them. Not every mirror can be trusted with byte
/// ranges (some answer them against a smaller, compressed copy), so each answer is checked against the
/// size the file must have, and the finished file against its SHA-256: a mirror that disagrees is skipped.
List<String> ipDbUrls(String file, String version) => [
  'https://fastly.jsdelivr.net/npm/$ipDbPackage@$version/$file',
  'https://unpkg.com/$ipDbPackage@$version/$file',
  'https://cdn.jsdelivr.net/npm/$ipDbPackage@$version/$file',
  'https://gcore.jsdelivr.net/npm/$ipDbPackage@$version/$file',
];

/// Where a download is.
class IpDbProgress {
  const IpDbProgress({
    required this.part,
    required this.parts,
    required this.file,
    required this.received,
    required this.total,
    required this.overallReceived,
    required this.overallTotal,
    this.bytesPerSecond = 0,
  });

  /// Index of the file being fetched and how many there are.
  final int part, parts;
  final String file;

  /// Bytes of this file so far and in all.
  final int received, total;

  /// Bytes of all files so far and in all.
  final int overallReceived, overallTotal;
  final double bytesPerSecond;

  /// 0-1 over everything.
  double get fraction => overallTotal == 0 ? 0 : (overallReceived / overallTotal).clamp(0.0, 1.0);

  Duration? get remaining => bytesPerSecond < 1 || overallTotal == 0
      ? null
      : Duration(seconds: ((overallTotal - overallReceived) / bytesPerSecond).ceil());
}

/// The download failed or was stopped; what was fetched is kept and the next try continues from it.
/// [message] is `stopped` (the user paused), `network` (no connection to the mirrors), `mirrors` (they
/// answered with the wrong file) or the error as text.
class IpDbDownloadException implements Exception {
  IpDbDownloadException(this.message);

  final String message;

  @override
  String toString() => message;
}

/// Lets the caller stop a download; the bytes already fetched stay on disk.
class IpDbCancel {
  bool get cancelled => _cancelled;
  bool _cancelled = false;
  void Function()? _onCancel;

  void cancel() {
    _cancelled = true;
    _onCancel?.call();
  }
}

class _Stopped implements Exception {}

/// A mirror that answered with something other than the file we want.
class _BadMirror implements Exception {
  _BadMirror(this.why);

  final String why;

  @override
  String toString() => why;
}

/// A client that never asks for compression: byte positions (and so Range resumes) must mean the same
/// thing on every request.
HttpClient _plainClient() => HttpClient()
  ..autoUncompress = false
  ..connectionTimeout = const Duration(seconds: 10);

/// Which version and files to fetch: asks the data API, and falls back to the version checked by hand.
Future<IpDbSource> _resolveFromCdn() async {
  Future<dynamic> getJson(String url) async {
    final client = _plainClient();
    try {
      final request = await client.getUrl(Uri.parse(url)).timeout(const Duration(seconds: 10));
      request.headers.set(HttpHeaders.acceptEncodingHeader, 'identity');
      final response = await request.close().timeout(const Duration(seconds: 10));
      if (response.statusCode != 200) throw HttpException('status ${response.statusCode}');
      return jsonDecode(await response.transform(utf8.decoder).join().timeout(const Duration(seconds: 10)));
    } finally {
      client.close(force: true);
    }
  }

  try {
    final tags = (await getJson('https://data.jsdelivr.com/v1/package/npm/$ipDbPackage')) as Map<String, dynamic>;
    final version = (tags['tags'] as Map<String, dynamic>)['latest'] as String;
    final listing = (await getJson('https://data.jsdelivr.com/v1/packages/npm/$ipDbPackage@$version?structure=flat'))
        as Map<String, dynamic>;
    final byName = {
      for (final f in listing['files'] as List<dynamic>)
        (f as Map<String, dynamic>)['name'] as String: IpDbFile(f['size'] as int, f['hash'] as String),
    };
    final files = {
      for (final name in ipDbFiles)
        if (byName['/$name'] != null) name: byName['/$name']!,
    };
    return files.length == ipDbFiles.length ? IpDbSource(version, files) : ipDbFallbackSource;
  } catch (_) {
    return ipDbFallbackSource;
  }
}

/// Fetches the CSV files into [dir], resumably: each file is kept as `name.part` while it downloads; a
/// request carries `Range: bytes=<size so far>-`, so a failure, a pause, a closed app or a dead network
/// loses nothing but the last few bytes. A finished file is checked against its SHA-256, renamed to its
/// real name and never fetched again.
class IpDbDownloader {
  IpDbDownloader({
    required this.dir,
    List<String> Function(String file, String version)? urls,
    Future<IpDbSource> Function()? resolve,
    this.stallTimeout = const Duration(seconds: 20),
    this.retryDelay = const Duration(seconds: 1),
    this.rounds = 2,
  }) : _urls = urls ?? ipDbUrls,
       _resolve = resolve ?? _resolveFromCdn;

  final Directory dir;
  final List<String> Function(String file, String version) _urls;
  final Future<IpDbSource> Function() _resolve;

  /// How long a connection may deliver nothing before it counts as dead.
  final Duration stallTimeout;
  final Duration retryDelay;

  /// How many times the whole list of mirrors is gone through before giving up.
  final int rounds;

  File get _manifest => File('${dir.path}${Platform.pathSeparator}download.json');

  File _file(String name, {bool part = false}) =>
      File('${dir.path}${Platform.pathSeparator}$name${part ? '.part' : ''}');

  Future<IpDbSource?> _saved() async {
    try {
      final m = jsonDecode(await _manifest.readAsString()) as Map<String, dynamic>;
      final files = (m['files'] as Map<String, dynamic>).map((k, v) => MapEntry(k, IpDbFile.fromJson(v as Map<String, dynamic>)));
      if (ipDbFiles.any((f) => files[f] == null)) return null;
      return IpDbSource(m['version'] as String, files);
    } catch (_) {
      return null;
    }
  }

  /// Bytes of each file on disk, for showing "stopped at 43%" without a network.
  Future<IpDbProgress?> onDisk() async {
    final source = await _saved();
    if (source == null) return null;
    var got = 0, all = 0, part = 0;
    for (var i = 0; i < ipDbFiles.length; i++) {
      final f = ipDbFiles[i];
      final total = source.files[f]!.size;
      all += total;
      final done = await _file(f).exists();
      final partial = _file(f, part: true);
      final have = done ? total : (await partial.exists() ? math.min(await partial.length(), total) : 0);
      got += have;
      if (done) part = i + 1;
    }
    part = part.clamp(0, ipDbFiles.length - 1);
    return IpDbProgress(
      part: part,
      parts: ipDbFiles.length,
      file: ipDbFiles[part],
      received: 0,
      total: source.files[ipDbFiles[part]]!.size,
      overallReceived: got,
      overallTotal: all,
    );
  }

  /// Fetches everything that is missing. Throws [IpDbDownloadException] when every mirror failed
  /// (call again to continue), and returns the version only when all files are complete and checked.
  Future<String> download(void Function(IpDbProgress) onProgress, IpDbCancel cancel) async {
    await dir.create(recursive: true);
    // A download that was started goes on with the same version and files, whatever the API says now.
    var source = await _saved();
    if (source == null) {
      source = await _resolve();
      await _manifest.writeAsString(
        jsonEncode({'version': source.version, 'files': source.files.map((k, v) => MapEntry(k, v.toJson()))}),
      );
    }
    final overallTotal = ipDbFiles.fold<int>(0, (a, f) => a + source!.files[f]!.size);

    final clock = Stopwatch()..start();
    final window = <(int, int)>[]; // (ms, overall bytes) over the last few seconds, for the speed
    double speed(int overall) {
      final now = clock.elapsedMilliseconds;
      window.add((now, overall));
      while (window.length > 1 && now - window.first.$1 > 3000) {
        window.removeAt(0);
      }
      final dt = now - window.first.$1;
      return dt < 300 ? 0 : (overall - window.first.$2) * 1000 / dt;
    }

    final bad = <String>{}; // mirrors (by origin) that disagreed with the expected file, for this run
    var before = 0;
    for (var i = 0; i < ipDbFiles.length; i++) {
      final f = ipDbFiles[i];
      final expected = source.files[f]!;
      if (await _file(f).exists() && await _file(f).length() == expected.size) {
        before += expected.size;
        continue;
      }
      var last = DateTime.fromMillisecondsSinceEpoch(0);
      await _fetchFile(f, source.version, expected, cancel, bad, (got) {
        final now = DateTime.now();
        if (now.difference(last).inMilliseconds < 100 && got != expected.size) return;
        last = now;
        onProgress(IpDbProgress(
          part: i,
          parts: ipDbFiles.length,
          file: f,
          received: got,
          total: expected.size,
          overallReceived: before + got,
          overallTotal: overallTotal,
          bytesPerSecond: speed(before + got),
        ));
      });
      before += expected.size;
    }
    return source.version;
  }

  Future<void> _fetchFile(
    String file,
    String version,
    IpDbFile expected,
    IpDbCancel cancel,
    Set<String> bad,
    void Function(int received) onBytes,
  ) async {
    final urls = _urls(file, version);
    Object? last;
    var attempts = 0;
    for (var round = 0; round < rounds; round++) {
      for (final url in urls) {
        final mirror = Uri.parse(url).origin; // a mirror that was wrong about one file is skipped for the others
        if (bad.contains(mirror)) continue;
        if (cancel.cancelled) throw IpDbDownloadException('stopped');
        attempts++;
        try {
          await _stream(url, file, expected, cancel, onBytes);
          return;
        } on _Stopped {
          throw IpDbDownloadException('stopped');
        } on _BadMirror catch (e) {
          bad.add(mirror); // it will not answer correctly a second time
          last = e;
        } catch (e) {
          last = e; // the next mirror continues from the byte this one stopped at
          await Future<void>.delayed(retryDelay);
        }
      }
    }
    // Reasons the screen can translate; anything else is shown as it is.
    final reason = attempts == 0 || last is _BadMirror
        ? 'mirrors'
        : (last is SocketException || last is TimeoutException || last is HandshakeException ? 'network' : '$last');
    throw IpDbDownloadException(reason);
  }

  Future<void> _stream(String url, String file, IpDbFile expected, IpDbCancel cancel, void Function(int) onBytes) async {
    final part = _file(file, part: true);
    var offset = await part.exists() ? await part.length() : 0;
    if (offset > expected.size) {
      await part.delete(); // not this file
      offset = 0;
    }
    final resumed = offset > 0;
    if (offset < expected.size) {
      final client = _plainClient();
      cancel._onCancel = () => client.close(force: true);
      RandomAccessFile? out;
      try {
        final request = await client.getUrl(Uri.parse(url)).timeout(stallTimeout);
        request.headers.set(HttpHeaders.acceptEncodingHeader, 'identity');
        if (offset > 0) request.headers.set(HttpHeaders.rangeHeader, 'bytes=$offset-');
        final response = await request.close().timeout(stallTimeout);
        if (response.statusCode == 206) {
          // The mirror must be talking about the same file: its total is the size we expect.
          final total = int.tryParse((response.headers.value(HttpHeaders.contentRangeHeader) ?? '').split('/').last);
          if (total != expected.size) {
            await response.drain<void>();
            throw _BadMirror('this mirror reports $total bytes for a file of ${expected.size}');
          }
        } else if (response.statusCode == 200) {
          if (response.contentLength > 0 && response.contentLength != expected.size) {
            await response.drain<void>();
            throw _BadMirror('this mirror reports ${response.contentLength} bytes for a file of ${expected.size}');
          }
          offset = 0; // no range support (or none asked for): the whole file comes again
        } else {
          await response.drain<void>();
          throw HttpException('status ${response.statusCode}');
        }
        out = await part.open(mode: offset == 0 ? FileMode.write : FileMode.append);
        var got = offset;
        onBytes(got);
        await for (final chunk in response.timeout(stallTimeout)) {
          if (cancel.cancelled) throw _Stopped();
          await out.writeFrom(chunk);
          got += chunk.length;
          if (got > expected.size) throw _BadMirror('more data than the ${expected.size} bytes expected');
          onBytes(got);
        }
        await out.close();
        out = null;
        if (got != expected.size) throw HttpException('incomplete: $got of ${expected.size} bytes');
      } on _Stopped {
        rethrow;
      } catch (e) {
        if (cancel.cancelled) throw _Stopped();
        rethrow;
      } finally {
        cancel._onCancel = null;
        await out?.close();
        client.close(force: true);
      }
    }
    // All the bytes are here: they must be the right ones.
    final digest = base64.encode((await sha256.bind(part.openRead()).first).bytes);
    if (digest != expected.sha256) {
      await part.delete();
      // After a resume the old bytes are the likelier culprit than the mirror: start the file again.
      if (resumed) throw HttpException('checksum differs after resuming; starting this file again');
      throw _BadMirror('the file from this mirror is damaged (checksum differs)');
    }
    await part.rename(_file(file).path);
    onBytes(expected.size);
  }

  /// Compiles the downloaded files into [target], removes them and returns what was built.
  Future<IpDbBuilt> build(String target) async {
    final built = await compileIpDb(_file(ipDbFiles[0]).path, _file(ipDbFiles[1]).path, target);
    await clean();
    return built;
  }

  /// Forgets a download (the files, the manifest).
  Future<void> clean() async {
    for (final f in ipDbFiles) {
      for (final file in [_file(f), _file(f, part: true)]) {
        if (await file.exists()) await file.delete();
      }
    }
    if (await _manifest.exists()) await _manifest.delete();
  }
}
