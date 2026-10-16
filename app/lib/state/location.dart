/// What the user picked as "where to connect", and the order in which the
/// profiles are tried for it.
///
/// The choice is stored as a string: `null` = Auto, `country:DE` = any server
/// in a country, anything else = one profile's ID (what older versions saved).
library;

import '../engine/engine.dart';

const _countryPrefix = 'country:';

String countryChoice(String code) => '$_countryPrefix$code';

/// The country code of a "country:XX" choice, or null.
String? countryOfChoice(String? choice) =>
    choice != null && choice.startsWith(_countryPrefix) ? choice.substring(_countryPrefix.length) : null;

/// What Connect will do: the profile to start with and the ones to fall back to.
class ConnectionPlan {
  const ConnectionPlan(this.primary, this.fallbacks);

  final ProfileInfo primary;
  final List<ProfileInfo> fallbacks;
}

/// Orders profiles by what is known about their speed: reachable ones from the
/// fastest, then untested ones in list order, then unreachable ones.
List<ProfileInfo> byLatency(Iterable<ProfileInfo> profiles) {
  int rank(ProfileInfo p) => switch (p.latencyMs) {
    null => 1,
    < 0 => 2,
    _ => 0,
  };
  final list = profiles.toList();
  final index = {for (var i = 0; i < list.length; i++) list[i].id: i};
  list.sort((a, b) {
    final r = rank(a).compareTo(rank(b));
    if (r != 0) return r;
    if (rank(a) == 0) {
      final l = a.latencyMs!.compareTo(b.latencyMs!);
      if (l != 0) return l;
    }
    return index[a.id]!.compareTo(index[b.id]!);
  });
  return list;
}

/// Profiles that Auto and country choices may start on. Tor only runs when the
/// user picks it, and a profile that wants a password the app doesn't have
/// can't start on its own.
bool _automatic(ProfileInfo p) => p.protocol != 'tor' && !(p.needsCredentials && !p.hasCredentials);

/// The order in which [profiles] are tried for [choice], or null without profiles.
/// [countryOf] gives the country of a profile (null = unknown).
ConnectionPlan? planConnection(List<ProfileInfo> profiles, String? choice, String? Function(ProfileInfo) countryOf) {
  if (profiles.isEmpty) return null;

  final specific = profiles.where((p) => p.id == choice).firstOrNull;
  if (specific != null) {
    // The rest of its group in list order, then all other profiles.
    final group = profiles.where((p) => p.group == specific.group).toList();
    final i = group.indexWhere((p) => p.id == specific.id);
    return ConnectionPlan(specific, [
      ...group.skip(i + 1),
      ...group.take(i),
      ...profiles.where((p) => p.group != specific.group),
    ]);
  }

  final auto = byLatency(profiles.where(_automatic));
  final code = countryOfChoice(choice);
  final ordered = code == null
      ? auto
      : [...auto.where((p) => countryOf(p) == code), ...auto.where((p) => countryOf(p) != code)];
  if (ordered.isEmpty) {
    // Only profiles that can't start on their own: show the first, the user fixes it.
    return ConnectionPlan(profiles.first, profiles.skip(1).toList());
  }
  return ConnectionPlan(ordered.first, ordered.skip(1).toList());
}

/// Profiles grouped by country for the location picker, countries in order of
/// their best server; profiles without a known country come last as [other].
class CountryGroup {
  const CountryGroup(this.code, this.profiles);

  /// ISO code, or null for the profiles whose country is unknown.
  final String? code;
  final List<ProfileInfo> profiles;

  /// Fastest known latency in the group, or null when none was tested.
  int? get bestLatency {
    final ok = profiles.map((p) => p.latencyMs).whereType<int>().where((ms) => ms >= 0);
    return ok.isEmpty ? null : ok.reduce((a, b) => a < b ? a : b);
  }

  /// All servers unreachable (as far as tested).
  bool get unreachable => profiles.every((p) => (p.latencyMs ?? 0) < 0 && p.latencyMs != null);
}

List<CountryGroup> groupByCountry(
  List<ProfileInfo> profiles,
  String? Function(ProfileInfo) countryOf,
  String Function(String code) nameOf,
) {
  final by = <String?, List<ProfileInfo>>{};
  for (final p in profiles) {
    if (p.protocol == 'tor') continue; // Tor has its own row
    by.putIfAbsent(countryOf(p), () => []).add(p);
  }
  final known = [
    for (final e in by.entries)
      if (e.key != null) CountryGroup(e.key, byLatency(e.value)),
  ]..sort((a, b) => nameOf(a.code!).compareTo(nameOf(b.code!)));
  return [...known, if (by[null] != null) CountryGroup(null, by[null]!)];
}
