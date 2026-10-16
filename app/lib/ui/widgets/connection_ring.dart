import 'package:flutter/material.dart';

import '../../engine/engine.dart';
import '../../l10n/l10n.dart';
import '../theme.dart';

/// The calm centre of the home screen: a lock inside a ring. While connecting
/// the ring fills 0-100 % with the core's real progress; after a failure it
/// stops, red, at the stage that failed. It pulses only while busy.
class ConnectionRing extends StatefulWidget {
  const ConnectionRing({super.key, required this.status, this.progress, this.size = 168});

  final ConnStatus status;
  final ConnectProgress? progress;
  final double size;

  @override
  State<ConnectionRing> createState() => _ConnectionRingState();
}

class _ConnectionRingState extends State<ConnectionRing> with SingleTickerProviderStateMixin {
  late final AnimationController _pulse = AnimationController(
    vsync: this,
    duration: const Duration(milliseconds: 1100),
  );

  bool get _busy => widget.status == ConnStatus.connecting || widget.status == ConnStatus.disconnecting;

  /// The ring is shown while connecting and, frozen, after a failed attempt.
  bool get _showRing =>
      widget.progress != null && (widget.status == ConnStatus.connecting || widget.status == ConnStatus.error);

  @override
  void initState() {
    super.initState();
    _syncAnimation();
  }

  @override
  void didUpdateWidget(ConnectionRing old) {
    super.didUpdateWidget(old);
    _syncAnimation();
  }

  void _syncAnimation() {
    if (_busy) {
      if (!_pulse.isAnimating) _pulse.repeat(reverse: true);
    } else {
      _pulse.stop();
      _pulse.value = 0;
    }
  }

  @override
  void dispose() {
    _pulse.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final color = statusColor(widget.status, scheme);
    final connected = widget.status == ConnStatus.connected;
    final connecting = widget.status == ConnStatus.connecting;
    final percent = widget.progress?.percent ?? 0;
    final inner = widget.size - 28;

    return Semantics(
      label: '${statusLabel(context.l10n, widget.status)}${_showRing ? ', $percent%' : ''}',
      child: SizedBox(
        width: widget.size,
        height: widget.size,
        child: Stack(
          alignment: Alignment.center,
          children: [
            // Soft glow behind the lock.
            AnimatedBuilder(
              animation: _pulse,
              builder: (context, child) => Container(
                width: inner,
                height: inner,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: scheme.surfaceContainer,
                  boxShadow: [
                    if (connected || _busy)
                      BoxShadow(
                        color: color.withValues(alpha: 0.18 + 0.2 * _pulse.value),
                        blurRadius: 28 + 20 * _pulse.value,
                        spreadRadius: 2 + 8 * _pulse.value,
                      ),
                  ],
                ),
              ),
            ),
            if (_showRing)
              TweenAnimationBuilder<double>(
                tween: Tween(end: percent / 100),
                duration: const Duration(milliseconds: 450),
                curve: Curves.easeOutCubic,
                builder: (context, value, _) => SizedBox(
                  width: widget.size,
                  height: widget.size,
                  child: CircularProgressIndicator(
                    value: value,
                    strokeWidth: 5,
                    strokeCap: StrokeCap.round,
                    color: color,
                    backgroundColor: color.withValues(alpha: 0.15),
                  ),
                ),
              )
            else
              Container(
                width: widget.size,
                height: widget.size,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  border: Border.all(color: color.withValues(alpha: connected ? 0.9 : 0.35), width: 3),
                ),
              ),
            if (connecting && _showRing)
              Text(
                '$percent%',
                style: Theme.of(context).textTheme.headlineSmall
                    ?.copyWith(color: color, fontFeatures: const [FontFeature.tabularFigures()]),
              )
            else
              Icon(statusIcon(widget.status), size: widget.size * 0.34, color: color),
          ],
        ),
      ),
    );
  }
}
