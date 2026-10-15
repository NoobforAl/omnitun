import 'package:flutter/material.dart';

import '../engine/engine.dart';

const _seed = Color(0xFF0E7C86);

ThemeData buildTheme(Brightness brightness) {
  final scheme = ColorScheme.fromSeed(seedColor: _seed, brightness: brightness);
  return ThemeData(
    colorScheme: scheme,
    useMaterial3: true,
    visualDensity: VisualDensity.adaptivePlatformDensity,
    cardTheme: const CardThemeData(margin: EdgeInsets.zero),
  );
}

/// Semantic colour for each connection state. Never the only signal:
/// screens always pair it with an icon and a label.
Color statusColor(ConnStatus status, ColorScheme scheme) {
  return switch (status) {
    ConnStatus.connected => const Color(0xFF1FA67A),
    ConnStatus.connecting || ConnStatus.disconnecting => const Color(0xFFE0A100),
    ConnStatus.error => scheme.error,
    ConnStatus.noInternet => scheme.outline,
    ConnStatus.disconnected => scheme.outline,
  };
}

IconData statusIcon(ConnStatus status) {
  return switch (status) {
    ConnStatus.connected => Icons.verified_user,
    ConnStatus.connecting || ConnStatus.disconnecting => Icons.sync,
    ConnStatus.error => Icons.error_outline,
    ConnStatus.noInternet => Icons.wifi_off,
    ConnStatus.disconnected => Icons.shield_outlined,
  };
}

String statusLabel(ConnStatus status) {
  return switch (status) {
    ConnStatus.connected => 'Connected',
    ConnStatus.connecting => 'Connecting…',
    ConnStatus.disconnecting => 'Disconnecting…',
    ConnStatus.error => 'Connection failed',
    ConnStatus.noInternet => 'No internet connection',
    ConnStatus.disconnected => 'Not connected',
  };
}

/// Human label for a connection stage reported by the core (PLAN.md 3.3 #6).
String stageLabel(String stage) {
  return switch (stage) {
    'permission' => 'Waiting for VPN permission',
    'network_check' => 'Checking internet connection',
    'prepare' => 'Preparing configuration',
    'engine_start' => 'Starting engine',
    'tunnel_up' => 'Opening tunnel',
    'handshake' => 'Contacting server',
    'connected' => 'Connected',
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
