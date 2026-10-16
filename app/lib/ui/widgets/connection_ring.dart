import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../../engine/engine.dart';
import '../../l10n/l10n.dart';
import '../theme.dart';

/// The power button of the home screen: a glowing disc inside a ring. While
/// connecting the ring fills 0-100 % with the core's real progress; after a
/// failure it stops, red, at the stage that failed. It pulses only while busy.
class ConnectionRing extends StatefulWidget {
  const ConnectionRing({super.key, required this.status, this.progress, this.size = 184, this.onTap});

  final ConnStatus status;
  final ConnectProgress? progress;
  final double size;

  /// Tapping the ring does what the main button does.
  final VoidCallback? onTap;

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
    final inner = widget.size - 36;

    return Semantics(
      button: widget.onTap != null,
      label: '${statusLabel(context.l10n, widget.status)}${_showRing ? ', $percent%' : ''}',
      child: GestureDetector(
        onTap: widget.onTap,
        child: SizedBox(
          width: widget.size,
          height: widget.size,
          child: Stack(
            alignment: Alignment.center,
            children: [
              // The disc: a soft light from the status colour, glowing while it matters.
              AnimatedBuilder(
                animation: _pulse,
                builder: (context, child) => Container(
                  width: inner,
                  height: inner,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    gradient: RadialGradient(
                      center: const Alignment(0, -0.4),
                      colors: [
                        Color.alphaBlend(color.withValues(alpha: connected || _busy ? 0.30 : 0.10), scheme.surfaceContainerHigh),
                        scheme.surfaceContainer,
                      ],
                    ),
                    boxShadow: [
                      if (connected || _busy)
                        BoxShadow(
                          color: color.withValues(alpha: 0.22 + 0.22 * _pulse.value),
                          blurRadius: 36 + 24 * _pulse.value,
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
                      strokeWidth: 6,
                      strokeCap: StrokeCap.round,
                      color: color,
                      backgroundColor: color.withValues(alpha: 0.16),
                    ),
                  ),
                )
              else
                CustomPaint(
                  size: Size.square(widget.size),
                  painter: _RingPainter(color: color, full: connected),
                ),
              if (connecting && _showRing)
                Text(
                  '$percent%',
                  style: Theme.of(context).textTheme.headlineSmall
                      ?.copyWith(color: color, fontFeatures: const [FontFeature.tabularFigures()]),
                )
              else
                Icon(statusIcon(widget.status), size: widget.size * 0.36, color: color),
            ],
          ),
        ),
      ),
    );
  }
}

/// The ring at rest: a faint track, and once connected a full ring that blends
/// from the status colour to the brand colour.
class _RingPainter extends CustomPainter {
  _RingPainter({required this.color, required this.full});

  final Color color;
  final bool full;

  @override
  void paint(Canvas canvas, Size size) {
    const stroke = 6.0;
    final rect = Offset.zero & size;
    final arc = rect.deflate(stroke / 2);
    final paint = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = stroke;
    if (full) {
      paint.shader = SweepGradient(
        colors: [color, brandIndigo, color],
        transform: const GradientRotation(-math.pi / 2),
      ).createShader(rect);
    } else {
      paint.color = color.withValues(alpha: 0.28);
    }
    canvas.drawArc(arc, 0, math.pi * 2, false, paint);
  }

  @override
  bool shouldRepaint(_RingPainter old) => old.color != color || old.full != full;
}
