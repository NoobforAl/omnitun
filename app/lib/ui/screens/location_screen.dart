import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../engine/engine.dart';
import '../../l10n/l10n.dart';
import '../../state/countries.dart';
import '../../state/location.dart';
import '../../state/providers.dart';
import '../widgets/country_flag.dart';
import '../widgets/credentials_dialog.dart';
import '../widgets/protocol_badge.dart';
import 'profile_editor_screen.dart';
import 'profiles_screen.dart';
import 'share_screen.dart';
import 'subscription_screen.dart';

/// Measures every profile's latency, for the dots and for Auto.
Future<void> testAllLatencies(WidgetRef ref) async {
  final engine = ref.read(engineProvider);
  final profiles = ref.read(profilesProvider);
  await Future.wait(
    profiles.map((p) async {
      final ms = await engine.testLatency(p);
      ref.read(profilesProvider.notifier).setLatency(p.id, ms);
    }),
  );
}

/// Where to connect, like Mullvad's location list: Automatic, then countries
/// that open into their servers. Tapping a country picks any server there.
class LocationScreen extends ConsumerStatefulWidget {
  const LocationScreen({super.key});

  @override
  ConsumerState<LocationScreen> createState() => _LocationScreenState();
}

class _LocationScreenState extends ConsumerState<LocationScreen> {
  final _search = TextEditingController();
  final _open = <String?>{};
  bool _testing = false;

  @override
  void dispose() {
    _search.dispose();
    super.dispose();
  }

  Future<void> _choose(String? choice) async {
    final ctrl = ref.read(connectionProvider.notifier);
    final current = ref.read(selectedProfileIdProvider);
    final profiles = ref.read(profilesProvider);
    final profile = profiles.where((p) => p.id == choice).firstOrNull;
    if (profile != null && profile.needsCredentials && !profile.hasCredentials) {
      if (!await askCredentials(context, ref, profile) || !mounted) return;
    }
    final wasActive = ctrl.isActive;
    ref.read(selectedProfileIdProvider.notifier).select(choice);
    Navigator.of(context).pop();
    // Choosing a place while connected moves the connection there.
    if (wasActive && choice != current) await ctrl.reconnect();
  }

  Future<void> _test() async {
    setState(() => _testing = true);
    try {
      await testAllLatencies(ref);
    } finally {
      if (mounted) setState(() => _testing = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final profiles = ref.watch(profilesProvider);
    final choice = ref.watch(selectedProfileIdProvider);
    final countryOf = ref.watch(profileCountryProvider);
    final scheme = Theme.of(context).colorScheme;
    final text = Theme.of(context).textTheme;
    final query = _search.text.trim().toLowerCase();

    bool matches(CountryGroup g, ProfileInfo p) =>
        query.isEmpty ||
        p.name.toLowerCase().contains(query) ||
        p.server.toLowerCase().contains(query) ||
        (g.code != null &&
            (context.countryName(g.code!).toLowerCase().contains(query) ||
                (countryNames[g.code]?.toLowerCase().contains(query) ?? false)));

    final groups = groupByCountry(profiles, countryOf, context.countryName);
    final shown = [
      for (final g in groups)
        if (g.profiles.any((p) => matches(g, p))) g,
    ];
    final tor = profiles.where((p) => p.protocol == 'tor').toList();

    return Scaffold(
      appBar: AppBar(
        title: Text(context.l10n.selectLocation),
        actions: [
          _testing
              ? const Padding(
                  padding: EdgeInsets.all(14),
                  child: SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2)),
                )
              : IconButton(
                  tooltip: context.l10n.testSpeedTooltip,
                  onPressed: profiles.isEmpty ? null : _test,
                  icon: const Icon(Icons.speed),
                ),
          PopupMenuButton<String>(
            tooltip: context.l10n.addServersTooltip,
            icon: const Icon(Icons.add),
            onSelected: (v) {
              switch (v) {
                case 'import':
                  showImportSheet(context, ref);
                case 'new':
                  Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => const NewServerScreen()));
                case 'subscription':
                  Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => const SubscriptionScreen()));
                case 'manage':
                  Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => const ProfilesScreen()));
              }
            },
            itemBuilder: (_) => [
              PopupMenuItem(value: 'import', child: Text(context.l10n.menuImport)),
              PopupMenuItem(value: 'new', child: Text(context.l10n.menuNewServer)),
              PopupMenuItem(value: 'subscription', child: Text(context.l10n.menuSubscription)),
              PopupMenuItem(value: 'manage', child: Text(context.l10n.menuManage)),
            ],
          ),
        ],
      ),
      body: Column(
        children: [
          if (profiles.length > 6)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 4, 16, 8),
              child: TextField(
                controller: _search,
                onChanged: (_) => setState(() {}),
                decoration: InputDecoration(
                  hintText: context.l10n.searchHint,
                  prefixIcon: const Icon(Icons.search),
                  filled: true,
                  fillColor: scheme.surfaceContainer,
                  border: OutlineInputBorder(borderRadius: BorderRadius.circular(12), borderSide: BorderSide.none),
                  isDense: true,
                ),
              ),
            ),
          Expanded(
            child: Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 560),
                child: ListView(
                  padding: const EdgeInsets.only(bottom: 24),
                  children: [
                    if (query.isEmpty)
                      _Row(
                        leading: CountryFlag(null, icon: Icons.bolt, label: context.l10n.automatic),
                        title: context.l10n.automatic,
                        subtitle: context.l10n.automaticHint,
                        selected:
                            choice == null ||
                            (profiles.every((p) => p.id != choice) && countryOfChoice(choice) == null),
                        onTap: () => _choose(null),
                      ),
                    if (profiles.isEmpty)
                      Padding(
                        padding: const EdgeInsets.all(32),
                        child: Column(
                          children: [
                            Text(context.l10n.noServersYet, style: text.titleMedium),
                            const SizedBox(height: 4),
                            Text(context.l10n.importHintShort, textAlign: TextAlign.center),
                            const SizedBox(height: 16),
                            FilledButton.icon(
                              onPressed: () => showImportSheet(context, ref),
                              icon: const Icon(Icons.add),
                              label: Text(context.l10n.importConfig),
                            ),
                          ],
                        ),
                      ),
                    for (final g in shown) ..._group(context, g, choice, countryOf, matches, query.isNotEmpty),
                    if (tor.isNotEmpty && query.isEmpty) ..._torRows(tor, choice),
                  ],
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Iterable<Widget> _group(
    BuildContext context,
    CountryGroup g,
    String? choice,
    String? Function(ProfileInfo) countryOf,
    bool Function(CountryGroup, ProfileInfo) matches,
    bool searching,
  ) {
    final name = g.code == null ? context.l10n.otherGroup : context.countryName(g.code!);
    final expanded = searching || _open.contains(g.code);
    final inCountry = g.code != null && choice == countryChoice(g.code!);
    final count = g.profiles.length;
    return [
      _Row(
        leading: CountryFlag(g.code),
        title: name,
        subtitle: context.l10n.serversCount(count),
        selected: inCountry,
        dot: g.code == null ? null : g.bestLatency ?? (g.unreachable ? -1 : null),
        // A country picks any of its servers; "Other" only opens.
        onTap: g.code == null
            ? () => setState(() => _open.contains(g.code) ? _open.remove(g.code) : _open.add(g.code))
            : () => _choose(countryChoice(g.code!)),
        trailing: IconButton(
          tooltip: expanded ? context.l10n.hideServers : context.l10n.showServers,
          onPressed: searching
              ? null
              : () => setState(() => _open.contains(g.code) ? _open.remove(g.code) : _open.add(g.code)),
          icon: Icon(expanded ? Icons.expand_less : Icons.expand_more),
        ),
      ),
      if (expanded)
        for (final p in g.profiles)
          if (matches(g, p)) _ServerRow(profile: p, selected: choice == p.id, onTap: () => _choose(p.id)),
    ];
  }

  Iterable<Widget> _torRows(List<ProfileInfo> tor, String? choice) => [
    Padding(padding: const EdgeInsets.fromLTRB(16, 16, 16, 4), child: Text(context.l10n.torGroup)),
    for (final p in tor) _ServerRow(profile: p, selected: choice == p.id, onTap: () => _choose(p.id), tor: true),
  ];
}

/// A country or special row of the list.
class _Row extends StatelessWidget {
  const _Row({
    required this.leading,
    required this.title,
    required this.subtitle,
    required this.selected,
    required this.onTap,
    this.trailing,
    this.dot,
  });

  final Widget leading;
  final String title;
  final String subtitle;
  final bool selected;
  final VoidCallback onTap;
  final Widget? trailing;
  final int? dot;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return ListTile(
      leading: leading,
      title: Text(title, overflow: TextOverflow.ellipsis),
      subtitle: Text(subtitle),
      selected: selected,
      selectedTileColor: scheme.primary.withValues(alpha: 0.12),
      onTap: onTap,
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (selected) Icon(Icons.check, color: scheme.primary),
          if (dot != null) Padding(padding: const EdgeInsets.symmetric(horizontal: 8), child: LatencyDot(dot)),
          ?trailing,
        ],
      ),
    );
  }
}

/// One server (profile) under its country.
class _ServerRow extends ConsumerWidget {
  const _ServerRow({required this.profile, required this.selected, required this.onTap, this.tor = false});

  final ProfileInfo profile;
  final bool selected;
  final VoidCallback onTap;
  final bool tor;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final scheme = Theme.of(context).colorScheme;
    final p = profile;
    // Only the protocols the core has an editor for can be edited.
    final editable = ref.watch(formsProvider).value?.any((f) => f.protocol == p.protocol) ?? false;
    return ListTile(
      contentPadding: const EdgeInsetsDirectional.only(start: 40, end: 4),
      title: Text(p.name, overflow: TextOverflow.ellipsis),
      subtitle: Text(
        p.needsCredentials && !p.hasCredentials
            ? context.l10n.needsSignIn(profileAddress(context, p))
            : profileAddress(context, p),
        overflow: TextOverflow.ellipsis,
      ),
      selected: selected,
      selectedTileColor: scheme.primary.withValues(alpha: 0.12),
      onTap: onTap,
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (selected) Icon(Icons.check, color: scheme.primary),
          Padding(padding: const EdgeInsets.symmetric(horizontal: 8), child: LatencyDot(p.latencyMs)),
          ProtocolBadge(p.protocol),
          PopupMenuButton<String>(
            tooltip: context.l10n.tooltipMore,
            onSelected: (v) async {
              switch (v) {
                case 'edit':
                  Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => ProfileEditorScreen(profile: p)));
                case 'share':
                  Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => ShareScreen(profile: p)));
                case 'country':
                  final code = await pickCountry(context, ref.read(profileCountryProvider)(p));
                  if (code != null) ref.read(countryOverridesProvider.notifier).set(p.id, code == '' ? null : code);
                case 'credentials':
                  askCredentials(context, ref, p);
                case 'delete':
                  ref.read(profilesProvider.notifier).remove(p.id);
              }
            },
            itemBuilder: (_) => [
              if (editable)
                PopupMenuItem(
                  value: 'edit',
                  child: Text(tor ? context.l10n.menuBridgeSettings : context.l10n.menuEditServer),
                ),
              PopupMenuItem(value: 'share', child: Text(context.l10n.menuShare)),
              if (!tor) PopupMenuItem(value: 'country', child: Text(context.l10n.menuSetCountry)),
              if (p.needsCredentials) PopupMenuItem(value: 'credentials', child: Text(context.l10n.menuCredentials)),
              PopupMenuItem(value: 'delete', child: Text(context.l10n.actionDelete)),
            ],
          ),
        ],
      ),
    );
  }
}

/// Lets the user say where a server is. Returns the country code, '' for "guess
/// again", or null when cancelled.
Future<String?> pickCountry(BuildContext context, String? current) {
  return showDialog<String>(context: context, builder: (_) => _CountryDialog(current));
}

class _CountryDialog extends StatefulWidget {
  const _CountryDialog(this.current);

  final String? current;

  @override
  State<_CountryDialog> createState() => _CountryDialogState();
}

class _CountryDialogState extends State<_CountryDialog> {
  final _query = TextEditingController();

  @override
  void dispose() {
    _query.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final q = _query.text.trim().toLowerCase();
    final entries =
        [for (final code in countryNames.keys) MapEntry(code, context.countryName(code))]
            .where(
              (e) =>
                  q.isEmpty ||
                  e.value.toLowerCase().contains(q) ||
                  countryNames[e.key]!.toLowerCase().contains(q) ||
                  e.key.toLowerCase() == q,
            )
            .toList()
          ..sort((a, b) => a.value.compareTo(b.value));
    return AlertDialog(
      title: Text(context.l10n.countryOfServer),
      contentPadding: const EdgeInsets.only(top: 12),
      content: SizedBox(
        width: 360,
        height: 420,
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 20),
              child: TextField(
                controller: _query,
                autofocus: true,
                onChanged: (_) => setState(() {}),
                decoration: InputDecoration(hintText: context.l10n.searchHint, prefixIcon: const Icon(Icons.search)),
              ),
            ),
            Expanded(
              child: ListView(
                children: [
                  if (q.isEmpty)
                    ListTile(
                      leading: const CountryFlag(null),
                      title: Text(context.l10n.guessFromName),
                      onTap: () => Navigator.pop(context, ''),
                    ),
                  for (final e in entries)
                    ListTile(
                      leading: CountryFlag(e.key),
                      title: Text(e.value),
                      selected: e.key == widget.current,
                      onTap: () => Navigator.pop(context, e.key),
                    ),
                ],
              ),
            ),
          ],
        ),
      ),
      actions: [TextButton(onPressed: () => Navigator.pop(context), child: Text(context.l10n.actionCancel))],
    );
  }
}
