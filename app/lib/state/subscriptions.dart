import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../engine/engine.dart';
import 'providers.dart';

/// How a subscription request authenticates (mirrors core/subscription.Auth).
class SubscriptionAuth {
  const SubscriptionAuth({
    this.type = 'none',
    this.username = '',
    this.password = '',
    this.token = '',
    this.header = '',
    this.queryKey = '',
  });

  /// none, basic, bearer, header or query.
  final String type;
  final String username;
  final String password;
  final String token;
  final String header;
  final String queryKey;

  Map<String, dynamic> toJson() => {
        'type': type,
        if (username.isNotEmpty) 'username': username,
        if (password.isNotEmpty) 'password': password,
        if (token.isNotEmpty) 'token': token,
        if (header.isNotEmpty) 'header': header,
        if (queryKey.isNotEmpty) 'query_key': queryKey,
      };

  factory SubscriptionAuth.fromJson(Map<String, dynamic> j) => SubscriptionAuth(
        type: j['type'] as String? ?? 'none',
        username: j['username'] as String? ?? '',
        password: j['password'] as String? ?? '',
        token: j['token'] as String? ?? '',
        header: j['header'] as String? ?? '',
        queryKey: j['query_key'] as String? ?? '',
      );
}

class Subscription {
  const Subscription({
    required this.id,
    required this.name,
    required this.url,
    this.auth = const SubscriptionAuth(),
    this.info = const SubscriptionInfo(),
    this.updatedAt,
    this.lastError,
  });

  final String id;
  final String name;
  final String url;
  final SubscriptionAuth auth;
  final SubscriptionInfo info;
  final DateTime? updatedAt;
  final String? lastError;

  /// The provider's interval, or once a day.
  Duration get refreshInterval =>
      Duration(hours: info.updateIntervalHours > 0 ? info.updateIntervalHours : 24);

  bool get isDue => updatedAt == null || DateTime.now().difference(updatedAt!) >= refreshInterval;

  Map<String, dynamic> toRequest() => {'url': url, 'auth': auth.toJson()};

  Subscription copyWith({
    String? name,
    String? url,
    SubscriptionAuth? auth,
    SubscriptionInfo? info,
    DateTime? updatedAt,
    String? lastError,
    bool clearError = false,
  }) =>
      Subscription(
        id: id,
        name: name ?? this.name,
        url: url ?? this.url,
        auth: auth ?? this.auth,
        info: info ?? this.info,
        updatedAt: updatedAt ?? this.updatedAt,
        lastError: clearError ? null : lastError ?? this.lastError,
      );

  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'url': url,
        'auth': auth.toJson(),
        'info': info.toJson(),
        if (updatedAt != null) 'updated_at': updatedAt!.toIso8601String(),
        if (lastError != null) 'last_error': lastError,
      };

  factory Subscription.fromJson(Map<String, dynamic> j) => Subscription(
        id: j['id'] as String,
        name: j['name'] as String,
        url: j['url'] as String,
        auth: SubscriptionAuth.fromJson(Map<String, dynamic>.from(j['auth'] as Map? ?? const {})),
        info: SubscriptionInfo.fromJson(Map<String, dynamic>.from(j['info'] as Map? ?? const {})),
        updatedAt: DateTime.tryParse(j['updated_at'] as String? ?? ''),
        lastError: j['last_error'] as String?,
      );
}

/// Subscriptions, persisted. Each one owns the profiles in the group with its ID.
class SubscriptionsController extends Notifier<List<Subscription>> {
  static const _key = 'subscriptions';
  final _refreshing = <String>{};

  @override
  List<Subscription> build() {
    final saved = ref.read(storeProvider).read<List<dynamic>>(_key) ?? const [];
    return [
      for (final s in saved)
        if (s is Map) Subscription.fromJson(Map<String, dynamic>.from(s)),
    ];
  }

  void _save(List<Subscription> next) {
    state = next;
    ref.read(storeProvider).write(_key, [for (final s in next) s.toJson()]);
  }

  void _put(Subscription sub) => _save([for (final s in state) s.id == sub.id ? sub : s]);

  bool isRefreshing(String id) => _refreshing.contains(id);

  /// Fetches a new subscription and keeps it only if the fetch works.
  /// Returns the number of profiles; throws with a user-facing message.
  Future<int> add({required String url, String name = '', SubscriptionAuth auth = const SubscriptionAuth()}) async {
    final id = 'sub-${DateTime.now().microsecondsSinceEpoch.toRadixString(36)}';
    final draft = Subscription(id: id, name: name, url: url.trim(), auth: auth);
    final result = await ref.read(engineProvider).fetchSubscription(draft.toRequest());
    final sub = draft.copyWith(
      name: name.isNotEmpty ? name : (result.info.title.isNotEmpty ? result.info.title : Uri.parse(draft.url).host),
      info: result.info,
      updatedAt: DateTime.now(),
    );
    _save([...state, sub]);
    ref.read(profilesProvider.notifier).replaceGroup(id, result.profiles);
    return result.profiles.length;
  }

  /// Re-downloads a subscription and replaces its profiles. Failures are
  /// recorded on the subscription (its profiles are kept) and rethrown.
  Future<int> refresh(String id) async {
    final sub = state.where((s) => s.id == id).firstOrNull;
    if (sub == null || !_refreshing.add(id)) return 0;
    state = [...state]; // let listeners show the spinner
    try {
      final result = await ref.read(engineProvider).fetchSubscription(sub.toRequest());
      ref.read(profilesProvider.notifier).replaceGroup(id, result.profiles);
      _put(sub.copyWith(info: result.info, updatedAt: DateTime.now(), clearError: true));
      return result.profiles.length;
    } catch (e) {
      _put(sub.copyWith(lastError: _message(e)));
      rethrow;
    } finally {
      _refreshing.remove(id);
      state = [...state];
    }
  }

  /// Refreshes every subscription whose interval has passed; errors are recorded, not thrown.
  Future<void> refreshDue() async {
    for (final s in [...state]) {
      if (s.isDue) {
        try {
          await refresh(s.id);
        } catch (_) {}
      }
    }
  }

  void update(Subscription sub) => _put(sub);

  void remove(String id) {
    _save(state.where((s) => s.id != id).toList());
    ref.read(profilesProvider.notifier).removeGroup(id);
  }

  static String _message(Object e) {
    final s = e.toString();
    return s.startsWith('Bad state: ') ? s.substring(11) : s;
  }
}

final subscriptionsProvider =
    NotifierProvider<SubscriptionsController, List<Subscription>>(SubscriptionsController.new);

/// Bytes as a short human string (KB, MB, GB).
String formatBytes(int bytes) {
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  var v = bytes.toDouble();
  var i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  return '${v.toStringAsFixed(v >= 100 || i == 0 ? 0 : 1)} ${units[i]}';
}
