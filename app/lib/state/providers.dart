import 'dart:async';
import 'dart:io';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../engine/core_engine.dart';
import '../engine/engine.dart';
import 'countries.dart';
import 'location.dart';
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

  /// Puts an edited profile where [oldId] was (same group and place in the list). If
  /// the edit gave it a new ID, a selection of the old one follows.
  void replace(String oldId, ProfileInfo edited) {
    final at = state.indexWhere((p) => p.id == oldId);
    if (at < 0) {
      addAll([edited]);
      return;
    }
    final keep = state[at];
    final next = [...state];
    next[at] = edited.group == keep.group ? edited : edited.inGroup(keep.group);
    // Another profile with the ID the edit now has would be a duplicate.
    _save([for (var i = 0; i < next.length; i++) if (i == at || next[i].id != edited.id) next[i]]);
    if (edited.id != oldId && ref.read(selectedProfileIdProvider) == oldId) {
      ref.read(selectedProfileIdProvider.notifier).select(edited.id);
    }
  }

  /// Replaces the profiles of [group] (a subscription refresh), keeping the
  /// group's position in the list. IDs are stable, so the selection survives.
  void replaceGroup(String group, List<ProfileInfo> profiles) {
    final at = state.indexWhere((p) => p.group == group);
    final rest = state.where((p) => p.group != group).toList();
    final fresh = <String>{};
    // Credentials entered in the app aren't part of the subscription; keep them.
    final old = {for (final p in state) p.id: p};
    final incoming = [
      for (final p in profiles)
        if (fresh.add(p.id))
          (p.needsCredentials && !p.hasCredentials && (old[p.id]?.hasCredentials ?? false))
              ? p.inGroup(group).withCredentials(old[p.id]!.raw['username'] as String, old[p.id]!.raw['password'] as String)
              : p.inGroup(group),
    ];
    // A profile also imported manually stays in its manual group.
    final manual = {for (final p in rest) p.id};
    incoming.removeWhere((p) => manual.contains(p.id));
    final insertAt = at < 0 ? rest.length : rest.where((p) => state.indexOf(p) < at).length;
    _save([...rest.take(insertAt), ...incoming, ...rest.skip(insertAt)]);
  }

  /// Saves the username and password of a profile that needs them (OpenVPN).
  void setCredentials(String id, String username, String password) =>
      _save([for (final p in state) p.id == id ? p.withCredentials(username, password) : p]);

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

/// The profile editors of the protocols the core can build by hand.
final formsProvider = FutureProvider<List<ProtocolFormSpec>>((ref) => ref.read(engineProvider).forms());

final profilesProvider =
    NotifierProvider<ProfilesController, List<ProfileInfo>>(ProfilesController.new);

/// Where the user wants to connect: null = Auto, "country:DE" = a country,
/// otherwise one profile's ID (see location.dart).
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

/// Countries the user set by hand, by profile ID (they win over the guess).
class CountryOverrides extends Notifier<Map<String, String>> {
  static const _key = 'countries';

  @override
  Map<String, String> build() {
    final saved = ref.read(storeProvider).read<Map<String, dynamic>>(_key) ?? const {};
    return {for (final e in saved.entries) e.key: e.value as String};
  }

  /// Sets the country of a profile; null goes back to the guess.
  void set(String id, String? code) {
    final next = {...state};
    code == null ? next.remove(id) : next[id] = code;
    state = next;
    ref.read(storeProvider).write(_key, next);
  }
}

final countryOverridesProvider = NotifierProvider<CountryOverrides, Map<String, String>>(CountryOverrides.new);

/// The country of a profile: the user's choice, else a guess from its name and address.
final profileCountryProvider = Provider<String? Function(ProfileInfo)>((ref) {
  final overrides = ref.watch(countryOverridesProvider);
  final guesses = <String, String?>{};
  return (p) => overrides[p.id] ?? guesses.putIfAbsent(p.id, () => guessCountry(p));
});

/// The profile to start with and the ones to fall back to, for the current choice.
final connectionPlanProvider = Provider<ConnectionPlan?>((ref) {
  return planConnection(
    ref.watch(profilesProvider),
    ref.watch(selectedProfileIdProvider),
    ref.watch(profileCountryProvider),
  );
});

/// The profile that Connect will start with.
final selectedProfileProvider = Provider<ProfileInfo?>((ref) => ref.watch(connectionPlanProvider)?.primary);

class ConnectionController extends Notifier<EngineState> {
  @override
  EngineState build() {
    final engine = ref.watch(engineProvider);
    final sub = engine.states.listen((s) {
      state = s;
      // Follow automatic switches so the UI shows the profile actually in use.
      final active = s.activeProfileId;
      // Auto and country choices stay as they are; they pick the profile themselves.
      final choice = ref.read(selectedProfileIdProvider);
      final specific = choice != null && countryOfChoice(choice) == null;
      if (s.failover != null && active != null && specific && active != choice) {
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
    if (isActive) {
      await ref.read(engineProvider).disconnect();
      return;
    }
    await _connect();
  }

  /// Switches to the location chosen now (disconnect, then connect).
  Future<void> reconnect() async {
    await ref.read(engineProvider).disconnect();
    await _connect();
  }

  Future<void> _connect() async {
    final engine = ref.read(engineProvider);
    final plan = ref.read(connectionPlanProvider);
    if (plan == null) return;
    final settings = ref.read(settingsProvider);
    await engine.connect(
      plan.primary,
      fallbacks: plan.fallbacks,
      options: ConnectOptions(
        tunMode: settings.tunMode,
        systemProxy: settings.systemProxy,
        dns: settings.remoteDns,
        includePackages: settings.splitMode == 'include' ? settings.splitApps : const [],
        excludePackages: settings.splitMode == 'exclude' ? settings.splitApps : const [],
        autoFailover: settings.autoFailover,
        bypassLan: settings.bypassLan,
        bypass: ref.read(bypassListsProvider.notifier).enabledText,
        bypassOnly: settings.bypassOnly,
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

/// A named list of destinations that go around the VPN (see core/singbox/bypasslist.go).
class BypassList {
  const BypassList({required this.id, required this.name, required this.text, this.enabled = true, this.entries = 0});

  final String id;
  final String name;

  /// One entry per line: a domain, `full:`, `keyword:`, `regexp:`, an address or a range.
  final String text;
  final bool enabled;

  /// How many entries the core found in [text] when it was added.
  final int entries;

  BypassList copyWith({String? name, String? text, bool? enabled, int? entries}) => BypassList(
    id: id,
    name: name ?? this.name,
    text: text ?? this.text,
    enabled: enabled ?? this.enabled,
    entries: entries ?? this.entries,
  );

  Map<String, dynamic> toJson() => {'id': id, 'name': name, 'text': text, 'enabled': enabled, 'entries': entries};

  factory BypassList.fromJson(Map<String, dynamic> j) => BypassList(
    id: j['id'] as String,
    name: j['name'] as String? ?? 'List',
    text: j['text'] as String? ?? '',
    enabled: j['enabled'] as bool? ?? true,
    entries: (j['entries'] as num?)?.toInt() ?? 0,
  );
}

/// The user's bypass lists, persisted.
class BypassListsController extends Notifier<List<BypassList>> {
  static const _key = 'bypass_lists';

  @override
  List<BypassList> build() {
    final saved = ref.read(storeProvider).read<List<dynamic>>(_key) ?? const [];
    return [
      for (final s in saved)
        if (s is Map) BypassList.fromJson(Map<String, dynamic>.from(s)),
    ];
  }

  void _save(List<BypassList> next) {
    state = next;
    ref.read(storeProvider).write(_key, [for (final l in next) l.toJson()]);
  }

  String add({required String name, required String text, required int entries}) {
    final id = 'bl-${DateTime.now().microsecondsSinceEpoch.toRadixString(36)}';
    _save([...state, BypassList(id: id, name: name, text: text, entries: entries)]);
    return id;
  }

  void update(BypassList list) => _save([for (final l in state) l.id == list.id ? list : l]);

  void remove(String id) => _save(state.where((l) => l.id != id).toList());

  /// The text of every enabled list, as sent to the core.
  String get enabledText => [for (final l in state) if (l.enabled) l.text.trim()].where((t) => t.isNotEmpty).join('\n');
}

final bypassListsProvider = NotifierProvider<BypassListsController, List<BypassList>>(BypassListsController.new);

/// Whether the first-start introduction has been seen.
class OnboardedController extends Notifier<bool> {
  static const _key = 'onboarded';

  @override
  bool build() => ref.read(storeProvider).read<bool>(_key) ?? false;

  void done() {
    state = true;
    ref.read(storeProvider).write(_key, true);
  }
}

final onboardedProvider = NotifierProvider<OnboardedController, bool>(OnboardedController.new);

class SettingsState {
  const SettingsState({
    this.tunMode = false,
    this.systemProxy = true,
    this.stopOnClose = true,
    this.autoFailover = true,
    this.bypassLan = true,
    this.bypassOnly = false,
    this.remoteDns = defaultRemoteDns,
    this.splitMode = 'off',
    this.splitApps = const [],
    this.theme = 'dark',
    this.language = '',
  });

  static const defaultRemoteDns = 'https://1.1.1.1/dns-query';

  /// Capture all traffic. Off = proxy mode (works without admin rights on desktop).
  final bool tunMode;

  /// In proxy mode, set the OS proxy to the local port while connected.
  final bool systemProxy;
  final bool stopOnClose;
  final bool autoFailover;

  /// Local networks (LAN, link-local, multicast) don't go through the VPN.
  final bool bypassLan;

  /// With bypass lists: only the listed destinations use the VPN, everything else goes direct.
  final bool bypassOnly;

  /// Resolver used through the tunnel: https://, tls://, tcp://, udp:// or an IP.
  final String remoteDns;

  /// Split tunneling (Android): 'off', 'include' (only [splitApps] use the VPN)
  /// or 'exclude' ([splitApps] bypass the VPN).
  final String splitMode;
  final List<String> splitApps;

  /// 'dark' (default), 'light' or 'system'.
  final String theme;

  /// The app's language: a code from l10n.dart's supportedLanguages, or '' for the system's.
  final String language;

  SettingsState copyWith({
    bool? tunMode,
    bool? systemProxy,
    bool? stopOnClose,
    bool? autoFailover,
    bool? bypassLan,
    bool? bypassOnly,
    String? remoteDns,
    String? splitMode,
    List<String>? splitApps,
    String? theme,
    String? language,
  }) =>
      SettingsState(
        tunMode: tunMode ?? this.tunMode,
        systemProxy: systemProxy ?? this.systemProxy,
        stopOnClose: stopOnClose ?? this.stopOnClose,
        autoFailover: autoFailover ?? this.autoFailover,
        bypassLan: bypassLan ?? this.bypassLan,
        bypassOnly: bypassOnly ?? this.bypassOnly,
        remoteDns: remoteDns ?? this.remoteDns,
        splitMode: splitMode ?? this.splitMode,
        splitApps: splitApps ?? this.splitApps,
        theme: theme ?? this.theme,
        language: language ?? this.language,
      );

  Map<String, dynamic> toJson() => {
        'tun_mode': tunMode,
        'system_proxy': systemProxy,
        'stop_on_close': stopOnClose,
        'auto_failover': autoFailover,
        'bypass_lan': bypassLan,
        'bypass_only': bypassOnly,
        'remote_dns': remoteDns,
        'split_mode': splitMode,
        'split_apps': splitApps,
        'theme': theme,
        'language': language,
      };

  /// Missing keys keep the values from [defaults].
  factory SettingsState.fromJson(Map<String, dynamic> j, SettingsState defaults) => SettingsState(
        tunMode: j['tun_mode'] as bool? ?? defaults.tunMode,
        systemProxy: j['system_proxy'] as bool? ?? defaults.systemProxy,
        stopOnClose: j['stop_on_close'] as bool? ?? defaults.stopOnClose,
        autoFailover: j['auto_failover'] as bool? ?? defaults.autoFailover,
        bypassLan: j['bypass_lan'] as bool? ?? defaults.bypassLan,
        bypassOnly: j['bypass_only'] as bool? ?? defaults.bypassOnly,
        remoteDns: j['remote_dns'] as String? ?? defaults.remoteDns,
        splitMode: j['split_mode'] as String? ?? defaults.splitMode,
        splitApps: [for (final a in (j['split_apps'] as List? ?? const [])) a as String],
        theme: j['theme'] as String? ?? defaults.theme,
        language: j['language'] as String? ?? defaults.language,
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
