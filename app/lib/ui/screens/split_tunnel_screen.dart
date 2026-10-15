import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../state/providers.dart';

class InstalledApp {
  const InstalledApp(this.package, this.label, this.system);

  final String package;
  final String label;
  final bool system;
}

/// Android app list and icons, via the "omnitun/core" channel.
class AndroidApps {
  static const _channel = MethodChannel('omnitun/core');
  static final _icons = <String, Future<Uint8List?>>{};

  static Future<List<InstalledApp>> list() async {
    final raw = await _channel.invokeListMethod<Map<dynamic, dynamic>>('listApps') ?? const [];
    final apps = [
      for (final a in raw) InstalledApp(a['package'] as String, a['label'] as String, a['system'] as bool),
    ]..sort((a, b) => a.label.toLowerCase().compareTo(b.label.toLowerCase()));
    return apps;
  }

  static Future<Uint8List?> icon(String package) =>
      _icons.putIfAbsent(package, () => _channel.invokeMethod<Uint8List>('appIcon', package));
}

/// Per-app split tunneling: choose which apps use (or bypass) the VPN.
class SplitTunnelScreen extends ConsumerStatefulWidget {
  const SplitTunnelScreen({super.key});

  @override
  ConsumerState<SplitTunnelScreen> createState() => _SplitTunnelScreenState();
}

class _SplitTunnelScreenState extends ConsumerState<SplitTunnelScreen> {
  late final Future<List<InstalledApp>> _apps = AndroidApps.list();
  String _query = '';
  bool _showSystem = false;

  void _setMode(String mode) =>
      ref.read(settingsProvider.notifier).update((s) => s.copyWith(splitMode: mode));

  void _toggle(String package, bool on) {
    ref.read(settingsProvider.notifier).update((s) {
      final apps = {...s.splitApps};
      on ? apps.add(package) : apps.remove(package);
      return s.copyWith(splitApps: apps.toList()..sort());
    });
  }

  @override
  Widget build(BuildContext context) {
    final s = ref.watch(settingsProvider);
    final selected = s.splitApps.toSet();
    final theme = Theme.of(context);

    return Scaffold(
      appBar: AppBar(title: const Text('Split tunneling')),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 4),
            child: SegmentedButton<String>(
              segments: const [
                ButtonSegment(value: 'off', label: Text('All apps')),
                ButtonSegment(value: 'include', label: Text('Only selected')),
                ButtonSegment(value: 'exclude', label: Text('Bypass selected')),
              ],
              selected: {s.splitMode},
              onSelectionChanged: (v) => _setMode(v.first),
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
            child: Text(
              switch (s.splitMode) {
                'include' => selected.isEmpty
                    ? 'No apps selected yet — until you pick some, all apps use the VPN.'
                    : 'Only the ${selected.length} selected app(s) use the VPN.',
                'exclude' => 'The ${selected.length} selected app(s) connect directly, without the VPN.',
                _ => 'Every app uses the VPN. Choose a mode to pick apps.',
              },
              style: theme.textTheme.bodySmall,
            ),
          ),
          if (s.splitMode != 'off') ...[
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 4, 16, 0),
              child: TextField(
                decoration: const InputDecoration(
                  prefixIcon: Icon(Icons.search),
                  hintText: 'Search apps',
                  border: OutlineInputBorder(),
                  isDense: true,
                ),
                onChanged: (v) => setState(() => _query = v.trim().toLowerCase()),
              ),
            ),
            SwitchListTile(
              dense: true,
              title: const Text('Show system apps'),
              value: _showSystem,
              onChanged: (v) => setState(() => _showSystem = v),
            ),
            Expanded(
              child: FutureBuilder<List<InstalledApp>>(
                future: _apps,
                builder: (context, snap) {
                  if (snap.hasError) return Center(child: Text('Could not list apps: ${snap.error}'));
                  if (!snap.hasData) return const Center(child: CircularProgressIndicator());
                  final apps = snap.data!.where((a) {
                    if (!_showSystem && a.system && !selected.contains(a.package)) return false;
                    return _query.isEmpty ||
                        a.label.toLowerCase().contains(_query) ||
                        a.package.toLowerCase().contains(_query);
                  }).toList()
                    // Selected apps first.
                    ..sort((a, b) => (selected.contains(b.package) ? 1 : 0) - (selected.contains(a.package) ? 1 : 0));
                  return ListView.builder(
                    itemCount: apps.length,
                    itemBuilder: (context, i) {
                      final a = apps[i];
                      return CheckboxListTile(
                        value: selected.contains(a.package),
                        onChanged: (v) => _toggle(a.package, v ?? false),
                        secondary: _AppIcon(a.package),
                        title: Text(a.label, overflow: TextOverflow.ellipsis),
                        subtitle: Text(a.package, overflow: TextOverflow.ellipsis, style: theme.textTheme.bodySmall),
                      );
                    },
                  );
                },
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _AppIcon extends StatelessWidget {
  const _AppIcon(this.package);

  final String package;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 40,
      height: 40,
      child: FutureBuilder<Uint8List?>(
        future: AndroidApps.icon(package),
        builder: (context, snap) => snap.data == null
            ? const Icon(Icons.android)
            : Image.memory(snap.data!, gaplessPlayback: true),
      ),
    );
  }
}
