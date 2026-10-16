import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../engine/engine.dart';
import '../../state/countries.dart';
import '../../state/location.dart';
import '../../state/providers.dart';
import '../theme.dart';
import '../widgets/connection_ring.dart';
import '../widgets/country_flag.dart';
import '../widgets/credentials_dialog.dart';
import 'location_screen.dart';
import 'logs_screen.dart';
import 'profiles_screen.dart';
import 'settings_screen.dart';

/// The one screen of the app: how safe you are, where you exit, one button.
/// Locations, logs and settings open from here and close with Back; there is
/// no navigation bar.
class HomeScreen extends ConsumerWidget {
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final conn = ref.watch(connectionProvider);
    final scheme = Theme.of(context).colorScheme;

    // Tell the user when the core switched profiles on its own, and why.
    ref.listen(connectionProvider.select((s) => s.failover), (prev, next) {
      if (next == null || next.at == prev?.at) return;
      final profiles = ref.read(profilesProvider);
      String name(String id) => profiles.where((p) => p.id == id).firstOrNull?.name ?? id;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          duration: const Duration(seconds: 6),
          content: Text('Switched to ${name(next.to)} — ${name(next.from)} stopped responding'),
        ),
      );
    });

    return Scaffold(
      body: SafeArea(
        bottom: false,
        child: Center(
          child: ConstrainedBox(
            // On a wide window the same screen stays one narrow column.
            constraints: const BoxConstraints(maxWidth: _columnWidth),
            child: Column(
              children: [
                const _TopBar(),
                Expanded(
                  child: LayoutBuilder(
                    builder: (context, box) => SingleChildScrollView(
                      child: ConstrainedBox(
                        constraints: BoxConstraints(minHeight: box.maxHeight),
                        child: Center(child: _Hero(state: conn)),
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
      // In the footer slot so that toasts appear above the panel, not over the button.
      // It is not navigation: nothing here switches pages.
      bottomNavigationBar: SafeArea(
        child: Center(
          heightFactor: 1,
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: _columnWidth),
            child: _BottomPanel(state: conn, scheme: scheme),
          ),
        ),
      ),
    );
  }
}

const _columnWidth = 480.0;

class _TopBar extends StatelessWidget {
  const _TopBar();

  void _open(BuildContext context, Widget page) =>
      Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => page));

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 8, 8, 0),
      child: Row(
        children: [
          Text(
            'OMNITUN',
            style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w800, letterSpacing: 2.4),
          ),
          const Spacer(),
          IconButton(
            tooltip: 'Logs',
            onPressed: () => _open(context, const LogsScreen()),
            icon: const Icon(Icons.receipt_long_outlined),
          ),
          IconButton(
            tooltip: 'Settings',
            onPressed: () => _open(context, const SettingsScreen()),
            icon: const Icon(Icons.settings_outlined),
          ),
        ],
      ),
    );
  }
}

/// Ring, state, and — once connected — the country you exit from.
class _Hero extends ConsumerStatefulWidget {
  const _Hero({required this.state});

  final EngineState state;

  @override
  ConsumerState<_Hero> createState() => _HeroState();
}

class _HeroState extends ConsumerState<_Hero> {
  Timer? _ticker;

  @override
  void initState() {
    super.initState();
    _ticker = Timer.periodic(const Duration(seconds: 1), (_) {
      if (widget.state.connectedSince != null && mounted) setState(() {});
    });
  }

  @override
  void dispose() {
    _ticker?.cancel();
    super.dispose();
  }

  String _elapsed(DateTime since) {
    final d = DateTime.now().difference(since);
    String two(int n) => n.toString().padLeft(2, '0');
    return '${two(d.inHours)}:${two(d.inMinutes % 60)}:${two(d.inSeconds % 60)}';
  }

  @override
  Widget build(BuildContext context) {
    final s = widget.state;
    final scheme = Theme.of(context).colorScheme;
    final text = Theme.of(context).textTheme;
    final color = statusColor(s.status, scheme);
    final countryOf = ref.watch(profileCountryProvider);
    final profiles = ref.watch(profilesProvider);
    final active = profiles.where((p) => p.id == s.activeProfileId).firstOrNull;
    final connected = s.status == ConnStatus.connected;
    final code = active == null ? null : countryOf(active);

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          ConnectionRing(status: s.status, progress: s.progress),
          const SizedBox(height: 28),
          Text(
            statusLabel(s.status),
            style: text.titleLarge?.copyWith(color: color, fontWeight: FontWeight.w700),
            textAlign: TextAlign.center,
          ),
          // While connecting, say which step is running (and for which profile).
          if (s.status == ConnStatus.connecting && s.progress != null)
            Padding(
              padding: const EdgeInsets.only(top: 6),
              child: AnimatedSwitcher(
                duration: const Duration(milliseconds: 200),
                child: Text(
                  [s.progress!.detail ?? stageLabel(s.progress!.stage), ?s.progress!.profile].join(' · '),
                  key: ValueKey('${s.progress!.stage}${s.progress!.profile}${s.progress!.detail}'),
                  style: text.bodyMedium?.copyWith(color: scheme.onSurfaceVariant),
                  textAlign: TextAlign.center,
                ),
              ),
            ),
          if (connected && active != null) ...[
            const SizedBox(height: 14),
            // Where you exit: the country, or Tor; nothing when the country is not known.
            if (code != null || active.protocol == 'tor')
              Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  CountryFlag(code, size: 28, icon: Icons.security),
                  const SizedBox(width: 10),
                  Flexible(
                    child: Text(
                      active.protocol == 'tor' ? 'Tor network' : countryNames[code] ?? code!,
                      style: text.titleMedium,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                ],
              ),
            const SizedBox(height: 4),
            Text(
              active.name,
              style: text.bodyMedium?.copyWith(color: scheme.onSurfaceVariant),
              overflow: TextOverflow.ellipsis,
              textAlign: TextAlign.center,
            ),
          ],
          if (connected && s.connectedSince != null) ...[
            const SizedBox(height: 14),
            Text(
              _elapsed(s.connectedSince!),
              style: text.bodyLarge?.copyWith(fontFeatures: const [FontFeature.tabularFigures()]),
            ),
            const SizedBox(height: 10),
            _TrafficRow(up: s.upBps, down: s.downBps),
          ],
        ],
      ),
    );
  }
}

class _TrafficRow extends StatelessWidget {
  const _TrafficRow({required this.up, required this.down});

  final int up;
  final int down;

  static String _fmt(int bps) {
    if (bps >= 1 << 20) return '${(bps / (1 << 20)).toStringAsFixed(1)} MB/s';
    if (bps >= 1 << 10) return '${(bps / (1 << 10)).toStringAsFixed(0)} KB/s';
    return '$bps B/s';
  }

  @override
  Widget build(BuildContext context) {
    final style = Theme.of(context).textTheme.bodyMedium
        ?.copyWith(color: Theme.of(context).colorScheme.onSurfaceVariant);
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        const Icon(Icons.arrow_downward, size: 16),
        Text(' ${_fmt(down)}', style: style),
        const SizedBox(width: 20),
        const Icon(Icons.arrow_upward, size: 16),
        Text(' ${_fmt(up)}', style: style),
      ],
    );
  }
}

/// Where you connect to, and the button.
class _BottomPanel extends ConsumerWidget {
  const _BottomPanel({required this.state, required this.scheme});

  final EngineState state;
  final ColorScheme scheme;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    final profiles = ref.watch(profilesProvider);
    final plan = ref.watch(connectionPlanProvider);
    final choice = ref.watch(selectedProfileIdProvider);
    final countryOf = ref.watch(profileCountryProvider);
    final ctrl = ref.read(connectionProvider.notifier);

    final active = switch (state.status) {
      ConnStatus.connected || ConnStatus.connecting || ConnStatus.noInternet => true,
      _ => false,
    };
    final connected = state.status == ConnStatus.connected || state.status == ConnStatus.noInternet;
    final busy = state.status == ConnStatus.disconnecting;
    final buttonText = active && !connected ? scheme.onSecondaryContainer : Colors.white;

    // What the row says: the choice, not the server it ended up on.
    final code = countryOfChoice(choice);
    final specific = profiles.where((p) => p.id == choice).firstOrNull;
    final (flagCode, title, subtitle) = plan == null
        ? (null, 'No servers', 'Add one to get started')
        : specific != null
        ? (countryOf(specific), specific.name, specific.address)
        : code != null
        ? (code, countryNames[code] ?? code, 'Any server in this country')
        : (null, 'Automatic', 'Fastest server, switches if one stops working');
    final rowIcon = specific == null ? Icons.bolt : (specific.protocol == 'tor' ? Icons.security : Icons.public);

    return Container(
      decoration: BoxDecoration(
        color: scheme.surfaceContainer,
        borderRadius: const BorderRadius.vertical(top: Radius.circular(24)),
      ),
      padding: const EdgeInsets.fromLTRB(20, 20, 20, 20),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (state.status == ConnStatus.error && state.error != null)
            Padding(
              padding: const EdgeInsets.only(bottom: 14),
              child: Text(
                state.error!,
                style: text.bodyMedium?.copyWith(color: scheme.error),
                textAlign: TextAlign.center,
              ),
            ),
          if (plan == null)
            _EmptyHint(onImport: () => showImportSheet(context, ref))
          else ...[
            Material(
              color: scheme.surfaceContainerHigh,
              borderRadius: BorderRadius.circular(14),
              child: InkWell(
                borderRadius: BorderRadius.circular(14),
                onTap: () =>
                    Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => const LocationScreen())),
                child: Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
                  child: Row(
                    children: [
                      CountryFlag(flagCode, size: 36, icon: rowIcon, label: title),
                      const SizedBox(width: 14),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text('Location', style: text.labelSmall?.copyWith(color: scheme.onSurfaceVariant)),
                            Text(title, style: text.titleMedium, overflow: TextOverflow.ellipsis),
                            Text(
                              subtitle,
                              style: text.bodySmall?.copyWith(color: scheme.onSurfaceVariant),
                              overflow: TextOverflow.ellipsis,
                            ),
                          ],
                        ),
                      ),
                      const Icon(Icons.chevron_right),
                    ],
                  ),
                ),
              ),
            ),
            const SizedBox(height: 14),
            SizedBox(
              height: 56,
              child: FilledButton(
                style: FilledButton.styleFrom(
                  backgroundColor: connected ? disconnectRed : (active ? scheme.secondaryContainer : connectedGreen),
                  foregroundColor: buttonText,
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
                ),
                onPressed: busy
                    ? null
                    : () async {
                        final profile = ref.read(selectedProfileProvider);
                        if (!ctrl.isActive && profile != null && profile.needsCredentials && !profile.hasCredentials) {
                          if (!await askCredentials(context, ref, profile)) return;
                        }
                        ctrl.toggle();
                      },
                child: Text(
                  connected ? 'Disconnect' : (active ? 'Cancel' : 'Connect'),
                  style: text.titleMedium?.copyWith(fontWeight: FontWeight.w700, color: buttonText),
                ),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _EmptyHint extends StatelessWidget {
  const _EmptyHint({required this.onImport});

  final VoidCallback onImport;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Text('No profiles yet', style: Theme.of(context).textTheme.titleMedium),
        const SizedBox(height: 4),
        const Text('Import a config file, link or subscription to get started.', textAlign: TextAlign.center),
        const SizedBox(height: 16),
        FilledButton.icon(onPressed: onImport, icon: const Icon(Icons.add), label: const Text('Import config')),
      ],
    );
  }
}
