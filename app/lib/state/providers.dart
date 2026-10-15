import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../engine/engine.dart';
import '../engine/mock_engine.dart';

final engineProvider = Provider<VpnEngine>((ref) {
  final engine = MockEngine();
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
    await engine.connect(profile, fallbacks: others);
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
    this.tunMode = true,
    this.stopOnClose = true,
    this.autoFailover = true,
    this.killSwitch = false,
  });

  final bool tunMode;
  final bool stopOnClose;
  final bool autoFailover;
  final bool killSwitch;

  SettingsState copyWith({bool? tunMode, bool? stopOnClose, bool? autoFailover, bool? killSwitch}) =>
      SettingsState(
        tunMode: tunMode ?? this.tunMode,
        stopOnClose: stopOnClose ?? this.stopOnClose,
        autoFailover: autoFailover ?? this.autoFailover,
        killSwitch: killSwitch ?? this.killSwitch,
      );
}

class SettingsController extends Notifier<SettingsState> {
  @override
  SettingsState build() => const SettingsState();

  void update(SettingsState Function(SettingsState) f) => state = f(state);
}

final settingsProvider = NotifierProvider<SettingsController, SettingsState>(SettingsController.new);
