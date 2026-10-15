import 'package:flutter/material.dart';

import '../../engine/engine.dart';
import '../theme.dart';

/// The large central connect toggle. Pulses only while connecting.
class ConnectButton extends StatefulWidget {
  const ConnectButton({super.key, required this.status, required this.onPressed, this.size = 180});

  final ConnStatus status;
  final VoidCallback? onPressed;
  final double size;

  @override
  State<ConnectButton> createState() => _ConnectButtonState();
}

class _ConnectButtonState extends State<ConnectButton> with SingleTickerProviderStateMixin {
  late final AnimationController _pulse =
      AnimationController(vsync: this, duration: const Duration(milliseconds: 1100));

  bool get _busy =>
      widget.status == ConnStatus.connecting || widget.status == ConnStatus.disconnecting;

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

    return Semantics(
      button: true,
      label: '$label. ${statusLabel(widget.status)}',
      child: AnimatedBuilder(
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
                  Icon(Icons.power_settings_new,
                      size: widget.size * 0.36, color: connected ? Colors.white : color),
                  const SizedBox(height: 4),
                  Text(
                    label.toUpperCase(),
                    style: Theme.of(context).textTheme.labelLarge?.copyWith(
                          letterSpacing: 1.2,
                          color: connected ? Colors.white : scheme.onSurface,
                        ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
