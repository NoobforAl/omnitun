import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../l10n/l10n.dart';
import '../../state/providers.dart';
import 'bypass_screen.dart';
import 'logs_screen.dart';
import 'notices_screen.dart';
import 'profiles_screen.dart';
import 'split_tunnel_screen.dart';

class SettingsScreen extends ConsumerWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l = context.l10n;
    final s = ref.watch(settingsProvider);
    final ctrl = ref.read(settingsProvider.notifier);
    final bypassEntries = ref.watch(bypassListsProvider).where((b) => b.enabled).fold<int>(0, (n, b) => n + b.entries);
    return Scaffold(
      appBar: AppBar(title: Text(l.settingsTitle)),
      body: ListView(
        children: [
          _Header(l.settingsServers),
          ListTile(
            leading: const Icon(Icons.dns_outlined),
            title: Text(l.manageServers),
            subtitle: Text(l.manageServersHint),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => const ProfilesScreen())),
          ),
          _Header(l.settingsConnection),
          SwitchListTile(
            title: Text(l.vpnMode),
            subtitle: Text(l.vpnModeHint),
            value: s.tunMode,
            onChanged: (v) => ctrl.update((s) => s.copyWith(tunMode: v)),
          ),
          SwitchListTile(
            title: Text(l.systemProxy),
            subtitle: Text(l.systemProxyHint),
            value: s.systemProxy,
            onChanged: s.tunMode ? null : (v) => ctrl.update((s) => s.copyWith(systemProxy: v)),
          ),
          SwitchListTile(
            title: Text(l.autoSwitch),
            subtitle: Text(l.autoSwitchHint),
            value: s.autoFailover,
            onChanged: (v) => ctrl.update((s) => s.copyWith(autoFailover: v)),
          ),
          SwitchListTile(
            title: Text(l.exitTitle),
            subtitle: Text(l.exitHint),
            value: s.showExit,
            onChanged: (v) => ctrl.update((s) => s.copyWith(showExit: v)),
          ),
          SwitchListTile(
            title: Text(l.bypassLan),
            subtitle: Text(l.bypassLanHint),
            value: s.bypassLan,
            onChanged: (v) => ctrl.update((s) => s.copyWith(bypassLan: v)),
          ),
          if (Platform.isAndroid)
            ListTile(
              title: Text(l.alwaysOnTitle),
              subtitle: Text(l.alwaysOnHint),
              trailing: const Icon(Icons.open_in_new),
              onTap: () => _openVpnSettings(context),
            ),
          ListTile(
            title: Text(l.bypassTitle),
            subtitle: Text(bypassEntries == 0 ? l.bypassListsHint : l.bypassListsInUse(bypassEntries)),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => const BypassScreen())),
          ),
          if (Platform.isAndroid)
            ListTile(
              title: Text(l.splitTunneling),
              subtitle: Text(switch (s.splitMode) {
                'include' => l.splitSummaryOnly(s.splitApps.length),
                'exclude' => l.splitSummaryBypass(s.splitApps.length),
                _ => l.splitSummaryOff,
              }),
              trailing: const Icon(Icons.chevron_right),
              enabled: s.tunMode,
              onTap: () => Navigator.of(context).push(MaterialPageRoute(builder: (_) => const SplitTunnelScreen())),
            ),
          ListTile(
            title: Text(l.remoteDns),
            subtitle: Text(s.remoteDns),
            trailing: const Icon(Icons.edit_outlined),
            onTap: () => _editDns(context, ref, s.remoteDns),
          ),
          _Header(l.settingsApp),
          ListTile(
            title: Text(l.language),
            subtitle: Text(
              s.language.isEmpty
                  ? l.languageSystem
                  : supportedLanguages.where((e) => e.$1 == s.language).map((e) => e.$2).firstOrNull ?? s.language,
            ),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => _pickLanguage(context, ref, s.language),
          ),
          ListTile(
            title: Text(l.appearance),
            subtitle: Text(switch (s.theme) {
              'light' => l.themeLight,
              'system' => l.themeSystem,
              _ => l.themeDark,
            }),
            trailing: SegmentedButton<String>(
              showSelectedIcon: false,
              segments: [
                ButtonSegment(value: 'dark', icon: const Icon(Icons.dark_mode_outlined), tooltip: l.themeDark),
                ButtonSegment(
                  value: 'system',
                  icon: const Icon(Icons.brightness_auto_outlined),
                  tooltip: l.themeSystem,
                ),
                ButtonSegment(value: 'light', icon: const Icon(Icons.light_mode_outlined), tooltip: l.themeLight),
              ],
              selected: {s.theme},
              onSelectionChanged: (v) => ctrl.update((s) => s.copyWith(theme: v.first)),
            ),
          ),
          ListTile(
            leading: const Icon(Icons.receipt_long_outlined),
            title: Text(l.tooltipLogs),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => const LogsScreen())),
          ),
          SwitchListTile(
            title: Text(l.stopOnClose),
            value: s.stopOnClose,
            onChanged: (v) => ctrl.update((s) => s.copyWith(stopOnClose: v)),
          ),
          ListTile(
            leading: const Icon(Icons.gavel_outlined),
            title: Text(l.noticesTitle),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => const NoticesScreen())),
          ),
          ListTile(
            leading: const Icon(Icons.info_outline),
            title: Text(l.appName),
            subtitle: Text(l.aboutLine(ref.watch(engineProvider).name)),
          ),
        ],
      ),
    );
  }
}

Future<void> _openVpnSettings(BuildContext context) async {
  final l = context.l10n;
  final opened = await const MethodChannel('omnitun/core').invokeMethod<bool>('openVpnSettings') ?? false;
  if (!opened && context.mounted) {
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(l.openVpnSettingsFallback)));
  }
}

/// The language of the app: the system's, or one of the translations (each shown in its own language).
Future<void> _pickLanguage(BuildContext context, WidgetRef ref, String current) async {
  final l = context.l10n;
  final picked = await showDialog<String>(
    context: context,
    builder: (context) => SimpleDialog(
      title: Text(l.language),
      children: [
        RadioGroup<String>(
          groupValue: current,
          onChanged: (v) => Navigator.pop(context, v),
          child: Column(
            children: [
              for (final option in [('', l.languageSystem), ...supportedLanguages])
                RadioListTile<String>(value: option.$1, title: Text(option.$2)),
            ],
          ),
        ),
      ],
    ),
  );
  if (picked != null) ref.read(settingsProvider.notifier).update((s) => s.copyWith(language: picked));
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
    final l = context.l10n;
    return AlertDialog(
      title: Text(l.remoteDns),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(l.remoteDnsHelp),
          const SizedBox(height: 12),
          TextField(
            controller: _controller,
            autofocus: true,
            keyboardType: TextInputType.url,
            textDirection: TextDirection.ltr, // an address is left to right in every language
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context, SettingsState.defaultRemoteDns),
          child: Text(l.actionReset),
        ),
        TextButton(onPressed: () => Navigator.pop(context), child: Text(l.actionCancel)),
        FilledButton(onPressed: () => Navigator.pop(context, _controller.text.trim()), child: Text(l.actionSave)),
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
      child: Text(
        text,
        style: Theme.of(context).textTheme.titleSmall?.copyWith(color: Theme.of(context).colorScheme.primary),
      ),
    );
  }
}
