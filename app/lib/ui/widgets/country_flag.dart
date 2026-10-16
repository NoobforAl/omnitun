import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

import '../../l10n/l10n.dart';
import '../../state/countries.dart';
import '../theme.dart';

/// A country's flag. Windows has no flag emoji, so there (and for an unknown
/// country) a small round badge is drawn instead: the country code, or a globe.
class CountryFlag extends StatelessWidget {
  const CountryFlag(this.code, {super.key, this.size = 32, this.icon = Icons.public, this.label});

  /// ISO code, or null when the country is unknown.
  final String? code;
  final double size;

  /// Shown instead of a flag when [code] is null.
  final IconData icon;

  /// What a screen reader says; defaults to the country's name.
  final String? label;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final c = code;
    final emoji = c != null && defaultTargetPlatform != TargetPlatform.windows;
    return Semantics(
      label: label ?? (c == null ? context.l10n.unknownCountry : context.countryName(c)),
      excludeSemantics: true,
      child: Container(
        width: size,
        height: size,
        alignment: Alignment.center,
        decoration: BoxDecoration(shape: BoxShape.circle, color: scheme.surfaceContainerHighest),
        child: emoji
            ? Text(flagEmoji(c), style: TextStyle(fontSize: size * 0.62, height: 1))
            : c == null
            ? Icon(icon, size: size * 0.6, color: scheme.onSurfaceVariant)
            : Text(
                c,
                style: TextStyle(
                  fontSize: size * 0.36,
                  fontWeight: FontWeight.w700,
                  color: scheme.onSurfaceVariant,
                  height: 1,
                ),
              ),
      ),
    );
  }
}

/// Dot showing how fast a server answered: green, amber, red, or grey when untested.
class LatencyDot extends StatelessWidget {
  const LatencyDot(this.ms, {super.key, this.size = 10});

  final int? ms;
  final double size;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final color = switch (ms) {
      null => scheme.outlineVariant,
      < 0 => scheme.error,
      < 150 => connectedGreen,
      < 400 => connectingAmber,
      _ => const Color(0xFFE5732F),
    };
    return Tooltip(
      message: switch (ms) {
        null => context.l10n.latencyNotTested,
        < 0 => context.l10n.latencyUnreachable,
        final v => context.l10n.latencyMs(v),
      },
      child: Container(
        width: size,
        height: size,
        decoration: BoxDecoration(color: color, shape: BoxShape.circle),
      ),
    );
  }
}
