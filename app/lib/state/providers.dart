import 'dart:async';
import 'dart:io';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../engine/engine.dart';
import '../engine/core_engine.dart';

final engineProvider = Provider<VpnEngine>((ref) {
  final engine = CoreEngine();
  ref.onDispose(engine.dispose);
  return engine;
});

/// Imported profiles. Persistence (drift/SQLite) comes in a later phase.
class ProfilesController extends Notifier<List<ProfileInfo>> {
  @override
  List<ProfileInfo> build() => const [];

  /// Adds profiles, skipping ones whose ID is already present.
  int addAll(Iterable<ProfileInfo> profiles) {
    final known = {for (final p in state) p.id};
    final fresh = profiles.where((p) => known.add(p.id)).toList();
    state = [...state, ...fresh];
    return fresh.length;
  }

  void remove(String id) => state = state.where((p) => p.id != id).toList();

  void setLatency(String id, int ms) =>
      state = [for (final p in state) p.id == id ? p.copyWith(latencyMs: ms) : p];
}

final profilesProvider =
    NotifierProvider<ProfilesController, List<ProfileInfo>>(ProfilesController.new);

class SelectedProfileController extends Notifier<String?> {
  @override
  String? build() => null;

  void select(String? id) => state = id;
}

final selectedProfileIdProvider =
    NotifierProvider<SelectedProfileController, String?>(SelectedProfileController.new);

/// The profile that Connect will use: the selection, or the first profile.
final selectedProfileProvider = Provider<ProfileInfo?>((ref) {
  final profiles = ref.watch(profilesProvider);
  final id = ref.watch(selectedProfileIdProvider);
  if (profiles.isEmpty) return null;
  return profiles.firstWhere((p) => p.id == id, orElse: () => profiles.first);
});

class ConnectionController extends Notifier<EngineState> {
  @override
  EngineState build() {
    final engine = ref.watch(engineProvider);
    final sub = engine.states.listen((s) => state = s);
    ref.onDispose(sub.cancel);
    return const EngineState();
  }

  bool get isActive => switch (state.status) {
        ConnStatus.connected || ConnStatus.connecting => true,
        _ => false,
      };

  Future<void> toggle() async {
    final engine = ref.read(engineProvider);
    if (isActive) {
      await engine.disconnect();
      return;
    }
    final profile = ref.read(selectedProfileProvider);
    if (profile == null) return;
    final others = ref.read(profilesProvider).where((p) => p.id != profile.id).toList();
    final settings = ref.read(settingsProvider);
    await engine.connect(
      profile,
      fallbacks: others,
      options: ConnectOptions(
        tunMode: settings.tunMode,
        systemProxy: settings.systemProxy,
        dns: settings.remoteDns,
      ),
    );
  }
}

final connectionProvider =
    NotifierProvider<ConnectionController, EngineState>(ConnectionController.new);

class LogsController extends Notifier<List<String>> {
  static const _max = 500;

  @override
  List<String> build() {
    final sub = ref.watch(engineProvider).logs.listen((line) {
      final next = [...state, line];
      state = next.length > _max ? next.sublist(next.length - _max) : next;
    });
    ref.onDispose(sub.cancel);
    return const [];
  }

  void clear() => state = const [];
}

final logsProvider = NotifierProvider<LogsController, List<String>>(LogsController.new);

class SettingsState {
  const SettingsState({
    this.tunMode = false,
    this.systemProxy = true,
    this.stopOnClose = true,
    this.autoFailover = true,
    this.killSwitch = false,
    this.remoteDns = defaultRemoteDns,
  });

  static const defaultRemoteDns = 'https://1.1.1.1/dns-query';

  /// Capture all traffic. Off = proxy mode (works without admin rights on desktop).
  final bool tunMode;

  /// In proxy mode, set the OS proxy to the local port while connected.
  final bool systemProxy;
  final bool stopOnClose;
  final bool autoFailover;
  final bool killSwitch;

  /// Resolver used through the tunnel: https://, tls://, tcp://, udp:// or an IP.
  final String remoteDns;

  SettingsState copyWith({
    bool? tunMode,
    bool? systemProxy,
    bool? stopOnClose,
    bool? autoFailover,
    bool? killSwitch,
    String? remoteDns,
  }) =>
      SettingsState(
        tunMode: tunMode ?? this.tunMode,
        systemProxy: systemProxy ?? this.systemProxy,
        stopOnClose: stopOnClose ?? this.stopOnClose,
        autoFailover: autoFailover ?? this.autoFailover,
        killSwitch: killSwitch ?? this.killSwitch,
        remoteDns: remoteDns ?? this.remoteDns,
      );
}

class SettingsController extends Notifier<SettingsState> {
  @override
  // Android captures all traffic by default; desktop starts in proxy mode (TUN needs admin rights there).
  SettingsState build() => SettingsState(tunMode: Platform.isAndroid);

  void update(SettingsState Function(SettingsState) f) => state = f(state);
}

final settingsProvider = NotifierProvider<SettingsController, SettingsState>(SettingsController.new);
