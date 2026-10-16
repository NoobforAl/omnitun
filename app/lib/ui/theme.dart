import 'package:flutter/material.dart';

import '../engine/engine.dart';
import '../l10n/app_localizations.dart';

const _seed = Color(0xFF0E7C86);

/// Calm dark blue like a night sky; the light theme is plain.
const _darkBackground = Color(0xFF10243A);
const _darkPanel = Color(0xFF1A3452);

/// Green of the "connected" state and of the Connect button.
const connectedGreen = Color(0xFF3DB36B);

/// Red of the Disconnect button.
const disconnectRed = Color(0xFFD64550);

ThemeData buildTheme(Brightness brightness) {
  var scheme = ColorScheme.fromSeed(seedColor: _seed, brightness: brightness);
  if (brightness == Brightness.dark) {
    scheme = scheme.copyWith(
      surface: _darkBackground,
      surfaceContainerLowest: _darkBackground,
      surfaceContainerLow: _darkPanel,
      surfaceContainer: _darkPanel,
      surfaceContainerHigh: const Color(0xFF22416A),
      surfaceContainerHighest: const Color(0xFF2A4C78),
    );
  }
  return ThemeData(
    colorScheme: scheme,
    useMaterial3: true,
    visualDensity: VisualDensity.adaptivePlatformDensity,
    scaffoldBackgroundColor: scheme.surface,
    cardTheme: const CardThemeData(margin: EdgeInsets.zero),
    appBarTheme: AppBarTheme(
      backgroundColor: scheme.surface,
      surfaceTintColor: Colors.transparent,
      scrolledUnderElevation: 0,
    ),
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
    ConnStatus.connecting || ConnStatus.disconnecting => const Color(0xFFE0A100),
    ConnStatus.error => scheme.error,
    ConnStatus.noInternet => scheme.outline,
    ConnStatus.disconnected => scheme.outline,
  };
}

IconData statusIcon(ConnStatus status) {
  return switch (status) {
    ConnStatus.connected => Icons.lock,
    ConnStatus.connecting || ConnStatus.disconnecting => Icons.sync,
    ConnStatus.error => Icons.error_outline,
    ConnStatus.noInternet => Icons.wifi_off,
    ConnStatus.disconnected => Icons.lock_open,
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
