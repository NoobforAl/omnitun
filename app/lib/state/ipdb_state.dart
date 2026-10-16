import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:path_provider/path_provider.dart';

import '../engine/engine.dart';
import 'ipdb.dart';
import 'providers.dart';

enum IpDbPhase {
  /// Not downloaded.
  none,

  /// Fetching the files.
  downloading,

  /// Building the database from the files.
  preparing,

  /// Stopped by the user; the bytes are kept.
  paused,

  /// Stopped by an error; the bytes are kept.
  failed,

  /// Installed.
  ready,
}

/// What an installed database is.
class IpDbInfo {
  const IpDbInfo({required this.version, required this.builtAt, required this.v4, required this.v6, required this.bytes});

  final String version;
  final DateTime builtAt;
  final int v4, v6, bytes;

  int get ranges => v4 + v6;

  Map<String, Object> toJson() => {
    'version': version,
    'built_at': builtAt.millisecondsSinceEpoch,
    'v4': v4,
    'v6': v6,
    'bytes': bytes,
  };

  factory IpDbInfo.fromJson(Map<String, dynamic> j) => IpDbInfo(
    version: j['version'] as String,
    builtAt: DateTime.fromMillisecondsSinceEpoch(j['built_at'] as int),
    v4: j['v4'] as int,
    v6: j['v6'] as int,
    bytes: j['bytes'] as int,
  );
}

class IpDbState {
  const IpDbState({this.phase = IpDbPhase.none, this.progress, this.error, this.info, this.db, this.enabled = true});

  final IpDbPhase phase;

  /// While downloading, or where a stopped download stopped.
  final IpDbProgress? progress;
  final String? error;

  /// The installed database (kept in use while an update downloads).
  final IpDbInfo? info;
  final IpCountryDb? db;

  /// The user's switch: an installed database is only used while this is on.
  final bool enabled;

  bool get installed => db != null;

  IpDbState copyWith({
    IpDbPhase? phase,
    IpDbProgress? Function()? progress,
    String? Function()? error,
    IpDbInfo? info,
    IpCountryDb? db,
    bool clearDb = false,
    bool? enabled,
  }) => IpDbState(
    phase: phase ?? this.phase,
    progress: progress != null ? progress() : this.progress,
    error: error != null ? error() : this.error,
    info: clearDb ? null : (info ?? this.info),
    db: clearDb ? null : (db ?? this.db),
    enabled: enabled ?? this.enabled,
  );
}

/// Where the database lives (overridden in tests).
final ipDbDirProvider = Provider<Future<Directory> Function()>((ref) {
  return () async {
    final base = await getApplicationSupportDirectory();
    return Directory('${base.path}${Platform.pathSeparator}ipdb');
  };
});

/// Makes the downloader for a folder (replaced in tests, which download from a local server).
final ipDbDownloaderProvider = Provider<IpDbDownloader Function(Directory)>((ref) => (dir) => IpDbDownloader(dir: dir));

/// Runs the optional offline IP → country database: download (resumable), build, load, delete.
class IpDbController extends Notifier<IpDbState> {
  static const _enabledKey = 'ipdb_enabled';

  IpDbCancel? _cancel;
  Directory? _base;

  @override
  IpDbState build() {
    ref.onDispose(() => _cancel?.cancel());
    Future.microtask(_init);
    return IpDbState(enabled: ref.read(storeProvider).read<bool>(_enabledKey) ?? true);
  }

  Future<Directory> _dir() async => _base ??= await ref.read(ipDbDirProvider)();

  File _bin(Directory d) => File('${d.path}${Platform.pathSeparator}ipcountry.bin');
  File _meta(Directory d) => File('${d.path}${Platform.pathSeparator}meta.json');
  Directory _staging(Directory d) => Directory('${d.path}${Platform.pathSeparator}staging');

  Future<void> _init() async {
    try {
      final d = await _dir();
      try {
        final info = IpDbInfo.fromJson(jsonDecode(await _meta(d).readAsString()) as Map<String, dynamic>);
        final db = IpCountryDb.fromBytes(await _bin(d).readAsBytes());
        if (!ref.mounted) return;
        state = state.copyWith(phase: IpDbPhase.ready, info: info, db: db);
      } catch (_) {
        // Nothing installed (or a file that cannot be read: it is replaced by the next download).
      }
      // A download that was left half-way, by an error or a closed app, is offered to continue.
      final left = await ref.read(ipDbDownloaderProvider)(_staging(d)).onDisk();
      if (!ref.mounted) return;
      if (left != null && left.overallReceived > 0 && state.phase != IpDbPhase.downloading) {
        state = state.copyWith(phase: IpDbPhase.paused, progress: () => left);
      }
    } catch (_) {
      // No place to keep a database (e.g. no platform folders): the feature is simply not available.
    }
  }

  /// Starts a download, or continues one that stopped (the bytes already fetched are kept).
  Future<void> start() async {
    if (state.phase == IpDbPhase.downloading || state.phase == IpDbPhase.preparing) return;
    final d = await _dir();
    final downloader = ref.read(ipDbDownloaderProvider)(_staging(d));
    final cancel = _cancel = IpDbCancel();
    state = state.copyWith(phase: IpDbPhase.downloading, error: () => null);
    try {
      final version = await downloader.download((p) {
        if (ref.mounted && !cancel.cancelled) state = state.copyWith(progress: () => p);
      }, cancel);
      if (!ref.mounted) return;
      state = state.copyWith(phase: IpDbPhase.preparing);
      final built = await downloader.build(_bin(d).path);
      final info = IpDbInfo(version: version, builtAt: DateTime.now(), v4: built.v4Ranges, v6: built.v6Ranges, bytes: built.bytes);
      await _meta(d).writeAsString(jsonEncode(info.toJson()));
      final db = IpCountryDb.fromBytes(await _bin(d).readAsBytes());
      if (!ref.mounted) return;
      state = state.copyWith(phase: IpDbPhase.ready, progress: () => null, error: () => null, info: info, db: db);
    } on IpDbDownloadException catch (e) {
      if (!ref.mounted) return;
      final left = await downloader.onDisk();
      state = state.copyWith(
        phase: e.message == 'stopped' ? IpDbPhase.paused : IpDbPhase.failed,
        progress: () => left ?? state.progress,
        error: () => e.message == 'stopped' ? null : e.message,
      );
    } catch (e) {
      if (!ref.mounted) return;
      // The files could not be built into a database: start again from a clean state.
      await downloader.clean();
      state = state.copyWith(phase: IpDbPhase.failed, progress: () => null, error: () => '$e');
    } finally {
      if (_cancel == cancel) _cancel = null;
    }
  }

  /// Stops the download; what was fetched is kept.
  void pause() => _cancel?.cancel();

  /// Removes the database and anything half-downloaded.
  Future<void> delete() async {
    _cancel?.cancel();
    final d = await _dir();
    try {
      if (await d.exists()) await d.delete(recursive: true);
    } catch (_) {}
    if (!ref.mounted) return;
    state = IpDbState(enabled: state.enabled);
  }

  void setEnabled(bool on) {
    ref.read(storeProvider).write(_enabledKey, on);
    state = state.copyWith(enabled: on);
  }
}

final ipDbProvider = NotifierProvider<IpDbController, IpDbState>(IpDbController.new);

/// The database while it is installed and switched on; null otherwise. Changes only when the
/// database does, not with every step of a download.
final ipDbActiveProvider = Provider<IpCountryDb?>((ref) {
  return ref.watch(ipDbProvider.select((s) => s.enabled ? s.db : null));
});

/// Countries of the servers' addresses, by host name, found with the database: an address is looked up
/// as it is, a name is resolved first. Done in the background and remembered; forgotten when the
/// database changes.
class DbCountries extends Notifier<Map<String, String>> {
  static const _key = 'db_countries';
  final _tried = <String>{};
  IpCountryDb? _for;

  @override
  Map<String, String> build() {
    final saved = ref.read(storeProvider).read<Map<String, dynamic>>(_key) ?? const {};
    ref.listen(ipDbActiveProvider, (prev, next) {
      if (identical(prev, next)) return;
      _tried.clear();
      _for = null;
      if (state.isNotEmpty) {
        state = {};
        ref.read(storeProvider).write(_key, <String, String>{});
      }
      _resolveAll();
    });
    ref.listen(profilesProvider, (prev, next) => _resolveAll());
    Future.microtask(_resolveAll);
    return {for (final e in saved.entries) e.key: e.value as String};
  }

  Future<void> _resolveAll() async {
    if (!ref.mounted) return;
    final db = ref.read(ipDbActiveProvider);
    if (db == null) return;
    _for = db;
    final hosts = {
      for (final p in ref.read(profilesProvider))
        if (p.protocol != 'tor' && p.server.isNotEmpty) p.server,
    }.where((h) => !state.containsKey(h) && _tried.add(h)).toList();
    // A few at a time: each name is a DNS question.
    for (var i = 0; i < hosts.length; i += 4) {
      await Future.wait(hosts.skip(i).take(4).map((h) => _resolve(db, h)));
      if (!ref.mounted || _for != db) return;
    }
  }

  Future<void> _resolve(IpCountryDb db, String host) async {
    try {
      final addresses = InternetAddress.tryParse(host) != null
          ? [host]
          : [for (final a in await InternetAddress.lookup(host).timeout(const Duration(seconds: 5))) a.address];
      final countries = {for (final a in addresses) ?db.lookup(a)};
      if (countries.length != 1 || !ref.mounted || _for != db) return; // not known, or it differs by address
      state = {...state, host: countries.single};
      ref.read(storeProvider).write(_key, state);
    } catch (_) {
      // No answer: the name guess stays.
    }
  }
}

final dbCountriesProvider = NotifierProvider<DbCountries, Map<String, String>>(DbCountries.new);

/// Fills in the country of an address that a lookup service answered without one, and prefers the
/// local database's answer when it is installed (nothing more is asked of a website for the country).
ExitInfo withDbCountry(ExitInfo info, IpCountryDb? db) {
  final c = db?.lookup(info.ip);
  return c == null ? info : ExitInfo(info.ip, c);
}
