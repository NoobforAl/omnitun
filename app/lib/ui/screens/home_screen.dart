import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../engine/engine.dart';
import '../../l10n/l10n.dart';
import '../../state/countries.dart';
import '../../state/location.dart';
import '../../state/providers.dart';
import '../theme.dart';
import '../widgets/connection_ring.dart';
import '../widgets/country_flag.dart';
import '../widgets/credentials_dialog.dart';
import 'location_screen.dart';
import 'profiles_screen.dart';
import 'settings_screen.dart';

/// The one screen of the app: how safe you are, where you exit, one button.
/// Locations and settings open from here and close with Back; there is no
/// navigation bar.
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
          content: Text(context.l10n.failoverToast(name(next.to), name(next.from))),
        ),
      );
    });

    return Scaffold(
      body: DecoratedBox(
        // A wash of the status colour fading into the page: calm when off, green when safe.
        decoration: BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topCenter,
            end: Alignment.bottomCenter,
            colors: [
              Color.alphaBlend(statusColor(conn.status, scheme).withValues(alpha: 0.14), scheme.surface),
              scheme.surface,
            ],
            stops: const [0, 0.7],
          ),
        ),
        child: SafeArea(
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

/// Connects, cancels or disconnects, asking for a password first when the
/// server needs one. Shared by the power ring and the main button.
Future<void> _toggle(BuildContext context, WidgetRef ref) async {
  final ctrl = ref.read(connectionProvider.notifier);
  if (ref.read(connectionPlanProvider) == null || ref.read(connectionProvider).status == ConnStatus.disconnecting) {
    return;
  }
  final profile = ref.read(selectedProfileProvider);
  if (!ctrl.isActive && profile != null && profile.needsCredentials && !profile.hasCredentials) {
    if (!await askCredentials(context, ref, profile)) return;
  }
  ctrl.toggle();
}

class _TopBar extends StatelessWidget {
  const _TopBar();

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 10, 8, 0),
      child: Row(
        children: [
          Container(
            width: 30,
            height: 30,
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(9),
              gradient: const LinearGradient(
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
                colors: [brandIndigo, brandViolet],
              ),
            ),
            child: const Icon(Icons.shield_rounded, size: 18, color: Colors.white),
          ),
          const SizedBox(width: 10),
          Text(
            context.l10n.appName,
            style: Theme.of(context).textTheme.titleLarge?.copyWith(color: scheme.onSurface),
          ),
          const Spacer(),
          IconButton(
            tooltip: context.l10n.tooltipSettings,
            onPressed: () => Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => const SettingsScreen())),
            icon: const Icon(Icons.tune_rounded),
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
    final exit = ref.watch(exitProvider);
    final tor = active?.protocol == 'tor';
    // The country the service saw is the truth; the guess is the fallback.
    final code = tor ? null : exit?.country ?? (active == null ? null : countryOf(active));
    // Where you appear to be: the country, or Tor; null when it is not known.
    final place = tor ? context.l10n.torNetwork : (code == null ? null : countryNames[code] ?? code);

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          ConnectionRing(status: s.status, progress: s.progress, onTap: () => _toggle(context, ref)),
          const SizedBox(height: 28),
          Text(
            connected && place != null ? context.l10n.connectedTo(place) : statusLabel(context.l10n, s.status),
            style: text.headlineSmall?.copyWith(color: color),
            textAlign: TextAlign.center,
          ),
          // While connecting, say which step is running (and for which profile).
          if (s.status == ConnStatus.connecting && s.progress != null)
            Padding(
              padding: const EdgeInsets.only(top: 6),
              child: AnimatedSwitcher(
                duration: const Duration(milliseconds: 200),
                child: Text(
                  [s.progress!.detail ?? stageLabel(context.l10n, s.progress!.stage), ?s.progress!.profile].join(' · '),
                  key: ValueKey('${s.progress!.stage}${s.progress!.profile}${s.progress!.detail}'),
                  style: text.bodyMedium?.copyWith(color: scheme.onSurfaceVariant),
                  textAlign: TextAlign.center,
                ),
              ),
            ),
          if (connected && active != null) ...[
            const SizedBox(height: 18),
            _ExitChip(code: code, tor: tor, title: active.name, ip: exit?.ip),
          ],
          if (connected && s.connectedSince != null) ...[
            const SizedBox(height: 18),
            Text(
              _elapsed(s.connectedSince!),
              style: text.headlineSmall?.copyWith(
                fontWeight: FontWeight.w500,
                fontFeatures: const [FontFeature.tabularFigures()],
              ),
            ),
            const SizedBox(height: 12),
            _TrafficRow(up: s.upBps, down: s.downBps),
          ],
        ],
      ),
    );
  }
}

/// The flag of the exit country with the server's name and, when known, the exit address.
class _ExitChip extends StatelessWidget {
  const _ExitChip({required this.code, required this.tor, required this.title, this.ip});

  final String? code;
  final bool tor;
  final String title;
  final String? ip;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final text = Theme.of(context).textTheme;
    return Container(
      padding: const EdgeInsetsDirectional.fromSTEB(8, 8, 18, 8),
      decoration: BoxDecoration(
        color: scheme.surfaceContainer,
        borderRadius: BorderRadius.circular(40),
        border: Border.all(color: scheme.outlineVariant),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          CountryFlag(
            code,
            size: 40,
            icon: tor ? Icons.security : Icons.public,
            label: tor ? context.l10n.torNetwork : null,
          ),
          const SizedBox(width: 12),
          Flexible(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(title, style: text.titleMedium, overflow: TextOverflow.ellipsis),
                if (ip != null)
                  Text(
                    context.l10n.exitIpLabel(ip!),
                    textDirection: TextDirection.ltr,
                    style: text.bodySmall?.copyWith(color: scheme.onSurfaceVariant),
                  ),
              ],
            ),
          ),
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
    final scheme = Theme.of(context).colorScheme;
    Widget pill(IconData icon, int bps) => Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      decoration: BoxDecoration(color: scheme.surfaceContainer, borderRadius: BorderRadius.circular(20)),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 16, color: scheme.primary),
          const SizedBox(width: 6),
          Text(
            _fmt(bps),
            textDirection: TextDirection.ltr,
            style: Theme.of(context).textTheme.bodyMedium?.copyWith(color: scheme.onSurfaceVariant),
          ),
        ],
      ),
    );
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [pill(Icons.south_rounded, down), const SizedBox(width: 10), pill(Icons.north_rounded, up)],
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

    final active = switch (state.status) {
      ConnStatus.connected || ConnStatus.connecting || ConnStatus.noInternet => true,
      _ => false,
    };
    final connected = state.status == ConnStatus.connected || state.status == ConnStatus.noInternet;
    final busy = state.status == ConnStatus.disconnecting;

    // What the row says: the choice, not the server it ended up on.
    final code = countryOfChoice(choice);
    final specific = profiles.where((p) => p.id == choice).firstOrNull;
    final (flagCode, title, subtitle) = plan == null
        ? (null, context.l10n.noServers, context.l10n.noServersHint)
        : specific != null
        ? (countryOf(specific), specific.name, profileAddress(context, specific))
        : code != null
        ? (code, countryNames[code] ?? code, context.l10n.anyServerInCountry)
        : (null, context.l10n.automatic, context.l10n.automaticHint);
    final rowIcon = specific == null ? Icons.bolt_rounded : (specific.protocol == 'tor' ? Icons.security : Icons.public);

    return Container(
      decoration: BoxDecoration(
        color: scheme.surfaceContainer,
        borderRadius: const BorderRadius.vertical(top: Radius.circular(32)),
        border: Border(top: BorderSide(color: scheme.outlineVariant)),
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
                localizedError(context.l10n, state) ?? state.error!,
                style: text.bodyMedium?.copyWith(color: scheme.error),
                textAlign: TextAlign.center,
              ),
            ),
          if (plan == null)
            _EmptyHint(onImport: () => showImportSheet(context, ref))
          else ...[
            Material(
              color: scheme.surfaceContainerHigh,
              borderRadius: BorderRadius.circular(18),
              child: InkWell(
                borderRadius: BorderRadius.circular(18),
                onTap: () =>
                    Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => const LocationScreen())),
                child: Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
                  child: Row(
                    children: [
                      CountryFlag(flagCode, size: 40, icon: rowIcon, label: title),
                      const SizedBox(width: 14),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              context.l10n.locationLabel,
                              style: text.labelSmall?.copyWith(color: scheme.onSurfaceVariant),
                            ),
                            Text(title, style: text.titleMedium, overflow: TextOverflow.ellipsis),
                            Text(
                              subtitle,
                              style: text.bodySmall?.copyWith(color: scheme.onSurfaceVariant),
                              overflow: TextOverflow.ellipsis,
                            ),
                          ],
                        ),
                      ),
                      Icon(Icons.chevron_right_rounded, color: scheme.onSurfaceVariant),
                    ],
                  ),
                ),
              ),
            ),
            const SizedBox(height: 14),
            _MainButton(
              label: connected
                  ? context.l10n.buttonDisconnect
                  : (active ? context.l10n.buttonCancel : context.l10n.buttonConnect),
              kind: connected ? _ButtonKind.stop : (active ? _ButtonKind.cancel : _ButtonKind.go),
              onPressed: busy ? null : () => _toggle(context, ref),
            ),
          ],
        ],
      ),
    );
  }
}

enum _ButtonKind { go, stop, cancel }

/// The one big button: a gradient to connect, a red tint to disconnect, a quiet one to cancel.
class _MainButton extends StatelessWidget {
  const _MainButton({required this.label, required this.kind, required this.onPressed});

  final String label;
  final _ButtonKind kind;
  final VoidCallback? onPressed;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final (fill, fg) = switch (kind) {
      _ButtonKind.go => (null, Colors.white),
      _ButtonKind.stop => (disconnectRed.withValues(alpha: 0.16), disconnectRed),
      _ButtonKind.cancel => (scheme.surfaceContainerHighest, scheme.onSurface),
    };
    return Opacity(
      opacity: onPressed == null ? 0.5 : 1,
      child: Material(
        color: fill ?? Colors.transparent,
        borderRadius: BorderRadius.circular(18),
        clipBehavior: Clip.antiAlias,
        child: Ink(
          decoration: BoxDecoration(
            gradient: fill == null
                ? const LinearGradient(
                    colors: [brandIndigo, brandViolet],
                    begin: Alignment.centerLeft,
                    end: Alignment.centerRight,
                  )
                : null,
          ),
          child: InkWell(
            onTap: onPressed,
            child: SizedBox(
              height: 58,
              child: Center(
                child: Text(
                  label,
                  style: Theme.of(context).textTheme.titleMedium?.copyWith(color: fg, fontWeight: FontWeight.w700),
                ),
              ),
            ),
          ),
        ),
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
        Text(context.l10n.emptyTitle, style: Theme.of(context).textTheme.titleMedium),
        const SizedBox(height: 4),
        Text(context.l10n.emptyHint, textAlign: TextAlign.center),
        const SizedBox(height: 16),
        FilledButton.icon(onPressed: onImport, icon: const Icon(Icons.add), label: Text(context.l10n.importConfig)),
      ],
    );
  }
}
