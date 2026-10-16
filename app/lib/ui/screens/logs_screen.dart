import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../l10n/l10n.dart';
import '../../state/providers.dart';

class LogsScreen extends ConsumerWidget {
  const LogsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final logs = ref.watch(logsProvider);
    return Scaffold(
      appBar: AppBar(
        title: Text(context.l10n.tooltipLogs),
        actions: [
          IconButton(
            tooltip: context.l10n.actionCopyAll,
            onPressed: logs.isEmpty ? null : () => Clipboard.setData(ClipboardData(text: logs.join('\n'))),
            icon: const Icon(Icons.copy_all),
          ),
          IconButton(
            tooltip: context.l10n.actionClear,
            onPressed: logs.isEmpty ? null : () => ref.read(logsProvider.notifier).clear(),
            icon: const Icon(Icons.delete_sweep_outlined),
          ),
        ],
      ),
      body: logs.isEmpty
          ? Center(child: Text(context.l10n.logsEmpty))
          : ListView.builder(
              reverse: true,
              padding: const EdgeInsets.all(12),
              itemCount: logs.length,
              itemBuilder: (_, i) => SelectableText(
                logs[logs.length - 1 - i],
                style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
              ),
            ),
    );
  }
}
