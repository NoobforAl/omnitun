import 'dart:async';
import 'dart:io';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../engine/core_engine.dart';
import '../engine/engine.dart';
import 'store.dart';

/// Overridden in main() with the store loaded from disk.
final storeProvider = Provider<AppStore>((ref) => AppStore.memory());

final engineProvider = Provider<VpnEngine>((ref) {
  final engine = CoreEngine();
  ref.onDispose(engine.dispose);
  return engine;
});

/// Imported profiles, persisted as the core's raw profile JSON.
class ProfilesController extends Notifier<List<ProfileInfo>> {
  static const _key = 'profiles';

  @override
  List<ProfileInfo> build() {
    final saved = ref.read(storeProvider).read<List<dynamic>>(_key) ?? const [];
    final out = <ProfileInfo>[];
    for (final raw in saved) {
      try {
        out.add(ProfileInfo.fromJson(Map<String, dynamic>.from(raw as Map)));
      } catch (_) {
        // Skip entries written by an incompatible version.
      }
    }
    return out;
  }

  void _save(List<ProfileInfo> next) {
    state = next;
    ref.read(storeProvider).write(_key, [for (final p in next) p.raw]);
  }

  /// Adds profiles, skipping ones whose ID is already present.
  int addAll(Iterable<ProfileInfo> profiles) {
    final known = {for (final p in state) p.id};
    final fresh = profiles.where((p) => known.add(p.id)).toList();
    if (fresh.isNotEmpty) _save([...state, ...fresh]);
    return fresh.length;
  }

  void remove(String id) => _save(state.where((p) => p.id != id).toList());

  /// Replaces the profiles of [group] (a subscription refresh), keeping the
  /// group's position in the list. IDs are stable, so the selection survives.
  void replaceGroup(String group, List<ProfileInfo> profiles) {
    final at = state.indexWhere((p) => p.group == group);
    final rest = state.where((p) => p.group != group).toList();
    final fresh = <String>{};
    final incoming = [
      for (final p in profiles)
        if (fresh.add(p.id)) p.inGroup(group),
    ];
    // A profile also imported manually stays in its manual group.
    final manual = {for (final p in rest) p.id};
    incoming.removeWhere((p) => manual.contains(p.id));
    final insertAt = at < 0 ? rest.length : rest.where((p) => state.indexOf(p) < at).length;
    _save([...rest.take(insertAt), ...incoming, ...rest.skip(insertAt)]);
  }

  void removeGroup(String group) => _save(state.where((p) => p.group != group).toList());

  /// Moves a profile within its group; [from]/[to] are positions inside the
  /// group ([to] already adjusted for the removed item).
  void reorderInGroup(String group, int from, int to) {
    final members = state.where((p) => p.group == group).toList();
    final moved = members.removeAt(from);
    members.insert(to, moved);
    var i = 0;
    _save([for (final p in state) p.group == group ? members[i++] : p]);
  }

  /// Moves a profile to [to] (index after removal); the order is the failover priority.
  void reorder(int from, int to) {
    final next = [...state];
    final p = next.removeAt(from);
    next.insert(to, p);
    _save(next);
  }

  // Latency is transient and not persisted.
  void setLatency(String id, int ms) =>
      state = [for (final p in state) p.id == id ? p.copyWith(latencyMs: ms) : p];
}

final profilesProvider =
    NotifierProvider<ProfilesController, List<ProfileInfo>>(ProfilesController.new);

class SelectedProfileController extends Notifier<String?> {
  static const _key = 'selected';

  @override
  String? build() => ref.read(storeProvider).read<String>(_key);

  void select(String? id) {
    state = id;
    ref.read(storeProvider).write(_key, id);
  }
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
    final sub = engine.states.listen((s) {
      state = s;
      // Follow automatic switches so the UI shows the profile actually in use.
      final active = s.activeProfileId;
      if (s.failover != null && active != null && active != ref.read(selectedProfileIdProvider)) {
        ref.read(selectedProfileIdProvider.notifier).select(active);
      }
    });
    ref.onDispose(sub.cancel);
    return const EngineState();
  }

  bool get isActive => switch (state.status) {
        ConnStatus.connected || ConnStatus.connecting || ConnStatus.noInternet => true,
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
    // Fallbacks: the rest of the selected profile's group in list order,
    // then all other profiles.
    final all = ref.read(profilesProvider);
    final group = all.where((p) => p.group == profile.group).toList();
    final i = group.indexWhere((p) => p.id == profile.id);
    final others = [
      ...group.skip(i + 1),
      ...group.take(i),
      ...all.where((p) => p.group != profile.group),
    ];
    final settings = ref.read(settingsProvider);
    await engine.connect(
      profile,
      fallbacks: others,
      options: ConnectOptions(
        tunMode: settings.tunMode,
        systemProxy: settings.systemProxy,
        dns: settings.remoteDns,
        autoFailover: settings.autoFailover,
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

  Map<String, dynamic> toJson() => {
        'tun_mode': tunMode,
        'system_proxy': systemProxy,
        'stop_on_close': stopOnClose,
        'auto_failover': autoFailover,
        'kill_switch': killSwitch,
        'remote_dns': remoteDns,
      };

  /// Missing keys keep the values from [defaults].
  factory SettingsState.fromJson(Map<String, dynamic> j, SettingsState defaults) => SettingsState(
        tunMode: j['tun_mode'] as bool? ?? defaults.tunMode,
        systemProxy: j['system_proxy'] as bool? ?? defaults.systemProxy,
        stopOnClose: j['stop_on_close'] as bool? ?? defaults.stopOnClose,
        autoFailover: j['auto_failover'] as bool? ?? defaults.autoFailover,
        killSwitch: j['kill_switch'] as bool? ?? defaults.killSwitch,
        remoteDns: j['remote_dns'] as String? ?? defaults.remoteDns,
      );
}

class SettingsController extends Notifier<SettingsState> {
  static const _key = 'settings';

  @override
  SettingsState build() {
    // Android captures all traffic by default; desktop starts in proxy mode (TUN needs admin rights there).
    final defaults = SettingsState(tunMode: Platform.isAndroid);
    final saved = ref.read(storeProvider).read<Map<String, dynamic>>(_key);
    final s = saved == null ? defaults : SettingsState.fromJson(saved, defaults);
    // The native side needs this to act when the OS closes the app.
    final engine = ref.read(engineProvider);
    Future.microtask(() => engine.setStopOnClose(s.stopOnClose));
    return s;
  }

  void update(SettingsState Function(SettingsState) f) {
    final before = state;
    state = f(state);
    ref.read(storeProvider).write(_key, state.toJson());
    if (state.stopOnClose != before.stopOnClose) {
      ref.read(engineProvider).setStopOnClose(state.stopOnClose);
    }
  }
}

final settingsProvider = NotifierProvider<SettingsController, SettingsState>(SettingsController.new);
