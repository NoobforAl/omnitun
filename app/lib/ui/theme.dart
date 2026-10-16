import 'package:flutter/material.dart';

import '../engine/engine.dart';
import '../l10n/app_localizations.dart';

/// Brand colours: indigo for actions, mint for "protected", on an ink-blue base.
const brandIndigo = Color(0xFF6C7BFF);
const brandViolet = Color(0xFF9B6CFF);

/// Green of the "connected" state and of the Connect button. Works on dark and light.
const connectedGreen = Color(0xFF1FCB91);

/// Amber of the "connecting" state.
const connectingAmber = Color(0xFFFFB547);

/// Red of the Disconnect button.
const disconnectRed = Color(0xFFF2536B);

const _ink = Color(0xFF0A0F1E);
const _inkPanel = Color(0xFF131B31);
const _inkHigh = Color(0xFF1B2644);
const _inkHighest = Color(0xFF26345A);

ThemeData buildTheme(Brightness brightness) {
  final dark = brightness == Brightness.dark;
  final scheme = dark
      ? const ColorScheme.dark(
          primary: brandIndigo,
          onPrimary: Colors.white,
          primaryContainer: Color(0xFF2A3470),
          onPrimaryContainer: Color(0xFFDDE1FF),
          secondary: connectedGreen,
          onSecondary: Color(0xFF00281B),
          secondaryContainer: _inkHighest,
          onSecondaryContainer: Color(0xFFE3E8FF),
          tertiary: brandViolet,
          error: disconnectRed,
          onError: Colors.white,
          surface: _ink,
          onSurface: Color(0xFFEAEEFB),
          onSurfaceVariant: Color(0xFF98A3C4),
          surfaceContainerLowest: _ink,
          surfaceContainerLow: _inkPanel,
          surfaceContainer: _inkPanel,
          surfaceContainerHigh: _inkHigh,
          surfaceContainerHighest: _inkHighest,
          outline: Color(0xFF55618A),
          outlineVariant: Color(0xFF2B3759),
        )
      : const ColorScheme.light(
          primary: Color(0xFF4A57E0),
          onPrimary: Colors.white,
          primaryContainer: Color(0xFFE1E4FF),
          onPrimaryContainer: Color(0xFF14196B),
          secondary: Color(0xFF0FA775),
          onSecondary: Colors.white,
          secondaryContainer: Color(0xFFE4E8F5),
          onSecondaryContainer: Color(0xFF1B2447),
          tertiary: Color(0xFF7A4FE0),
          error: Color(0xFFD93A53),
          onError: Colors.white,
          surface: Color(0xFFF4F6FC),
          onSurface: Color(0xFF111833),
          onSurfaceVariant: Color(0xFF58628A),
          surfaceContainerLowest: Colors.white,
          surfaceContainerLow: Colors.white,
          surfaceContainer: Colors.white,
          surfaceContainerHigh: Color(0xFFEBEEF9),
          surfaceContainerHighest: Color(0xFFE0E5F5),
          outline: Color(0xFF8A94B8),
          outlineVariant: Color(0xFFD5DAEC),
        );

  const radius = 16.0;
  final shape = RoundedRectangleBorder(borderRadius: BorderRadius.circular(radius));
  final base = ThemeData(brightness: brightness, useMaterial3: true, colorScheme: scheme);
  final text = base.textTheme.copyWith(
    headlineSmall: base.textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.w700),
    titleLarge: base.textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w700, letterSpacing: -0.2),
    titleMedium: base.textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w600),
    titleSmall: base.textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w700, letterSpacing: 0.6),
  );

  return base.copyWith(
    textTheme: text,
    visualDensity: VisualDensity.adaptivePlatformDensity,
    scaffoldBackgroundColor: scheme.surface,
    splashFactory: InkSparkle.splashFactory,
    cardTheme: CardThemeData(
      margin: EdgeInsets.zero,
      elevation: 0,
      color: scheme.surfaceContainer,
      shape: shape,
    ),
    appBarTheme: AppBarTheme(
      backgroundColor: scheme.surface,
      surfaceTintColor: Colors.transparent,
      scrolledUnderElevation: 0,
      titleTextStyle: text.titleLarge?.copyWith(color: scheme.onSurface),
    ),
    listTileTheme: ListTileThemeData(
      iconColor: scheme.onSurfaceVariant,
      subtitleTextStyle: text.bodySmall?.copyWith(color: scheme.onSurfaceVariant),
      contentPadding: const EdgeInsets.symmetric(horizontal: 20),
    ),
    dividerTheme: DividerThemeData(color: scheme.outlineVariant, space: 1, thickness: 1),
    switchTheme: SwitchThemeData(
      thumbColor: WidgetStateProperty.resolveWith(
        (s) => s.contains(WidgetState.selected) ? Colors.white : scheme.outline,
      ),
      trackColor: WidgetStateProperty.resolveWith(
        (s) => s.contains(WidgetState.selected) ? scheme.primary : scheme.surfaceContainerHighest,
      ),
      trackOutlineColor: WidgetStateProperty.all(Colors.transparent),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        minimumSize: const Size(64, 52),
        shape: shape,
        textStyle: text.titleMedium,
      ),
    ),
    textButtonTheme: TextButtonThemeData(style: TextButton.styleFrom(shape: shape)),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: scheme.surfaceContainerHigh,
      border: OutlineInputBorder(borderRadius: BorderRadius.circular(14), borderSide: BorderSide.none),
      enabledBorder: OutlineInputBorder(borderRadius: BorderRadius.circular(14), borderSide: BorderSide.none),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(14),
        borderSide: BorderSide(color: scheme.primary, width: 1.6),
      ),
    ),
    dialogTheme: DialogThemeData(
      backgroundColor: scheme.surfaceContainerHigh,
      surfaceTintColor: Colors.transparent,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(24)),
    ),
    bottomSheetTheme: BottomSheetThemeData(
      backgroundColor: scheme.surfaceContainerHigh,
      surfaceTintColor: Colors.transparent,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(28))),
    ),
    popupMenuTheme: PopupMenuThemeData(
      color: scheme.surfaceContainerHighest,
      surfaceTintColor: Colors.transparent,
      shape: shape,
    ),
    snackBarTheme: SnackBarThemeData(
      behavior: SnackBarBehavior.floating,
      backgroundColor: scheme.surfaceContainerHighest,
      contentTextStyle: text.bodyMedium?.copyWith(color: scheme.onSurface),
      shape: shape,
    ),
    segmentedButtonTheme: SegmentedButtonThemeData(
      style: SegmentedButton.styleFrom(
        side: BorderSide(color: scheme.outlineVariant),
        selectedBackgroundColor: scheme.primaryContainer,
        selectedForegroundColor: scheme.onPrimaryContainer,
      ),
    ),
    progressIndicatorTheme: ProgressIndicatorThemeData(color: scheme.primary),
  );
}

ThemeMode themeModeOf(String setting) => switch (setting) {
  'light' => ThemeMode.light,
  'system' => ThemeMode.system,
  _ => ThemeMode.dark,
};

/// Semantic colour for each connection state. Never the only signal:
/// screens always pair it with an icon and a label.
Color statusColor(ConnStatus status, ColorScheme scheme) {
  return switch (status) {
    ConnStatus.connected => connectedGreen,
    ConnStatus.connecting || ConnStatus.disconnecting => connectingAmber,
    ConnStatus.error => scheme.error,
    ConnStatus.noInternet => scheme.outline,
    ConnStatus.disconnected => scheme.outline,
  };
}

IconData statusIcon(ConnStatus status) {
  return switch (status) {
    ConnStatus.connected => Icons.shield_rounded,
    ConnStatus.connecting || ConnStatus.disconnecting => Icons.sync_rounded,
    ConnStatus.error => Icons.priority_high_rounded,
    ConnStatus.noInternet => Icons.wifi_off_rounded,
    ConnStatus.disconnected => Icons.power_settings_new_rounded,
  };
}

String statusLabel(AppLocalizations l, ConnStatus status) {
  return switch (status) {
    ConnStatus.connected => l.statusConnected,
    ConnStatus.connecting => l.statusConnecting,
    ConnStatus.disconnecting => l.statusDisconnecting,
    ConnStatus.error => l.statusError,
    ConnStatus.noInternet => l.statusNoInternet,
    ConnStatus.disconnected => l.statusDisconnected,
  };
}

/// Human label for a connection stage reported by the core (PLAN.md 3.3 #6).
String stageLabel(AppLocalizations l, String stage) {
  return switch (stage) {
    'permission' => l.stagePermission,
    'network_check' => l.stageNetworkCheck,
    'prepare' => l.stagePrepare,
    'engine_start' => l.stageEngineStart,
    'tunnel_up' => l.stageTunnelUp,
    'handshake' => l.stageHandshake,
    'connected' => l.stageConnected,
    _ => stage,
  };
}

/// Short protocol badge labels, kept fixed so users learn them.
String protocolBadge(String protocol) {
  return switch (protocol) {
    'wireguard' => 'WG',
    'amneziawg' => 'AWG',
    'vless' => 'VLESS',
    'vmess' => 'VMESS',
    'trojan' => 'TRJ',
    'shadowsocks' => 'SS',
    'hysteria2' => 'HY2',
    'tuic' => 'TUIC',
    'hysteria' => 'HY1',
    'anytls' => 'ANYTLS',
    'ssh' => 'SSH',
    'openvpn' => 'OVPN',
    'ikev2' => 'IKEv2',
    'sstp' => 'SSTP',
    'tor' => 'TOR',
    _ => protocol.toUpperCase(),
  };
}
