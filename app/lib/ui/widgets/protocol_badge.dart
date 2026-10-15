import 'package:flutter/material.dart';

import '../theme.dart';

class ProtocolBadge extends StatelessWidget {
  const ProtocolBadge(this.protocol, {super.key});

  final String protocol;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: scheme.secondaryContainer,
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        protocolBadge(protocol),
        style: Theme.of(context).textTheme.labelSmall?.copyWith(
              color: scheme.onSecondaryContainer,
              fontWeight: FontWeight.w600,
            ),
      ),
    );
  }
}

/// Latency text coloured by quality; shows a dash when untested.
class LatencyText extends StatelessWidget {
  const LatencyText(this.ms, {super.key});

  final int? ms;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final (text, color) = switch (ms) {
      null => ('—', scheme.outline),
      -1 => ('timeout', scheme.error),
      final v when v < 150 => ('$v ms', const Color(0xFF1FA67A)),
      final v when v < 400 => ('$v ms', const Color(0xFFE0A100)),
      final v => ('$v ms', scheme.error),
    };
    return Text(text, style: Theme.of(context).textTheme.labelMedium?.copyWith(color: color));
  }
}
