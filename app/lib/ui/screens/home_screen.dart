import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../engine/engine.dart';
import '../../state/providers.dart';
import '../theme.dart';
import '../widgets/connect_button.dart';
import '../widgets/protocol_badge.dart';
import '../widgets/credentials_dialog.dart';

class HomeScreen extends ConsumerWidget {
  const HomeScreen({super.key, required this.onOpenProfiles});

  final VoidCallback onOpenProfiles;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final conn = ref.watch(connectionProvider);
    final profile = ref.watch(selectedProfileProvider);
    final scheme = Theme.of(context).colorScheme;
    final text = Theme.of(context).textTheme;

    // Tell the user when the core switched profiles on its own, and why.
    ref.listen(connectionProvider.select((s) => s.failover), (prev, next) {
      if (next == null || next.at == prev?.at) return;
      final profiles = ref.read(profilesProvider);
      String name(String id) => profiles.where((p) => p.id == id).firstOrNull?.name ?? id;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        duration: const Duration(seconds: 6),
        content: Text('Switched to ${name(next.to)} — ${name(next.from)} stopped responding'),
      ));
    });

    return SafeArea(
      child: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 420),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                _StatusLine(state: conn),
                const SizedBox(height: 40),
                ConnectButton(
                  status: conn.status,
                  progress: conn.progress,
                  onPressed: profile == null
                      ? null
                      : () async {
                          final conn = ref.read(connectionProvider.notifier);
                          if (!conn.isActive && profile.needsCredentials && !profile.hasCredentials) {
                            if (!await askCredentials(context, ref, profile)) return;
                          }
                          conn.toggle();
                        },
                ),
                const SizedBox(height: 40),
                if (conn.status == ConnStatus.connected) ...[
                  _TrafficRow(up: conn.upBps, down: conn.downBps),
                  const SizedBox(height: 24),
                ],
                if (conn.status == ConnStatus.error && conn.error != null) ...[
                  Text(conn.error!, style: text.bodyMedium?.copyWith(color: scheme.error), textAlign: TextAlign.center),
                  const SizedBox(height: 24),
                ],
                if (profile == null)
                  _EmptyHint(onImport: onOpenProfiles)
                else
                  Card(
                    child: ListTile(
                      leading: const Icon(Icons.public),
                      title: Text(profile.name, overflow: TextOverflow.ellipsis),
                      subtitle: Text(profile.address, overflow: TextOverflow.ellipsis),
                      trailing: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [ProtocolBadge(profile.protocol), const Icon(Icons.chevron_right)],
                      ),
                      onTap: onOpenProfiles,
                    ),
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _StatusLine extends StatefulWidget {
  const _StatusLine({required this.state});

  final EngineState state;

  @override
  State<_StatusLine> createState() => _StatusLineState();
}

class _StatusLineState extends State<_StatusLine> {
  Timer? _ticker;

  @override
  void initState() {
    super.initState();
    _ticker = Timer.periodic(const Duration(seconds: 1), (_) {
      if (widget.state.connectedSince != null) setState(() {});
    });
  }

  @override
  void dispose() {
    _ticker?.cancel();
    super.dispose();
  }

  String _elapsed(DateTime since) {
    final d = DateTime.now().difference(since);
    String two(int n) => n.toString().padLeft(2, '0');
    return '${two(d.inHours)}:${two(d.inMinutes % 60)}:${two(d.inSeconds % 60)}';
  }

  @override
  Widget build(BuildContext context) {
    final s = widget.state;
    final scheme = Theme.of(context).colorScheme;
    final color = statusColor(s.status, scheme);
    return Column(
      children: [
        Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(statusIcon(s.status), color: color),
            const SizedBox(width: 8),
            Text(statusLabel(s.status),
                style: Theme.of(context).textTheme.titleLarge?.copyWith(color: color)),
          ],
        ),
        if (s.status == ConnStatus.connected && s.connectedSince != null)
          Padding(
            padding: const EdgeInsets.only(top: 4),
            child: Text(_elapsed(s.connectedSince!),
                style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                      fontFeatures: const [FontFeature.tabularFigures()],
                    )),
          ),
        // While connecting, say which step is running (and for which profile).
        if (s.status == ConnStatus.connecting && s.progress != null)
          Padding(
            padding: const EdgeInsets.only(top: 4),
            child: AnimatedSwitcher(
              duration: const Duration(milliseconds: 200),
              child: Text(
                [s.progress!.detail ?? stageLabel(s.progress!.stage), ?s.progress!.profile].join(' · '),
                key: ValueKey('${s.progress!.stage}${s.progress!.profile}${s.progress!.detail}'),
                style: Theme.of(context).textTheme.bodyMedium,
                textAlign: TextAlign.center,
              ),
            ),
          ),
      ],
    );
  }
}

class _TrafficRow extends StatelessWidget {
  const _TrafficRow({required this.up, required this.down});

  final int up;
  final int down;

  static String _fmt(int bps) {
    if (bps >= 1 << 20) return '${(bps / (1 << 20)).toStringAsFixed(1)} MB/s';
    if (bps >= 1 << 10) return '${(bps / (1 << 10)).toStringAsFixed(0)} KB/s';
    return '$bps B/s';
  }

  @override
  Widget build(BuildContext context) {
    final style = Theme.of(context).textTheme.bodyLarge;
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        const Icon(Icons.arrow_downward, size: 18),
        Text(' ${_fmt(down)}', style: style),
        const SizedBox(width: 24),
        const Icon(Icons.arrow_upward, size: 18),
        Text(' ${_fmt(up)}', style: style),
      ],
    );
  }
}

class _EmptyHint extends StatelessWidget {
  const _EmptyHint({required this.onImport});

  final VoidCallback onImport;

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Text('No profiles yet', style: Theme.of(context).textTheme.titleMedium),
        const SizedBox(height: 4),
        const Text('Import a config file, link or subscription to get started.', textAlign: TextAlign.center),
        const SizedBox(height: 16),
        FilledButton.icon(onPressed: onImport, icon: const Icon(Icons.add), label: const Text('Import config')),
      ],
    );
  }
}
