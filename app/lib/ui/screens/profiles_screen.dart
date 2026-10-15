import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../engine/engine.dart';
import '../../state/providers.dart';
import '../widgets/protocol_badge.dart';

class ProfilesScreen extends ConsumerWidget {
  const ProfilesScreen({super.key});

  Future<void> _testAll(WidgetRef ref) async {
    final engine = ref.read(engineProvider);
    final profiles = ref.read(profilesProvider);
    await Future.wait(profiles.map((p) async {
      final ms = await engine.testLatency(p);
      ref.read(profilesProvider.notifier).setLatency(p.id, ms);
    }));
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final profiles = ref.watch(profilesProvider);
    final selected = ref.watch(selectedProfileProvider);
    final groups = <String, List<ProfileInfo>>{};
    for (final p in profiles) {
      groups.putIfAbsent(p.group, () => []).add(p);
    }

    return Scaffold(
      appBar: AppBar(
        title: const Text('Profiles'),
        actions: [
          IconButton(
            tooltip: 'Test latency',
            onPressed: profiles.isEmpty ? null : () => _testAll(ref),
            icon: const Icon(Icons.speed),
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => showImportSheet(context, ref),
        icon: const Icon(Icons.add),
        label: const Text('Import'),
      ),
      body: profiles.isEmpty
          ? const Center(child: Text('No profiles. Tap Import to add one.'))
          : ListView(
              padding: const EdgeInsets.only(bottom: 96),
              children: [
                for (final entry in groups.entries) ...[
                  Padding(
                    padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
                    child: Text(entry.key, style: Theme.of(context).textTheme.titleSmall),
                  ),
                  for (final p in entry.value)
                    ListTile(
                      selected: p.id == selected?.id,
                      leading: Icon(p.id == selected?.id ? Icons.radio_button_checked : Icons.radio_button_off),
                      title: Text(p.name, overflow: TextOverflow.ellipsis),
                      subtitle: Text('${p.server}:${p.port}', overflow: TextOverflow.ellipsis),
                      trailing: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          LatencyText(p.latencyMs),
                          const SizedBox(width: 8),
                          ProtocolBadge(p.protocol),
                          PopupMenuButton<String>(
                            onSelected: (v) {
                              if (v == 'delete') ref.read(profilesProvider.notifier).remove(p.id);
                            },
                            itemBuilder: (_) => const [
                              PopupMenuItem(value: 'delete', child: Text('Delete')),
                            ],
                          ),
                        ],
                      ),
                      onTap: () => ref.read(selectedProfileIdProvider.notifier).select(p.id),
                    ),
                ],
              ],
            ),
    );
  }
}

Future<void> showImportSheet(BuildContext context, WidgetRef ref) {
  return showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    showDragHandle: true,
    builder: (context) => const _ImportSheet(),
  );
}

class _ImportSheet extends ConsumerStatefulWidget {
  const _ImportSheet();

  @override
  ConsumerState<_ImportSheet> createState() => _ImportSheetState();
}

class _ImportSheetState extends ConsumerState<_ImportSheet> {
  final _controller = TextEditingController();
  String? _message;
  bool _busy = false;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  Future<void> _paste() async {
    final data = await Clipboard.getData(Clipboard.kTextPlain);
    if (data?.text != null) _controller.text = data!.text!;
  }

  Future<void> _import() async {
    setState(() => _busy = true);
    final result = await ref.read(engineProvider).importText(_controller.text);
    final added = ref.read(profilesProvider.notifier).addAll(result.profiles);
    if (!mounted) return;
    if (added > 0 && result.errors.isEmpty) {
      Navigator.of(context).pop();
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Imported $added profile(s)')));
      return;
    }
    setState(() {
      _busy = false;
      _message = 'Imported $added, failed ${result.errors.length}';
    });
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: EdgeInsets.fromLTRB(16, 0, 16, 16 + MediaQuery.viewInsetsOf(context).bottom),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text('Import config', style: Theme.of(context).textTheme.titleLarge),
          const SizedBox(height: 4),
          const Text('Paste share links (vless://, trojan://, ss://, hy2://, …), a subscription, or a WireGuard .conf.'),
          const SizedBox(height: 12),
          TextField(
            controller: _controller,
            minLines: 4,
            maxLines: 8,
            decoration: const InputDecoration(border: OutlineInputBorder(), hintText: 'vless://…'),
          ),
          if (_message != null) Padding(padding: const EdgeInsets.only(top: 8), child: Text(_message!)),
          const SizedBox(height: 12),
          Row(
            children: [
              TextButton.icon(onPressed: _paste, icon: const Icon(Icons.content_paste), label: const Text('Paste')),
              const Spacer(),
              FilledButton(onPressed: _busy ? null : _import, child: const Text('Import')),
            ],
          ),
        ],
      ),
    );
  }
}
