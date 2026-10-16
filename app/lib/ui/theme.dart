import 'package:flutter/material.dart';

import '../engine/engine.dart';
import '../l10n/app_localizations.dart';

/// A dark-fantasy palette, all in cold, muted blues that are easy on the eyes: a night
/// sky of slate, steel blue for actions, frost for "protected", worn gold for "working".
const brandIndigo = Color(0xFF4A78B8);
const brandViolet = Color(0xFF30578F);

/// Frost-teal glow of the "connected" state. Works on dark and light.
const connectedGreen = Color(0xFF63B8C6);

/// Worn gold of the "connecting" state.
const connectingAmber = Color(0xFFC9A55C);

/// Dried-blood red of the Disconnect button.
const disconnectRed = Color(0xFFC4566B);

const _ink = Color(0xFF080C13);
const _inkPanel = Color(0xFF0F1622);
const _inkHigh = Color(0xFF16202F);
const _inkHighest = Color(0xFF1F2D42);

/// Engraved capitals (Cinzel, SIL OFL) for headings; Inter for text; Vazirmatn for
/// Persian and Arabic, which the other two do not cover.
const headingFont = 'Cinzel';
const bodyFont = 'Inter';
const fallbackFonts = ['Inter', 'Vazirmatn'];

ThemeData buildTheme(Brightness brightness) {
  final dark = brightness == Brightness.dark;
  final scheme = dark
      ? const ColorScheme.dark(
          primary: Color(0xFF86A8D8),
          onPrimary: Color(0xFF0A1424),
          primaryContainer: Color(0xFF1E3555),
          onPrimaryContainer: Color(0xFFD6E4F7),
          secondary: connectedGreen,
          onSecondary: Color(0xFF06222A),
          secondaryContainer: _inkHighest,
          onSecondaryContainer: Color(0xFFDCE6F4),
          tertiary: connectingAmber,
          error: disconnectRed,
          onError: Colors.white,
          surface: _ink,
          onSurface: Color(0xFFDCE3EE),
          onSurfaceVariant: Color(0xFF8D9AB0),
          surfaceContainerLowest: _ink,
          surfaceContainerLow: _inkPanel,
          surfaceContainer: _inkPanel,
          surfaceContainerHigh: _inkHigh,
          surfaceContainerHighest: _inkHighest,
          outline: Color(0xFF4B5B75),
          outlineVariant: Color(0xFF223047),
        )
      // Cold stone by day.
      : const ColorScheme.light(
          primary: Color(0xFF2D5287),
          onPrimary: Colors.white,
          primaryContainer: Color(0xFFD3E0F2),
          onPrimaryContainer: Color(0xFF0E2340),
          secondary: Color(0xFF2E8B98),
          onSecondary: Colors.white,
          secondaryContainer: Color(0xFFD0D9E6),
          onSecondaryContainer: Color(0xFF16233A),
          tertiary: Color(0xFF9A7B33),
          error: Color(0xFFB03A52),
          onError: Colors.white,
          surface: Color(0xFFE6EAF0),
          onSurface: Color(0xFF141B28),
          onSurfaceVariant: Color(0xFF55637C),
          surfaceContainerLowest: Color(0xFFF3F5F8),
          surfaceContainerLow: Color(0xFFF3F5F8),
          surfaceContainer: Color(0xFFF3F5F8),
          surfaceContainerHigh: Color(0xFFDDE3EC),
          surfaceContainerHighest: Color(0xFFD0D9E6),
          outline: Color(0xFF8796AD),
          outlineVariant: Color(0xFFCBD3DF),
        );

  const radius = 18.0;
  final shape = RoundedRectangleBorder(borderRadius: BorderRadius.circular(radius));
  final base = ThemeData(
    brightness: brightness,
    useMaterial3: true,
    colorScheme: scheme,
    fontFamily: bodyFont,
    fontFamilyFallback: fallbackFonts,
  );
  final text = base.textTheme.copyWith(
    headlineSmall: base.textTheme.headlineSmall?.copyWith(
      fontFamily: headingFont,
      fontFamilyFallback: fallbackFonts,
      fontWeight: FontWeight.w700,
      letterSpacing: 0.4,
    ),
    titleLarge: base.textTheme.titleLarge?.copyWith(
      fontFamily: headingFont,
      fontFamilyFallback: fallbackFonts,
      fontWeight: FontWeight.w700,
      letterSpacing: 0.6,
    ),
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
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(radius),
        side: BorderSide(color: scheme.outlineVariant),
      ),
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
    // Darker in the light theme, where the pale glow colours would not read.
    ConnStatus.connected => scheme.brightness == Brightness.dark ? connectedGreen : scheme.secondary,
    ConnStatus.connecting || ConnStatus.disconnecting =>
      scheme.brightness == Brightness.dark ? connectingAmber : scheme.tertiary,
    ConnStatus.error => scheme.error,
    ConnStatus.noInternet => scheme.outline,
    ConnStatus.disconnected => scheme.outline,
  };
}

IconData statusIcon(ConnStatus status) {
  return switch (status) {
    ConnStatus.connected => Icons.auto_awesome_rounded,
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
