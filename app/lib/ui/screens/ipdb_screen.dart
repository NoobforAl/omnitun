import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../l10n/l10n.dart';
import '../../state/ipdb.dart';
import '../../state/ipdb_state.dart';
import '../theme.dart';

/// "12.3 MB"
String formatBytes(int bytes) {
  if (bytes >= 1000000) return '${(bytes / 1000000).toStringAsFixed(1)} MB';
  if (bytes >= 1000) return '${(bytes / 1000).toStringAsFixed(0)} KB';
  return '$bytes B';
}

/// "01:05"
String formatClock(Duration d) {
  String two(int n) => n.toString().padLeft(2, '0');
  return '${two(d.inMinutes)}:${two(d.inSeconds % 60)}';
}

/// The Settings row's second line.
String ipDbSubtitle(AppLocalizations l, IpDbState s) {
  final percent = ((s.progress?.fraction ?? 0) * 100).floor();
  return switch (s.phase) {
    IpDbPhase.downloading => l.ipdbSubDownloading(percent),
    IpDbPhase.preparing => l.ipdbPreparing,
    IpDbPhase.paused || IpDbPhase.failed => l.ipdbSubStopped(percent),
    IpDbPhase.ready => (s.enabled ? l.ipdbSubReady : l.ipdbSubOff)(formatBytes(s.info?.bytes ?? 0)),
    IpDbPhase.none => l.ipdbSubNone,
  };
}

/// The optional offline country database: what it does, how to get it, and — while it downloads — a
/// ring that fills as the bytes arrive. A download that stops (an error, Pause, the app closing)
/// keeps what it fetched and Continue picks up right there.
class IpDbScreen extends ConsumerWidget {
  const IpDbScreen({super.key});

  Future<void> _confirmDelete(BuildContext context, WidgetRef ref) async {
    final l = context.l10n;
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(l.ipdbDeleteTitle),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: Text(l.actionCancel)),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: disconnectRed),
            onPressed: () => Navigator.pop(context, true),
            child: Text(l.actionDelete),
          ),
        ],
      ),
    );
    if (ok == true) await ref.read(ipDbProvider.notifier).delete();
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l = context.l10n;
    final s = ref.watch(ipDbProvider);
    final ctrl = ref.read(ipDbProvider.notifier);
    final scheme = Theme.of(context).colorScheme;
    final text = Theme.of(context).textTheme;
    final working = s.phase == IpDbPhase.downloading || s.phase == IpDbPhase.preparing;
    final stopped = s.phase == IpDbPhase.paused || s.phase == IpDbPhase.failed;
    final p = s.progress;

    return Scaffold(
      appBar: AppBar(title: Text(l.ipdbTitle)),
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 520),
          child: ListView(
            padding: const EdgeInsets.fromLTRB(20, 8, 20, 32),
            children: [
              if (s.phase == IpDbPhase.none) ...[
                Text(l.ipdbIntro, style: text.bodyLarge?.copyWith(color: scheme.onSurfaceVariant)),
                const SizedBox(height: 4),
              ],
              const SizedBox(height: 12),
              Center(child: _Ring(state: s)),
              const SizedBox(height: 20),
              if (s.phase != IpDbPhase.none || s.installed) _Steps(state: s),
              const SizedBox(height: 16),

              // What is happening, in numbers.
              if (s.phase == IpDbPhase.downloading && p != null) _Stats(progress: p),
              if (stopped && p != null)
                Center(
                  child: Text(
                    l.ipdbStoppedAt((p.fraction * 100).floor()),
                    textAlign: TextAlign.center,
                    style: text.bodyMedium?.copyWith(color: scheme.onSurfaceVariant),
                  ),
                ),
              if (s.phase == IpDbPhase.failed && s.error != null) ...[
                const SizedBox(height: 8),
                Center(
                  child: Text(
                    l.ipdbFailed(_reason(l, s.error!)),
                    textAlign: TextAlign.center,
                    style: text.bodySmall?.copyWith(color: scheme.error),
                  ),
                ),
              ],
              if (s.phase == IpDbPhase.none)
                Center(
                  child: Text(
                    l.ipdbSizeNote('23 MB'),
                    style: text.bodySmall?.copyWith(color: scheme.onSurfaceVariant),
                  ),
                ),
              const SizedBox(height: 20),

              // The buttons.
              if (s.phase == IpDbPhase.none)
                _Primary(icon: Icons.download_rounded, label: l.ipdbDownload, onPressed: ctrl.start),
              if (s.phase == IpDbPhase.downloading)
                OutlinedButton.icon(
                  onPressed: ctrl.pause,
                  icon: const Icon(Icons.pause_rounded),
                  label: Text(l.ipdbPause),
                  style: OutlinedButton.styleFrom(minimumSize: const Size.fromHeight(52)),
                ),
              if (stopped) ...[
                _Primary(icon: Icons.play_arrow_rounded, label: l.ipdbContinue, onPressed: ctrl.start),
                if (s.installed || p != null)
                  TextButton(
                    onPressed: () => _confirmDelete(context, ref),
                    child: Text(l.ipdbDelete, style: TextStyle(color: scheme.error)),
                  ),
              ],

              // An installed database.
              if (s.installed && !working && !stopped) ...[
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          l.ipdbReadyLine(_count(s.info!.ranges), formatBytes(s.info!.bytes)),
                          style: text.titleMedium,
                        ),
                        const SizedBox(height: 4),
                        Text(
                          l.ipdbVersion(s.info!.version, _date(s.info!.builtAt)),
                          style: text.bodySmall?.copyWith(color: scheme.onSurfaceVariant),
                        ),
                      ],
                    ),
                  ),
                ),
                const SizedBox(height: 12),
                Card(
                  clipBehavior: Clip.antiAlias,
                  child: SwitchListTile(
                    title: Text(l.ipdbUse),
                    value: s.enabled,
                    onChanged: ctrl.setEnabled,
                  ),
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    Expanded(
                      child: OutlinedButton.icon(
                        onPressed: ctrl.start,
                        icon: const Icon(Icons.refresh_rounded),
                        label: Text(l.ipdbUpdate),
                        style: OutlinedButton.styleFrom(minimumSize: const Size.fromHeight(48)),
                      ),
                    ),
                    const SizedBox(width: 12),
                    TextButton.icon(
                      onPressed: () => _confirmDelete(context, ref),
                      icon: Icon(Icons.delete_outline_rounded, color: scheme.error),
                      label: Text(l.ipdbDelete, style: TextStyle(color: scheme.error)),
                    ),
                  ],
                ),
              ],
              const SizedBox(height: 24),
              Text(
                l.ipdbSource,
                textAlign: TextAlign.center,
                style: text.bodySmall?.copyWith(color: scheme.onSurfaceVariant),
              ),
            ],
          ),
        ),
      ),
    );
  }

  /// The error code of the downloader in words.
  static String _reason(AppLocalizations l, String code) => switch (code) {
    'network' => l.ipdbErrNetwork,
    'mirrors' => l.ipdbErrMirrors,
    _ => code,
  };

  static String _count(int n) => n >= 1000 ? '${(n / 1000).toStringAsFixed(0)}k' : '$n';

  static String _date(DateTime t) =>
      '${t.year}-${t.month.toString().padLeft(2, '0')}-${t.day.toString().padLeft(2, '0')}';
}

class _Primary extends StatelessWidget {
  const _Primary({required this.icon, required this.label, required this.onPressed});

  final IconData icon;
  final String label;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return Material(
      borderRadius: BorderRadius.circular(18),
      clipBehavior: Clip.antiAlias,
      color: Colors.transparent,
      child: Ink(
        decoration: const BoxDecoration(
          gradient: LinearGradient(colors: [brandIndigo, brandViolet]),
        ),
        child: InkWell(
          onTap: onPressed,
          child: SizedBox(
            height: 52,
            child: Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Icon(icon, color: Colors.white),
                const SizedBox(width: 8),
                Text(
                  label,
                  style: Theme.of(context).textTheme.titleMedium?.copyWith(
                    color: Colors.white,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

/// Bytes so far, speed and time left.
class _Stats extends StatelessWidget {
  const _Stats({required this.progress});

  final IpDbProgress progress;

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final scheme = Theme.of(context).colorScheme;
    final style = Theme.of(context).textTheme.bodyMedium?.copyWith(
      color: scheme.onSurfaceVariant,
      fontFeatures: const [FontFeature.tabularFigures()],
    );
    final left = progress.remaining;
    Widget cell(IconData icon, String text) => Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 16, color: scheme.primary),
        const SizedBox(width: 6),
        Text(text, style: style, textDirection: TextDirection.ltr),
      ],
    );
    return Wrap(
      alignment: WrapAlignment.center,
      spacing: 18,
      runSpacing: 6,
      children: [
        cell(Icons.downloading_rounded, '${formatBytes(progress.overallReceived)} / ${formatBytes(progress.overallTotal)}'),
        if (progress.bytesPerSecond > 1) cell(Icons.speed_rounded, '${formatBytes(progress.bytesPerSecond.round())}/s'),
        if (left != null) cell(Icons.timer_outlined, l.ipdbLeft(formatClock(left))),
      ],
    );
  }
}

/// IPv4 → IPv6 → Prepare, each done, in progress or waiting.
class _Steps extends StatelessWidget {
  const _Steps({required this.state});

  final IpDbState state;

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    // 0 = waiting, 1 = in progress, 2 = done
    List<int> marks;
    switch (state.phase) {
      case IpDbPhase.ready:
        marks = [2, 2, 2];
      case IpDbPhase.preparing:
        marks = [2, 2, 1];
      case IpDbPhase.downloading:
      case IpDbPhase.paused:
      case IpDbPhase.failed:
        final part = state.progress?.part ?? 0;
        marks = [part > 0 ? 2 : 1, part > 0 ? 1 : 0, 0];
      case IpDbPhase.none:
        marks = [0, 0, 0];
    }
    final labels = [l.ipdbStepV4, l.ipdbStepV6, l.ipdbStepPrepare];
    return Row(
      children: [
        for (var i = 0; i < 3; i++) ...[
          if (i > 0) Expanded(child: _Line(done: marks[i] > 0)),
          _Dot(label: labels[i], mark: marks[i], failed: state.phase == IpDbPhase.failed && marks[i] == 1),
        ],
      ],
    );
  }
}

class _Line extends StatelessWidget {
  const _Line({required this.done});

  final bool done;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Container(
      height: 2,
      margin: const EdgeInsets.only(bottom: 22, left: 4, right: 4),
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(1),
        color: done ? connectedGreen.withValues(alpha: 0.7) : scheme.outlineVariant,
      ),
    );
  }
}

class _Dot extends StatelessWidget {
  const _Dot({required this.label, required this.mark, required this.failed});

  final String label;
  final int mark;
  final bool failed;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final color = failed ? scheme.error : (mark == 2 ? connectedGreen : (mark == 1 ? connectingAmber : scheme.outline));
    return SizedBox(
      width: 64,
      child: Column(
        children: [
          AnimatedContainer(
            duration: const Duration(milliseconds: 300),
            width: 28,
            height: 28,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color: mark == 2 ? color.withValues(alpha: 0.18) : Colors.transparent,
              border: Border.all(color: color.withValues(alpha: mark == 0 ? 0.5 : 1), width: 2),
            ),
            child: mark == 2
                ? Icon(Icons.check_rounded, size: 16, color: color)
                : (mark == 1
                      ? Center(child: Container(width: 8, height: 8, decoration: BoxDecoration(shape: BoxShape.circle, color: color)))
                      : null),
          ),
          const SizedBox(height: 6),
          Text(
            label,
            style: Theme.of(context).textTheme.labelSmall?.copyWith(
              color: mark == 0 ? scheme.onSurfaceVariant : scheme.onSurface,
            ),
          ),
        ],
      ),
    );
  }
}

/// The big ring: a gradient arc that grows with the download, rune ticks inside that light up one
/// by one, and in the middle the percent (or a symbol when there is nothing to count).
class _Ring extends StatefulWidget {
  const _Ring({required this.state});

  final IpDbState state;

  @override
  State<_Ring> createState() => _RingState();
}

class _RingState extends State<_Ring> with SingleTickerProviderStateMixin {
  late final AnimationController _spin = AnimationController(vsync: this, duration: const Duration(milliseconds: 1400));

  bool get _preparing => widget.state.phase == IpDbPhase.preparing;

  @override
  void initState() {
    super.initState();
    _sync();
  }

  @override
  void didUpdateWidget(_Ring old) {
    super.didUpdateWidget(old);
    _sync();
  }

  // It turns only while preparing (a short moment), never otherwise.
  void _sync() {
    if (_preparing) {
      if (!_spin.isAnimating) _spin.repeat();
    } else {
      _spin.stop();
    }
  }

  @override
  void dispose() {
    _spin.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final text = Theme.of(context).textTheme;
    final s = widget.state;
    final failed = s.phase == IpDbPhase.failed;
    final paused = s.phase == IpDbPhase.paused;
    final ready = s.phase == IpDbPhase.ready;
    final fraction = ready ? 1.0 : (s.progress?.fraction ?? 0.0);
    final color = failed
        ? scheme.error
        : paused
        ? connectingAmber
        : ready
        ? connectedGreen
        : scheme.primary;
    final percent = (fraction * 100).floor();

    Widget center;
    if (ready) {
      center = Icon(Icons.verified_rounded, size: 64, color: color);
    } else if (s.phase == IpDbPhase.none) {
      center = Icon(Icons.public_rounded, size: 64, color: scheme.onSurfaceVariant);
    } else if (_preparing) {
      center = Icon(Icons.memory_rounded, size: 56, color: color);
    } else {
      center = Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            '$percent%',
            style: text.headlineMedium?.copyWith(
              color: color,
              fontWeight: FontWeight.w700,
              fontFeatures: const [FontFeature.tabularFigures()],
            ),
          ),
          if (paused || failed) Icon(paused ? Icons.pause_rounded : Icons.priority_high_rounded, size: 20, color: color),
        ],
      );
    }

    return Semantics(
      label: ipDbSubtitle(context.l10n, s),
      child: SizedBox(
        width: 220,
        height: 220,
        child: TweenAnimationBuilder<double>(
          tween: Tween(end: fraction),
          duration: const Duration(milliseconds: 350),
          curve: Curves.easeOutCubic,
          builder: (context, value, _) => AnimatedBuilder(
            animation: _spin,
            builder: (context, _) => CustomPaint(
              painter: _RingPainter(
                fraction: value,
                spin: _preparing ? _spin.value : null,
                color: color,
                track: scheme.outlineVariant,
                dim: scheme.outline,
                glow: !(s.phase == IpDbPhase.none),
              ),
              child: Center(child: center),
            ),
          ),
        ),
      ),
    );
  }
}

class _RingPainter extends CustomPainter {
  _RingPainter({required this.fraction, required this.spin, required this.color, required this.track, required this.dim, required this.glow});

  final double fraction;

  /// 0-1 turns of the "working" arc while preparing; null otherwise.
  final double? spin;
  final Color color, track, dim;
  final bool glow;

  @override
  void paint(Canvas canvas, Size size) {
    const stroke = 10.0;
    final c = size.center(Offset.zero);
    final r = size.width / 2 - stroke;
    final rect = Rect.fromCircle(center: c, radius: r);

    canvas.drawCircle(
      c,
      r,
      Paint()
        ..style = PaintingStyle.stroke
        ..strokeWidth = stroke
        ..color = track.withValues(alpha: 0.6),
    );

    const start = -math.pi / 2;
    if (spin != null) {
      // Preparing: a short arc going round.
      final a = start + spin! * math.pi * 2;
      canvas.drawArc(
        rect,
        a,
        math.pi * 0.7,
        false,
        Paint()
          ..style = PaintingStyle.stroke
          ..strokeWidth = stroke
          ..strokeCap = StrokeCap.round
          ..color = color,
      );
    } else if (fraction > 0.001) {
      final sweep = math.pi * 2 * fraction;
      final arc = Paint()
        ..style = PaintingStyle.stroke
        ..strokeWidth = stroke
        ..strokeCap = StrokeCap.round
        ..shader = SweepGradient(
          startAngle: start,
          endAngle: start + math.max(sweep, 0.05),
          colors: [Color.lerp(color, brandIndigo, 0.6)!, color],
        ).createShader(rect);
      if (glow) {
        canvas.drawArc(
          rect,
          start,
          sweep,
          false,
          Paint()
            ..style = PaintingStyle.stroke
            ..strokeWidth = stroke + 6
            ..strokeCap = StrokeCap.round
            ..color = color.withValues(alpha: 0.22)
            ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 8),
        );
      }
      canvas.drawArc(rect, start, sweep, false, arc);
    }

    // Runes: sixty ticks inside the ring, lit one by one as the bytes arrive.
    const ticks = 60;
    final outer = r - stroke - 8;
    final lit = (fraction * ticks).floor();
    for (var i = 0; i < ticks; i++) {
      final a = start + math.pi * 2 * i / ticks;
      final on = i < lit || fraction >= 1;
      final len = i.isEven ? 9.0 : 5.0;
      canvas.drawLine(
        c + Offset(math.cos(a), math.sin(a)) * (outer - len),
        c + Offset(math.cos(a), math.sin(a)) * outer,
        Paint()
          ..strokeWidth = 2
          ..strokeCap = StrokeCap.round
          ..color = on ? color.withValues(alpha: 0.9) : dim.withValues(alpha: 0.35),
      );
    }
  }

  @override
  bool shouldRepaint(_RingPainter old) =>
      old.fraction != fraction || old.spin != spin || old.color != color || old.glow != glow;
}
