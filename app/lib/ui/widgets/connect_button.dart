import 'package:flutter/material.dart';

import '../../engine/engine.dart';
import '../theme.dart';

/// The large central connect toggle. While connecting, a ring around it fills
/// 0-100 % with the core's real progress; on failure the ring stops, red, at
/// the stage that failed. Pulses only while busy.
class ConnectButton extends StatefulWidget {
  const ConnectButton({
    super.key,
    required this.status,
    required this.onPressed,
    this.progress,
    this.size = 180,
  });

  final ConnStatus status;
  final VoidCallback? onPressed;
  final ConnectProgress? progress;
  final double size;

  @override
  State<ConnectButton> createState() => _ConnectButtonState();
}

class _ConnectButtonState extends State<ConnectButton> with SingleTickerProviderStateMixin {
  late final AnimationController _pulse =
      AnimationController(vsync: this, duration: const Duration(milliseconds: 1100));

  bool get _busy =>
      widget.status == ConnStatus.connecting || widget.status == ConnStatus.disconnecting;

  /// The ring is shown while connecting and, frozen, after a failed attempt.
  bool get _showRing =>
      widget.progress != null &&
      (widget.status == ConnStatus.connecting || widget.status == ConnStatus.error);

  @override
  void initState() {
    super.initState();
    _syncAnimation();
  }

  @override
  void didUpdateWidget(ConnectButton old) {
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
    final label = connected || _busy ? 'Disconnect' : 'Connect';
    final percent = widget.progress?.percent ?? 0;
    final ringSize = widget.size + 28;

    final button = AnimatedBuilder(
      animation: _pulse,
      builder: (context, child) => Container(
        width: widget.size,
        height: widget.size,
        decoration: BoxDecoration(
          shape: BoxShape.circle,
          boxShadow: [
            BoxShadow(
              color: color.withValues(alpha: 0.25 + 0.25 * _pulse.value),
              blurRadius: 24 + 24 * _pulse.value,
              spreadRadius: 4 + 10 * _pulse.value,
            ),
          ],
        ),
        child: child,
      ),
      child: Material(
        shape: const CircleBorder(),
        color: connected ? color : scheme.surfaceContainerHighest,
        child: InkWell(
          customBorder: const CircleBorder(),
          onTap: widget.onPressed,
          child: Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(Icons.power_settings_new, size: widget.size * 0.32, color: connected ? Colors.white : color),
                const SizedBox(height: 4),
                Text(
                  label.toUpperCase(),
                  style: Theme.of(context).textTheme.labelLarge?.copyWith(
                        letterSpacing: 1.2,
                        color: connected ? Colors.white : scheme.onSurface,
                      ),
                ),
                if (_showRing && widget.status == ConnStatus.connecting)
                  Padding(
                    padding: const EdgeInsets.only(top: 2),
                    child: Text(
                      '$percent%',
                      style: Theme.of(context).textTheme.labelMedium?.copyWith(
                            color: color,
                            fontFeatures: const [FontFeature.tabularFigures()],
                          ),
                    ),
                  ),
              ],
            ),
          ),
        ),
      ),
    );

    return Semantics(
      button: true,
      label: '$label. ${statusLabel(widget.status)}${_showRing ? ', $percent percent' : ''}',
      child: SizedBox(
        width: ringSize,
        height: ringSize,
        child: Stack(
          alignment: Alignment.center,
          children: [
            if (_showRing)
              TweenAnimationBuilder<double>(
                tween: Tween(end: percent / 100),
                duration: const Duration(milliseconds: 450),
                curve: Curves.easeOutCubic,
                builder: (context, value, _) => SizedBox(
                  width: ringSize,
                  height: ringSize,
                  child: CircularProgressIndicator(
                    value: value,
                    strokeWidth: 6,
                    strokeCap: StrokeCap.round,
                    color: color,
                    backgroundColor: color.withValues(alpha: 0.15),
                  ),
                ),
              ),
            button,
          ],
        ),
      ),
    );
  }
}
