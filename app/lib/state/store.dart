import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:path_provider/path_provider.dart';

/// Persists app data as one JSON document in the app's private support
/// directory. Writes are debounced and atomic (write temp file, then rename).
class AppStore {
  AppStore._(this._file, this.data);

  /// In-memory store for tests.
  AppStore.memory([Map<String, dynamic>? data])
      : _file = null,
        data = data ?? {};

  final File? _file;
  final Map<String, dynamic> data;
  Timer? _debounce;

  static Future<AppStore> open() async {
    final dir = await getApplicationSupportDirectory();
    final file = File('${dir.path}${Platform.pathSeparator}omnitun.json');
    var data = <String, dynamic>{};
    if (await file.exists()) {
      try {
        data = jsonDecode(await file.readAsString()) as Map<String, dynamic>;
      } catch (_) {
        // Keep a copy of an unreadable file instead of silently losing it.
        await file.rename('${file.path}.corrupt').catchError((_) => file);
      }
    }
    return AppStore._(file, data);
  }

  T? read<T>(String key) => data[key] as T?;

  void write(String key, Object? value) {
    data[key] = value;
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 300), flush);
  }

  Future<void> flush() async {
    _debounce?.cancel();
    final file = _file;
    if (file == null) return;
    final tmp = File('${file.path}.tmp');
    await tmp.writeAsString(jsonEncode(data), flush: true);
    await tmp.rename(file.path);
  }
}
