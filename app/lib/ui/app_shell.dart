import 'package:flutter/material.dart';

import 'screens/home_screen.dart';
import 'screens/logs_screen.dart';
import 'screens/profiles_screen.dart';
import 'screens/settings_screen.dart';

/// Breakpoint where the bottom bar becomes a side rail.
const _railBreakpoint = 840.0;

class AppShell extends StatefulWidget {
  const AppShell({super.key});

  @override
  State<AppShell> createState() => _AppShellState();
}

class _AppShellState extends State<AppShell> {
  int _index = 0;

  static const _destinations = [
    (icon: Icons.shield_outlined, selected: Icons.shield, label: 'Home'),
    (icon: Icons.dns_outlined, selected: Icons.dns, label: 'Profiles'),
    (icon: Icons.receipt_long_outlined, selected: Icons.receipt_long, label: 'Logs'),
    (icon: Icons.settings_outlined, selected: Icons.settings, label: 'Settings'),
  ];

  Widget _page() => switch (_index) {
        0 => HomeScreen(onOpenProfiles: () => setState(() => _index = 1)),
        1 => const ProfilesScreen(),
        2 => const LogsScreen(),
        _ => const SettingsScreen(),
      };

  @override
  Widget build(BuildContext context) {
    final wide = MediaQuery.sizeOf(context).width >= _railBreakpoint;
    if (wide) {
      return Scaffold(
        body: Row(
          children: [
            NavigationRail(
              selectedIndex: _index,
              onDestinationSelected: (i) => setState(() => _index = i),
              labelType: NavigationRailLabelType.all,
              destinations: [
                for (final d in _destinations)
                  NavigationRailDestination(
                    icon: Icon(d.icon),
                    selectedIcon: Icon(d.selected),
                    label: Text(d.label),
                  ),
              ],
            ),
            const VerticalDivider(width: 1),
            Expanded(child: _page()),
          ],
        ),
      );
    }
    return Scaffold(
      body: _page(),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _index,
        onDestinationSelected: (i) => setState(() => _index = i),
        destinations: [
          for (final d in _destinations)
            NavigationDestination(icon: Icon(d.icon), selectedIcon: Icon(d.selected), label: d.label),
        ],
      ),
    );
  }
}
