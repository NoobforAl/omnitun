import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../engine/engine.dart';
import '../../state/providers.dart';
import '../../state/subscriptions.dart';
import '../widgets/protocol_badge.dart';
import 'location_screen.dart';
import 'profile_editor_screen.dart';
import 'qr_scan_screen.dart';
import 'share_screen.dart';
import 'subscription_screen.dart';
import '../widgets/credentials_dialog.dart';

class ProfilesScreen extends ConsumerWidget {
  const ProfilesScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final profiles = ref.watch(profilesProvider);
    final subs = ref.watch(subscriptionsProvider);
    final selected = ref.watch(selectedProfileIdProvider);
    ref.watch(formsProvider); // loads the editors, so Edit appears in the menus
    final manual = profiles.where((p) => p.group == ProfileInfo.defaultGroup).toList();

    return Scaffold(
      appBar: AppBar(
        title: const Text('Manage servers'),
        actions: [
          IconButton(
            tooltip: 'Add subscription',
            onPressed: () => Navigator.of(context).push(MaterialPageRoute(builder: (_) => const SubscriptionScreen())),
            icon: const Icon(Icons.add_link),
          ),
          IconButton(
            tooltip: 'Test latency',
            onPressed: profiles.isEmpty ? null : () => testAllLatencies(ref),
            icon: const Icon(Icons.speed),
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => showImportSheet(context, ref),
        icon: const Icon(Icons.add),
        label: const Text('Import'),
      ),
      body: profiles.isEmpty && subs.isEmpty
          ? const Center(child: Text('No profiles. Tap Import to add one.'))
          : CustomScrollView(
              slivers: [
                SliverToBoxAdapter(
                  child: Padding(
                    padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
                    child: Text(
                      'Long-press and drag to reorder. With auto-switch on, the rest of the '
                      'group is tried next, in this order.',
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                  ),
                ),
                if (manual.isNotEmpty) ...[
                  const SliverToBoxAdapter(child: _GroupTitle('My profiles')),
                  _GroupList(group: ProfileInfo.defaultGroup, profiles: manual, selectedId: selected),
                ],
                for (final sub in subs) ...[
                  SliverToBoxAdapter(child: _SubscriptionHeader(sub)),
                  _GroupList(
                    group: sub.id,
                    profiles: profiles.where((p) => p.group == sub.id).toList(),
                    selectedId: selected,
                  ),
                ],
                const SliverToBoxAdapter(child: SizedBox(height: 96)),
              ],
            ),
    );
  }
}

class _GroupTitle extends StatelessWidget {
  const _GroupTitle(this.text);

  final String text;

  @override
  Widget build(BuildContext context) => Padding(
        padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
        child: Text(text, style: Theme.of(context).textTheme.titleSmall),
      );
}

/// One group's profiles, reorderable within the group.
class _GroupList extends ConsumerWidget {
  const _GroupList({required this.group, required this.profiles, required this.selectedId});

  final String group;
  final List<ProfileInfo> profiles;
  final String? selectedId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return SliverReorderableList(
      itemCount: profiles.length,
      onReorderItem: (from, to) => ref.read(profilesProvider.notifier).reorderInGroup(group, from, to),
      itemBuilder: (context, i) {
        final p = profiles[i];
        return ReorderableDelayedDragStartListener(
          key: ValueKey(p.id),
          index: i,
          child: Material(
            type: MaterialType.transparency,
            child: ListTile(
              selected: p.id == selectedId,
              leading: Icon(p.id == selectedId ? Icons.radio_button_checked : Icons.radio_button_off),
              title: Text(p.name, overflow: TextOverflow.ellipsis),
              subtitle: Text(
                p.needsCredentials && !p.hasCredentials ? '${p.address} · needs sign-in' : p.address,
                overflow: TextOverflow.ellipsis,
              ),
              trailing: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  LatencyText(p.latencyMs),
                  const SizedBox(width: 8),
                  ProtocolBadge(p.protocol),
                  PopupMenuButton<String>(
                    onSelected: (v) {
                      if (v == 'edit') {
                        Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => ProfileEditorScreen(profile: p)));
                      }
                      if (v == 'share') {
                        Navigator.of(context).push(MaterialPageRoute<void>(builder: (_) => ShareScreen(profile: p)));
                      }
                      if (v == 'delete') ref.read(profilesProvider.notifier).remove(p.id);
                      if (v == 'credentials') askCredentials(context, ref, p);
                    },
                    itemBuilder: (_) => [
                      if (ref.read(formsProvider).value?.any((f) => f.protocol == p.protocol) ?? false)
                        PopupMenuItem(value: 'edit', child: Text(p.protocol == 'tor' ? 'Bridge settings' : 'Edit')),
                      const PopupMenuItem(value: 'share', child: Text('Share link or QR')),
                      if (p.needsCredentials)
                        const PopupMenuItem(value: 'credentials', child: Text('Username and password')),
                      const PopupMenuItem(value: 'delete', child: Text('Delete')),
                    ],
                  ),
                ],
              ),
              onTap: () => ref.read(selectedProfileIdProvider.notifier).select(p.id),
            ),
          ),
        );
      },
    );
  }
}

/// Subscription name, usage, expiry, last update and actions.
class _SubscriptionHeader extends ConsumerWidget {
  const _SubscriptionHeader(this.sub);

  final Subscription sub;

  Future<void> _refresh(BuildContext context, WidgetRef ref) async {
    final messenger = ScaffoldMessenger.of(context);
    try {
      final n = await ref.read(subscriptionsProvider.notifier).refresh(sub.id);
      messenger.showSnackBar(SnackBar(content: Text('${sub.name}: $n profile(s)')));
    } catch (_) {
      // The error is shown in the header.
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    ref.watch(subscriptionsProvider);
    final refreshing = ref.read(subscriptionsProvider.notifier).isRefreshing(sub.id);
    final theme = Theme.of(context);
    final info = sub.info;
    final details = <String>[
      if (info.total > 0) '${formatBytes(info.used)} of ${formatBytes(info.total)}'
      else if (info.used > 0) '${formatBytes(info.used)} used',
      if (info.expire != null) _expiry(info.expire!),
      if (sub.updatedAt != null) 'updated ${_ago(sub.updatedAt!)}',
    ];

    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 4, 4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.cloud_sync_outlined, size: 18, color: theme.colorScheme.primary),
              const SizedBox(width: 8),
              Expanded(child: Text(sub.name, style: theme.textTheme.titleSmall, overflow: TextOverflow.ellipsis)),
              refreshing
                  ? const Padding(
                      padding: EdgeInsets.all(12),
                      child: SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2)),
                    )
                  : IconButton(
                      tooltip: 'Refresh',
                      onPressed: () => _refresh(context, ref),
                      icon: const Icon(Icons.refresh),
                    ),
              PopupMenuButton<String>(
                onSelected: (v) {
                  if (v == 'edit') {
                    Navigator.of(context).push(MaterialPageRoute(builder: (_) => SubscriptionScreen(existing: sub)));
                  } else if (v == 'delete') {
                    ref.read(subscriptionsProvider.notifier).remove(sub.id);
                  }
                },
                itemBuilder: (_) => const [
                  PopupMenuItem(value: 'edit', child: Text('Edit')),
                  PopupMenuItem(value: 'delete', child: Text('Delete subscription and its profiles')),
                ],
              ),
            ],
          ),
          if (info.total > 0)
            Padding(
              padding: const EdgeInsets.only(right: 12, top: 2, bottom: 4),
              child: LinearProgressIndicator(
                value: (info.used / info.total).clamp(0, 1).toDouble(),
                color: info.used >= info.total ? theme.colorScheme.error : null,
              ),
            ),
          if (details.isNotEmpty) Text(details.join(' · '), style: theme.textTheme.bodySmall),
          if (sub.lastError != null)
            Padding(
              padding: const EdgeInsets.only(top: 2, right: 12),
              child: Text('Last refresh failed: ${sub.lastError}',
                  style: theme.textTheme.bodySmall?.copyWith(color: theme.colorScheme.error)),
            ),
        ],
      ),
    );
  }

  static String _expiry(DateTime at) {
    final days = at.difference(DateTime.now()).inDays;
    if (at.isBefore(DateTime.now())) return 'expired';
    if (days == 0) return 'expires today';
    return 'expires in $days day${days == 1 ? '' : 's'}';
  }

  static String _ago(DateTime t) {
    final d = DateTime.now().difference(t);
    if (d.inMinutes < 1) return 'just now';
    if (d.inHours < 1) return '${d.inMinutes} min ago';
    if (d.inDays < 1) return '${d.inHours} h ago';
    return '${d.inDays} d ago';
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

  Future<void> _openFile() async {
    try {
      final file = await FilePicker.pickFile(dialogTitle: 'Open config file');
      if (file == null) return;
      _controller.text = await file.xFile.readAsString();
      await _import();
    } catch (e) {
      if (mounted) setState(() => _message = 'Could not read file: $e');
    }
  }

  Future<void> _scanQr() async {
    final text = await Navigator.of(context).push<String>(
      MaterialPageRoute(builder: (_) => const QrScanScreen()),
    );
    if (text == null || !mounted) return;
    _controller.text = text;
    await _import();
  }

  /// Opens the subscription form (replacing this sheet), optionally prefilled.
  void _openSubscription([String url = '']) {
    final navigator = Navigator.of(context);
    navigator.pop();
    navigator.push(MaterialPageRoute(builder: (_) => SubscriptionScreen(initialUrl: url)));
  }

  Future<void> _import() async {
    final text = _controller.text.trim();
    if (text.isEmpty) return;
    // A single http(s) or nordvpn:// URL is a subscription link, not a config.
    if (!text.contains('\n') && RegExp(r'^(https?|nordvpn)://\S+$').hasMatch(text)) {
      _openSubscription(text);
      return;
    }
    setState(() => _busy = true);
    final result = await ref.read(engineProvider).importText(_controller.text);
    final added = ref.read(profilesProvider.notifier).addAll(result.profiles);
    if (!mounted) return;
    if (result.profiles.isNotEmpty && result.errors.isEmpty) {
      Navigator.of(context).pop();
      final dupes = result.profiles.length - added;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text('Imported $added profile(s)${dupes > 0 ? ', $dupes already present' : ''}'),
      ));
      return;
    }
    setState(() {
      _busy = false;
      _message = result.profiles.isEmpty && result.errors.isNotEmpty
          ? 'Nothing imported: ${result.errors.first}'
          : 'Imported $added, failed ${result.errors.length}';
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
          const Text('Paste share links (vless://, trojan://, ss://, hy2://, anytls://, ssh://, sstp://, ikev2://, …), a Clash or sing-box config, a subscription, '
              'a WireGuard .conf, an OpenVPN .ovpn or Tor bridge lines — or open a file or scan a QR code.'),
          const SizedBox(height: 12),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              if (QrScanScreen.supported)
                FilledButton.tonalIcon(
                  onPressed: _busy ? null : _scanQr,
                  icon: const Icon(Icons.qr_code_scanner),
                  label: const Text('Scan QR'),
                ),
              FilledButton.tonalIcon(
                onPressed: _busy ? null : _openFile,
                icon: const Icon(Icons.file_open_outlined),
                label: const Text('Open file'),
              ),
              FilledButton.tonalIcon(
                onPressed: _busy ? null : _paste,
                icon: const Icon(Icons.content_paste),
                label: const Text('Paste'),
              ),
              FilledButton.tonalIcon(
                onPressed: _busy ? null : _openSubscription,
                icon: const Icon(Icons.add_link),
                label: const Text('Subscription'),
              ),
              PopupMenuButton<String>(
                enabled: !_busy,
                tooltip: 'Add Tor',
                onSelected: (kind) {
                  _controller.text = 'tor://$kind';
                  _import();
                },
                itemBuilder: (_) => const [
                  PopupMenuItem(value: '', child: Text('Tor (direct)')),
                  PopupMenuItem(value: 'snowflake', child: Text('Tor + Snowflake bridge')),
                  PopupMenuItem(value: 'obfs4', child: Text('Tor + obfs4 bridges')),
                ],
                child: IgnorePointer(
                  child: FilledButton.tonalIcon(
                    onPressed: () {},
                    icon: const Icon(Icons.security),
                    label: const Text('Tor'),
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _controller,
            minLines: 4,
            maxLines: 8,
            decoration: const InputDecoration(border: OutlineInputBorder(), hintText: 'vless://…'),
          ),
          if (_message != null) Padding(padding: const EdgeInsets.only(top: 8), child: Text(_message!)),
          const SizedBox(height: 12),
          Align(
            alignment: Alignment.centerRight,
            child: FilledButton(onPressed: _busy ? null : _import, child: const Text('Import')),
          ),
        ],
      ),
    );
  }
}
