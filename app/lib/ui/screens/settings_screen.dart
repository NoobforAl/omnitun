import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../state/providers.dart';

class SettingsScreen extends ConsumerWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final s = ref.watch(settingsProvider);
    final ctrl = ref.read(settingsProvider.notifier);
    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: ListView(
        children: [
          const _Header('Connection'),
          SwitchListTile(
            title: const Text('VPN mode (all traffic)'),
            subtitle: const Text('Off = proxy only: local SOCKS/HTTP port, no VPN permission needed'),
            value: s.tunMode,
            onChanged: (v) => ctrl.update((s) => s.copyWith(tunMode: v)),
          ),
          SwitchListTile(
            title: const Text('Set system proxy'),
            subtitle: const Text('Proxy mode: point the OS proxy settings at Omnitun while connected'),
            value: s.systemProxy,
            onChanged: s.tunMode ? null : (v) => ctrl.update((s) => s.copyWith(systemProxy: v)),
          ),
          SwitchListTile(
            title: const Text('Auto-switch on failure'),
            subtitle: const Text('Try the next profile in the group when one stops working'),
            value: s.autoFailover,
            onChanged: (v) => ctrl.update((s) => s.copyWith(autoFailover: v)),
          ),
          SwitchListTile(
            title: const Text('Kill switch'),
            subtitle: const Text('Block traffic while the VPN is reconnecting'),
            value: s.killSwitch,
            onChanged: (v) => ctrl.update((s) => s.copyWith(killSwitch: v)),
          ),
          const _Header('App'),
          SwitchListTile(
            title: const Text('Stop VPN when app is closed'),
            value: s.stopOnClose,
            onChanged: (v) => ctrl.update((s) => s.copyWith(stopOnClose: v)),
          ),
          ListTile(
            leading: const Icon(Icons.info_outline),
            title: const Text('Omnitun'),
            subtitle: Text('MIT License · engine: ${ref.watch(engineProvider).name}'),
          ),
        ],
      ),
    );
  }
}

class _Header extends StatelessWidget {
  const _Header(this.text);

  final String text;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 20, 16, 4),
      child: Text(text, style: Theme.of(context).textTheme.titleSmall?.copyWith(
            color: Theme.of(context).colorScheme.primary,
          )),
    );
  }
}
