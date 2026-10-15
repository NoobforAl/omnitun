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
          ListTile(
            title: const Text('Remote DNS'),
            subtitle: Text(s.remoteDns),
            trailing: const Icon(Icons.edit_outlined),
            onTap: () => _editDns(context, ref, s.remoteDns),
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

Future<void> _editDns(BuildContext context, WidgetRef ref, String current) async {
  final value = await showDialog<String>(context: context, builder: (_) => _DnsDialog(current));
  if (value != null && value.isNotEmpty) {
    ref.read(settingsProvider.notifier).update((s) => s.copyWith(remoteDns: value));
  }
}

/// Owns its controller so it is disposed only after the dialog leaves the tree.
class _DnsDialog extends StatefulWidget {
  const _DnsDialog(this.initial);

  final String initial;

  @override
  State<_DnsDialog> createState() => _DnsDialogState();
}

class _DnsDialogState extends State<_DnsDialog> {
  late final _controller = TextEditingController(text: widget.initial);

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Remote DNS'),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text('Resolver used through the tunnel. Examples: https://1.1.1.1/dns-query, '
              'tls://1.1.1.1, tcp://8.8.8.8, 9.9.9.9'),
          const SizedBox(height: 12),
          TextField(controller: _controller, autofocus: true, keyboardType: TextInputType.url),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context, SettingsState.defaultRemoteDns),
          child: const Text('Reset'),
        ),
        TextButton(onPressed: () => Navigator.pop(context), child: const Text('Cancel')),
        FilledButton(onPressed: () => Navigator.pop(context, _controller.text.trim()), child: const Text('Save')),
      ],
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
