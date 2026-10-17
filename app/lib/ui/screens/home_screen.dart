import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../engine/engine.dart';
import '../../l10n/errors.dart';
import '../../l10n/l10n.dart';
import '../../state/location.dart';
import '../../state/providers.dart';
import '../theme.dart';
import '../widgets/country_flag.dart';
import '../../state/geo_data.dart';
import '../widgets/credentials_dialog.dart';
import '../widgets/earth_globe.dart';
import '../widgets/star_field.dart';
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
      String name(String id) =>
          profiles.where((p) => p.id == id).firstOrNull?.name ?? id;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          duration: const Duration(seconds: 6),
          content: Text(
            context.l10n.failoverToast(name(next.to), name(next.from)),
          ),
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
              Color.alphaBlend(
                statusColor(conn.status, scheme).withValues(alpha: 0.14),
                scheme.surface,
              ),
              scheme.surface,
            ],
            stops: const [0, 0.7],
          ),
        ),
        child: CustomPaint(
          painter: const StarFieldPainter(),
          child: LayoutBuilder(
            builder: (context, box) {
              final g = _Geometry(box, MediaQuery.paddingOf(context).top, MediaQuery.textScalerOf(context),
                  connected: conn.status == ConnStatus.connected);
              // The globe and the status sit at fixed places. Where they do not fit (a short screen,
              // a landscape phone, large text) the page scrolls instead of hiding behind the panel.
              final page = SizedBox(
                width: box.maxWidth,
                height: math.max(box.maxHeight, g.bottom),
                child: Stack(
                  children: [
                    // The Earth is the picture; the status hangs under it.
                    Positioned(
                      left: g.left,
                      top: g.top,
                      width: g.d,
                      height: g.d,
                      child: _Earth(state: conn),
                    ),
                    Positioned(
                      top: g.contentTop,
                      left: 0,
                      right: 0,
                      child: Center(
                        child: ConstrainedBox(
                          // On a wide window the same screen stays one narrow column.
                          constraints: const BoxConstraints(maxWidth: _columnWidth),
                          child: _Hero(state: conn),
                        ),
                      ),
                    ),
                  ],
                ),
              );
              return Stack(
                children: [
                  SingleChildScrollView(child: page),
                  SafeArea(
                    bottom: false,
                    child: Align(
                      alignment: Alignment.topCenter,
                      child: ConstrainedBox(
                        constraints: const BoxConstraints(maxWidth: _columnWidth),
                        child: const _TopBar(),
                      ),
                    ),
                  ),
                ],
              );
            },
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
            // A long error, large text or a small screen must not push the page off: the panel
            // takes at most half the height and scrolls.
            child: ConstrainedBox(
              constraints: BoxConstraints(maxHeight: MediaQuery.sizeOf(context).height * 0.5),
              child: SingleChildScrollView(child: _BottomPanel(state: conn, scheme: scheme)),
            ),
          ),
        ),
      ),
    );
  }
}

const _columnWidth = 480.0;

/// Connects, cancels or disconnects, asking for a password first when the
/// server needs one.
Future<void> _toggle(BuildContext context, WidgetRef ref) async {
  final ctrl = ref.read(connectionProvider.notifier);
  if (ref.read(connectionPlanProvider) == null ||
      ref.read(connectionProvider).status == ConnStatus.disconnecting) {
    return;
  }
  final profile = ref.read(selectedProfileProvider);
  if (!ctrl.isActive &&
      profile != null &&
      profile.needsCredentials &&
      !profile.hasCredentials) {
    if (!await askCredentials(context, ref, profile)) return;
  }
  ctrl.toggle();
}

/// The country the connection exits in: what the service saw, else the guess from
/// the server's name; null for Tor, when not connected, or when unknown.
String? _exitCode(WidgetRef ref, EngineState s) {
  if (s.status != ConnStatus.connected) return null;
  final active = ref.watch(profilesProvider).where((p) => p.id == s.activeProfileId).firstOrNull;
  if (active == null || active.protocol == 'tor') return null;
  return ref.watch(exitProvider)?.country ?? ref.watch(profileCountryProvider)(active);
}

/// Where things sit: the globe and the status block under it form one group, centred (a little
/// above the middle) between the top bar and the bottom panel.
class _Geometry {
  factory _Geometry(BoxConstraints box, double safeTop, TextScaler text, {required bool connected}) {
    // The status block: a title and a chip, or (connected) also the clock and the speed; as large as the text is.
    const topBar = 54.0, minGlobe = 56.0;
    final status = (connected ? 192.0 : 100.0) * math.min(text.scale(1), 1.6);
    final region = box.maxHeight - safeTop - topBar;
    final gap = region < 330 ? 10.0 : 22.0; // on a short screen the globe and its status huddle
    final d = math.max(minGlobe, math.min(math.min(box.maxWidth * 0.74, 400.0), region - gap - status));
    final free = math.max(0.0, region - (d + gap + status));
    final top = safeTop + topBar + free * 0.38;
    return _Geometry._(d, (box.maxWidth - d) / 2, top, gap, top + d + gap + status + 12);
  }

  _Geometry._(this.d, this.left, this.top, this._gap, this.bottom);

  final double d, left, top;
  final double _gap;

  /// Where the status block ends: the page is at least this tall.
  final double bottom;

  /// Below the globe: nothing is written on the Earth.
  double get contentTop => top + d + _gap;
}

/// The Earth. Not connected: it shows where you really are (red). Connecting and connected:
/// it flies to the exit country (gold, then teal) with an arc from where you are.
class _Earth extends ConsumerWidget {
  const _Earth({required this.state});

  final EngineState state;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final scheme = Theme.of(context).colorScheme;
    (double, double)? centerOf(String? code) => code == null ? null : countryCenters[code];

    // Where you are: only when address lookups are allowed (Settings).
    final allowed = ref.watch(settingsProvider.select((s) => s.showExit));
    final home = allowed ? centerOf(ref.watch(homeProvider)?.country) : null;

    // Where the connection exits: what the service saw once connected; while connecting, the
    // country of the server about to be used (a guess from its name or address).
    String? exitCode;
    if (state.status == ConnStatus.connected) {
      exitCode = _exitCode(ref, state);
    } else if (state.status == ConnStatus.connecting) {
      final primary = ref.watch(connectionPlanProvider)?.primary;
      if (primary != null && primary.protocol != 'tor') exitCode = ref.watch(profileCountryProvider)(primary);
    }

    return EarthBackdrop(
      home: home,
      exit: centerOf(exitCode),
      homeColor: disconnectRed,
      exitColor: statusColor(state.status, scheme),
    );
  }
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
          ClipRRect(
            borderRadius: BorderRadius.circular(9),
            child: Image.asset(
              'assets/logo.png',
              width: 26,
              height: 26,
              filterQuality: FilterQuality.medium,
            ),
          ),
          const SizedBox(width: 10),
          Text(
            context.l10n.appName,
            style: Theme.of(context).textTheme.titleMedium
                ?.copyWith(color: scheme.onSurface),
          ),
          const Spacer(),
          IconButton(
            tooltip: context.l10n.tooltipSettings,
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute<void>(builder: (_) => const SettingsScreen()),
            ),
            icon: const Icon(Icons.tune_rounded),
          ),
        ],
      ),
    );
  }
}

/// The status under the globe: what is happening, where you are or exit, and — once connected — the clock and speed.
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
    // Not protected is red, like the marker of where you are on the globe.
    final color = s.status == ConnStatus.disconnected ? scheme.error : statusColor(s.status, scheme);
    final countryOf = ref.watch(profileCountryProvider);
    final profiles = ref.watch(profilesProvider);
    final active = profiles.where((p) => p.id == s.activeProfileId).firstOrNull;
    final connected = s.status == ConnStatus.connected;
    final exit = ref.watch(exitProvider);
    final tor = active?.protocol == 'tor';
    // The country the service saw is the truth; the guess is the fallback.
    final code = tor ? null : exit?.country ?? (active == null ? null : countryOf(active));
    // Where you appear to be: the country, or Tor; null when it is not known.
    final place = tor ? context.l10n.torNetwork : (code == null ? null : context.countryName(code));
    // Where you really are, while not protected (only when address lookups are allowed).
    final allowed = ref.watch(settingsProvider.select((s) => s.showExit));
    final homeCode = allowed ? ref.watch(homeProvider)?.country : null;
    final idle = s.status == ConnStatus.disconnected || s.status == ConnStatus.error;
    final progress = s.progress;

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 24),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            connected && place != null ? context.l10n.connectedTo(place) : statusLabel(context.l10n, s.status),
            style: text.titleLarge?.copyWith(color: color),
            textAlign: TextAlign.center,
          ),
          // While connecting: the step, and a slim bar with the real progress.
          if (s.status == ConnStatus.connecting && progress != null) ...[
            const SizedBox(height: 8),
            AnimatedSwitcher(
              duration: const Duration(milliseconds: 200),
              child: Text(
                [progressDetail(context.l10n, progress) ?? stageLabel(context.l10n, progress.stage), ?progress.profile]
                    .join(' · '),
                key: ValueKey('${progress.stage}${progress.profile}${progress.detail}'),
                style: text.bodyMedium?.copyWith(color: scheme.onSurfaceVariant),
                textAlign: TextAlign.center,
              ),
            ),
          ],
          // Not connected: where you are right now.
          if (idle && homeCode != null) ...[
            const SizedBox(height: 14),
            _ExitChip(code: homeCode, tor: false, title: context.countryName(homeCode), accent: disconnectRed),
          ],
          if (connected && active != null) ...[
            const SizedBox(height: 16),
            _ExitChip(code: code, tor: tor, title: active.name, ip: exit?.ip, accent: color),
          ],
          if (connected && s.connectedSince != null) ...[
            const SizedBox(height: 16),
            Text(
              _elapsed(s.connectedSince!),
              style: text.titleLarge?.copyWith(
                fontWeight: FontWeight.w500,
                fontFeatures: const [FontFeature.tabularFigures()],
              ),
            ),
            const SizedBox(height: 10),
            _TrafficRow(up: s.upBps, down: s.downBps),
          ],
        ],
      ),
    );
  }
}

/// An address shown in part only: `147.78.x.x`, `2a07:d883:…`. Enough to recognise, not to track.
String maskIp(String ip) {
  if (ip.contains(':')) return '${ip.split(':').take(2).join(':')}:…';
  final o = ip.split('.');
  return o.length == 4 ? '${o[0]}.${o[1]}.x.x' : ip;
}

/// The flag of the exit country with the server's name and, when known, the exit address.
class _ExitChip extends StatelessWidget {
  const _ExitChip({
    required this.code,
    required this.tor,
    required this.title,
    this.ip,
    required this.accent,
  });

  final String? code;
  final bool tor;
  final String title;
  final String? ip;

  /// Colour of the chip's outline: red where you are unprotected, the status colour when connected.
  final Color accent;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final text = Theme.of(context).textTheme;
    return Container(
      padding: const EdgeInsetsDirectional.fromSTEB(6, 6, 14, 6),
      decoration: BoxDecoration(
        color: scheme.surfaceContainer,
        borderRadius: BorderRadius.circular(40),
        border: Border.all(color: accent.withValues(alpha: 0.45)),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          CountryFlag(
            code,
            size: 30,
            icon: tor ? Icons.security : Icons.public,
            label: tor ? context.l10n.torNetwork : null,
          ),
          const SizedBox(width: 12),
          Flexible(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  title,
                  style: text.titleSmall?.copyWith(letterSpacing: 0, fontWeight: FontWeight.w600),
                  overflow: TextOverflow.ellipsis,
                ),
                if (ip != null)
                  Text(
                    context.l10n.exitIpLabel(maskIp(ip!)),
                    textDirection: TextDirection.ltr,
                    style: text.bodySmall?.copyWith(
                      color: scheme.onSurfaceVariant,
                    ),
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
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: scheme.surfaceContainer,
        borderRadius: BorderRadius.circular(20),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 16, color: scheme.primary),
          const SizedBox(width: 6),
          Text(
            _fmt(bps),
            textDirection: TextDirection.ltr,
            style: Theme.of(context).textTheme.bodyMedium
                ?.copyWith(color: scheme.onSurfaceVariant),
          ),
        ],
      ),
    );
    // Wraps onto two lines where the two pills do not fit side by side (narrow screen, large text).
    return Wrap(
      alignment: WrapAlignment.center,
      spacing: 10,
      runSpacing: 6,
      children: [
        pill(Icons.south_rounded, down),
        pill(Icons.north_rounded, up),
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
    // On a short screen the panel keeps only what is needed to connect: the choice and the button.
    final compact = MediaQuery.sizeOf(context).height < 560;
    final profiles = ref.watch(profilesProvider);
    final plan = ref.watch(connectionPlanProvider);
    final choice = ref.watch(selectedProfileIdProvider);
    final countryOf = ref.watch(profileCountryProvider);

    final active = switch (state.status) {
      ConnStatus.connected ||
      ConnStatus.connecting ||
      ConnStatus.noInternet => true,
      _ => false,
    };
    final connected =
        state.status == ConnStatus.connected ||
        state.status == ConnStatus.noInternet;
    final busy = state.status == ConnStatus.disconnecting;

    // What the row says: the choice, not the server it ended up on.
    final code = countryOfChoice(choice);
    final specific = profiles.where((p) => p.id == choice).firstOrNull;
    final (flagCode, title, subtitle) = plan == null
        ? (null, context.l10n.noServers, context.l10n.noServersHint)
        : specific != null
        ? (
            countryOf(specific),
            specific.name,
            profileAddress(context, specific),
          )
        : code != null
        ? (code, context.countryName(code), context.l10n.anyServerInCountry)
        : (null, context.l10n.automatic, context.l10n.automaticHint);
    final rowIcon = specific == null
        ? Icons.bolt_rounded
        : (specific.protocol == 'tor' ? Icons.security : Icons.public);

    return Container(
      decoration: BoxDecoration(
        color: scheme.surfaceContainer,
        borderRadius: const BorderRadius.vertical(top: Radius.circular(32)),
        border: Border(top: BorderSide(color: scheme.outlineVariant)),
      ),
      padding: EdgeInsets.fromLTRB(16, compact ? 10 : 14, 16, compact ? 10 : 14),
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
                onTap: () => Navigator.of(context).push(
                  MaterialPageRoute<void>(
                    builder: (_) => const LocationScreen(),
                  ),
                ),
                child: Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 12,
                    vertical: 8,
                  ),
                  child: Row(
                    children: [
                      CountryFlag(
                        flagCode,
                        size: 30,
                        icon: rowIcon,
                        label: title,
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            if (!compact)
                              Text(
                                context.l10n.locationLabel,
                                style: text.labelSmall?.copyWith(
                                  color: scheme.onSurfaceVariant,
                                ),
                              ),
                            Text(
                              title,
                              style: text.titleSmall?.copyWith(letterSpacing: 0, fontWeight: FontWeight.w600),
                              overflow: TextOverflow.ellipsis,
                            ),
                            if (!compact)
                              Text(
                                subtitle,
                                style: text.bodySmall?.copyWith(
                                  color: scheme.onSurfaceVariant,
                                ),
                                overflow: TextOverflow.ellipsis,
                              ),
                          ],
                        ),
                      ),
                      Icon(
                        Icons.chevron_right_rounded,
                        color: scheme.onSurfaceVariant,
                      ),
                    ],
                  ),
                ),
              ),
            ),
            const SizedBox(height: 10),
            _MainButton(
              label: connected
                  ? context.l10n.buttonDisconnect
                  : (active
                        ? context.l10n.buttonCancel
                        : context.l10n.buttonConnect),
              kind: connected
                  ? _ButtonKind.stop
                  : (active ? _ButtonKind.cancel : _ButtonKind.go),
              onPressed: busy ? null : () => _toggle(context, ref),
              progress: state.status == ConnStatus.connecting ? state.progress?.percent : null,
            ),
          ],
        ],
      ),
    );
  }
}

enum _ButtonKind { go, stop, cancel }

/// The one big button: a gradient to connect, a red tint to disconnect, and while connecting it IS the
/// progress: it fills from the start side as the core reports its steps, with the percent in it; tapping cancels.
class _MainButton extends StatelessWidget {
  const _MainButton({required this.label, required this.kind, required this.onPressed, this.progress});

  final String label;
  final _ButtonKind kind;
  final VoidCallback? onPressed;

  /// 0-100 while connecting, otherwise null.
  final int? progress;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final (fill, fg) = switch (kind) {
      _ButtonKind.go => (null, Colors.white),
      _ButtonKind.stop => (disconnectRed.withValues(alpha: 0.16), disconnectRed),
      _ButtonKind.cancel => (scheme.surfaceContainerHighest, scheme.onSurface),
    };
    final p = progress;
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
          child: Stack(
            children: [
              if (p != null)
                Positioned.fill(
                  child: Align(
                    alignment: AlignmentDirectional.centerStart,
                    child: TweenAnimationBuilder<double>(
                      tween: Tween(end: p / 100),
                      duration: const Duration(milliseconds: 450),
                      curve: Curves.easeOutCubic,
                      builder: (context, value, _) => FractionallySizedBox(
                        key: const Key('connect-progress'),
                        widthFactor: value.clamp(0.0, 1.0),
                        heightFactor: 1,
                        child: DecoratedBox(
                          decoration: BoxDecoration(
                            gradient: LinearGradient(
                              colors: [
                                brandIndigo.withValues(alpha: 0.55),
                                brandViolet.withValues(alpha: 0.55),
                              ],
                            ),
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
              InkWell(
                onTap: onPressed,
                child: SizedBox(
                  height: 48,
                  child: Center(
                    child: Text(
                      p == null ? label : '$label · $p%',
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        color: fg,
                        fontWeight: FontWeight.w700,
                        fontFeatures: const [FontFeature.tabularFigures()],
                      ),
                    ),
                  ),
                ),
              ),
            ],
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
        Text(
          context.l10n.emptyTitle,
          style: Theme.of(context).textTheme.titleMedium,
        ),
        const SizedBox(height: 4),
        Text(context.l10n.emptyHint, textAlign: TextAlign.center),
        const SizedBox(height: 16),
        FilledButton.icon(
          onPressed: onImport,
          icon: const Icon(Icons.add),
          label: Text(context.l10n.importConfig),
        ),
      ],
    );
  }
}
