import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../engine/core_engine.dart';
import '../engine/engine.dart';
import 'countries.dart';
import 'ipdb_state.dart';
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

  void remove(String id) {
    _save(state.where((p) => p.id != id).toList());
    _fixSelection();
  }

  /// Deletes every server (all subscriptions' too) except the built-in Tor ones, which are the
  /// app's own default and never come from the user's files.
  void clearAll() {
    _save(state.where((p) => p.protocol == 'tor').toList());
    _fixSelection();
  }

  /// When the chosen server is gone, or Auto has nothing left to start on, choose the default.
  void _fixSelection() {
    final choice = ref.read(selectedProfileIdProvider);
    if (choice != null && countryOfChoice(choice) != null) return;
    final gone = choice != null && !state.any((p) => p.id == choice);
    final nothingAuto = choice == null && !state.any((p) => p.protocol != 'tor');
    if (!gone && !nothingAuto) return;
    final fallback = resolveDefault(state, ref.read(defaultProfileIdProvider));
    if (fallback != null) ref.read(selectedProfileIdProvider.notifier).select(fallback.id);
  }

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

  void removeGroup(String group) {
    _save(state.where((p) => p.group != group).toList());
    _fixSelection();
  }

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

/// Tor with Snowflake bridges, obfs4 bridges and direct: the app's own default configuration. It
/// needs nothing from the user and is always available; their servers come on top of it.
const torDefaultLinks = ['tor://snowflake', 'tor://obfs4', 'tor://'];

/// The kind of bridges a Tor profile uses ("snowflake", "obfs4"), or null (direct, or not Tor).
String? torKind(ProfileInfo p) {
  if (p.protocol != 'tor') return null;
  return (p.raw['tor'] as Map?)?['builtin'] as String?;
}

/// The profile the user marked as default; null = the built-in default (Tor with Snowflake).
class DefaultProfileController extends Notifier<String?> {
  static const _key = 'default_profile';

  /// Set when the app itself added the Tor defaults to an empty list: the introduction is still due.
  static const freshInstall = 'fresh_install';

  @override
  String? build() => ref.read(storeProvider).read<String>(_key);

  void set(String? id) {
    state = id;
    ref.read(storeProvider).write(_key, id);
  }

  /// Adds the Tor defaults on a brand new install (and whenever [force] asks, after the user deleted
  /// them) and selects the default then. Returns how many were added.
  Future<int> ensureTorDefaults({bool force = false}) async {
    final store = ref.read(storeProvider);
    const seeded = 'defaults_seeded';
    if (!force && store.read<bool>(seeded) == true) return 0;
    final fresh = ref.read(profilesProvider).isEmpty;
    if (!force && !fresh) {
      // Someone who already set the app up has their own list; Restore Tor defaults is in the menu.
      store.write(seeded, true);
      return 0;
    }
    final ImportResult result;
    try {
      result = await ref.read(engineProvider).importText(torDefaultLinks.join('\n'));
    } catch (_) {
      return 0; // the core is not ready; the next start tries again
    }
    final added = ref.read(profilesProvider.notifier).addAll(result.profiles);
    store.write(seeded, true);
    if (fresh && !force) store.write(freshInstall, true);
    final choice = ref.read(selectedProfileIdProvider);
    if ((fresh || force) && choice == null) {
      final d = resolveDefault(ref.read(profilesProvider), state);
      if (d != null) ref.read(selectedProfileIdProvider.notifier).select(d.id);
    }
    return added;
  }
}

final defaultProfileIdProvider =
    NotifierProvider<DefaultProfileController, String?>(DefaultProfileController.new);

/// The default profile: the one the user chose, else Tor with Snowflake, else any Tor profile.
/// What the app selects when it has no valid choice (first start, a deleted server).
final defaultProfileProvider = Provider<ProfileInfo?>(
  (ref) => resolveDefault(ref.watch(profilesProvider), ref.watch(defaultProfileIdProvider)),
);

/// [defaultProfileProvider] as a function, for the controllers that cannot depend on it.
ProfileInfo? resolveDefault(List<ProfileInfo> profiles, String? chosen) {
  final mine = profiles.where((p) => p.id == chosen).firstOrNull;
  if (mine != null) return mine;
  return profiles.where((p) => torKind(p) == 'snowflake').firstOrNull ??
      profiles.where((p) => p.protocol == 'tor').firstOrNull;
}

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

/// Countries seen when connected (the exit lookup), by profile ID.
class LearnedCountries extends Notifier<Map<String, String>> {
  static const _key = 'learned_countries';

  @override
  Map<String, String> build() {
    final saved = ref.read(storeProvider).read<Map<String, dynamic>>(_key) ?? const {};
    return {for (final e in saved.entries) e.key: e.value as String};
  }

  void learn(String id, String code) {
    if (state[id] == code) return;
    final next = {...state, id: code};
    state = next;
    ref.read(storeProvider).write(_key, next);
  }
}

final learnedCountriesProvider = NotifierProvider<LearnedCountries, Map<String, String>>(LearnedCountries.new);

/// The country of a profile: the user's choice, else the one seen when connected, else the one the
/// offline database gives for the server's address (when it is installed), else a guess from the
/// server's name and address.
final profileCountryProvider = Provider<String? Function(ProfileInfo)>((ref) {
  final overrides = ref.watch(countryOverridesProvider);
  final learned = ref.watch(learnedCountriesProvider);
  final fromDb = ref.watch(dbCountriesProvider);
  final active = ref.watch(ipDbActiveProvider) != null; // a database that is switched off or deleted says nothing
  final guesses = <String, String?>{};
  return (p) =>
      overrides[p.id] ??
      learned[p.id] ??
      (active ? fromDb[p.server] : null) ??
      guesses.putIfAbsent(p.id, () => guessCountry(p));
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

/// What the exit lookup found for the current connection (null until it did, or when it is off).
class ExitController extends Notifier<ExitInfo?> {
  int _generation = 0;

  @override
  ExitInfo? build() {
    ref.listen(connectionProvider.select((s) => (s.status, s.activeProfileId)), (prev, next) {
      final (status, id) = next;
      if (status != ConnStatus.connected) {
        _generation++;
        if (state != null) state = null;
        return;
      }
      // Once per connection (and again after a failover moved it).
      if (prev?.$1 == ConnStatus.connected && prev?.$2 == id) return;
      _lookup(id);
    });
    return null;
  }

  Future<void> _lookup(String? profileId) async {
    if (!ref.read(settingsProvider).showExit) return;
    final generation = ++_generation;
    state = null;
    try {
      final info = withDbCountry(await ref.read(engineProvider).exitInfo(), ref.read(ipDbActiveProvider));
      if (generation != _generation) return; // connected elsewhere, or disconnected, meanwhile
      state = info;
      // Tor's exit changes with every circuit, so it says nothing about the profile.
      final profile = ref.read(profilesProvider).where((p) => p.id == profileId).firstOrNull;
      if (info.country != null && profile != null && profile.protocol != 'tor') {
        ref.read(learnedCountriesProvider.notifier).learn(profile.id, info.country!);
      }
    } catch (_) {
      // The address is a nicety; a failed lookup shows nothing.
    }
  }
}

final exitProvider = NotifierProvider<ExitController, ExitInfo?>(ExitController.new);

/// True when running under `flutter test`: no real network, no real waiting.
bool get _underTest => Platform.environment.containsKey('FLUTTER_TEST');

/// Asks a public service where this device's own address is, directly (not through any tunnel).
typedef DirectLookup = Future<ExitInfo> Function();

final directLookupProvider = Provider<DirectLookup>((ref) {
  /// One GET, read as text.
  Future<String> get(String url) async {
    final client = HttpClient()..connectionTimeout = const Duration(seconds: 6);
    try {
      final request = await client.getUrl(Uri.parse(url)).timeout(const Duration(seconds: 8));
      final response = await request.close().timeout(const Duration(seconds: 8));
      if (response.statusCode != 200) throw HttpException('status ${response.statusCode}');
      return await response.transform(const Utf8Decoder()).join().timeout(const Duration(seconds: 8));
    } finally {
      client.close(force: true);
    }
  }

  return () async {
    // Tests never reach the network.
    if (_underTest) throw const SocketException('no network in tests');
    // Some networks block one of the services, so there are two; the first that answers wins.
    final sources = <Future<ExitInfo> Function()>[
      () async {
        // Cloudflare's trace: "key=value" lines with ip= and loc=.
        final lines = (await get('https://1.1.1.1/cdn-cgi/trace')).split('\n');
        final fields = {
          for (final l in lines)
            if (l.contains('=')) l.substring(0, l.indexOf('=')): l.substring(l.indexOf('=') + 1).trim(),
        };
        final loc = fields['loc'];
        return ExitInfo(fields['ip'] ?? '', loc == null || loc.length != 2 ? null : loc);
      },
      () async => ExitInfo.fromJson(jsonDecode(await get('https://api.country.is/')) as Map<String, dynamic>),
    ];
    Object? last;
    for (final source in sources) {
      try {
        final info = await source();
        if (info.country != null) return info;
      } catch (e) {
        last = e;
      }
    }
    throw last ?? const HttpException('no answer');
  };
});

/// Where the device really is, by its address: shown on the globe while the VPN is off (like Mullvad's
/// "you are here"). Looked up only while not connected and only when the user allows address lookups
/// (Settings); the last answer is remembered so the globe has it at once on the next start.
class HomeLocation extends Notifier<ExitInfo?> {
  static const _key = 'home_country';
  int _generation = 0;

  @override
  ExitInfo? build() {
    ref.listen(connectionProvider.select((s) => s.status), (prev, next) {
      if (next == ConnStatus.disconnected && prev != next) _lookup(const Duration(milliseconds: 1500));
    });
    ref.listen(settingsProvider.select((s) => s.showExit), (prev, next) {
      if (next && !prev!) _lookup();
    });
    Future.microtask(_lookup);
    final saved = ref.read(storeProvider).read<String>(_key);
    return saved == null ? null : ExitInfo('', saved);
  }

  Future<void> _lookup([Duration delay = Duration.zero]) async {
    final generation = ++_generation;
    if (delay > Duration.zero && !_underTest) await Future<void>.delayed(delay);
    bool idle() =>
        generation == _generation &&
        ref.mounted &&
        ref.read(settingsProvider).showExit &&
        const {ConnStatus.disconnected, ConnStatus.error}.contains(ref.read(connectionProvider).status);
    if (!idle()) return;
    try {
      final info = withDbCountry(await ref.read(directLookupProvider)(), ref.read(ipDbActiveProvider));
      if (!idle()) return; // connected meanwhile: that answer would describe the VPN, not the device
      state = info;
      if (info.country != null) ref.read(storeProvider).write(_key, info.country);
    } catch (_) {
      // Offline or blocked: keep what is known.
    }
  }
}

final homeProvider = NotifierProvider<HomeLocation, ExitInfo?>(HomeLocation.new);

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
    this.showExit = true,
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

  /// After connecting, ask a public service (through the VPN) for the exit address and country.
  final bool showExit;

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
    bool? showExit,
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
        showExit: showExit ?? this.showExit,
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
        'show_exit': showExit,
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
        showExit: j['show_exit'] as bool? ?? defaults.showExit,
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
