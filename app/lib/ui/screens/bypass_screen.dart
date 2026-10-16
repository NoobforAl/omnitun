import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../engine/engine.dart';
import '../../l10n/l10n.dart';
import '../../state/providers.dart';

/// Lists of domains, addresses and ranges that go around the VPN (or, in
/// "only these" mode, the only ones that use it).
class BypassScreen extends ConsumerWidget {
  const BypassScreen({super.key});

  void _edit(BuildContext context, {BypassList? list, String text = '', String name = ''}) {
    Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => BypassEditScreen(existing: list, initialText: text, initialName: name),
      ),
    );
  }

  Future<void> _fromClipboard(BuildContext context) async {
    final text = (await Clipboard.getData(Clipboard.kTextPlain))?.text ?? '';
    if (!context.mounted) return;
    if (text.trim().isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(context.l10n.bypassClipboardEmpty)));
      return;
    }
    _edit(context, text: text, name: context.l10n.bypassPastedName);
  }

  Future<void> _fromFile(BuildContext context) async {
    try {
      final file = await FilePicker.pickFile(dialogTitle: context.l10n.bypassPickFile);
      if (file == null || !context.mounted) return;
      final text = await file.xFile.readAsString();
      if (!context.mounted) return;
      _edit(context, text: text, name: file.name.replaceFirst(RegExp(r'\.[A-Za-z0-9]+$'), ''));
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(context.l10n.couldNotReadFile('$e'))));
      }
    }
  }

  Future<void> _fromUrl(BuildContext context) async {
    final url = await showDialog<String>(context: context, builder: (_) => const _UrlDialog());
    if (url == null || !context.mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    final l = context.l10n;
    try {
      final text = await downloadList(l, url);
      if (!context.mounted) return;
      _edit(context, text: text, name: Uri.parse(url).host);
    } catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(l.bypassDownloadFailed('$e'))));
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final lists = ref.watch(bypassListsProvider);
    final settings = ref.watch(settingsProvider);
    final scheme = Theme.of(context).colorScheme;
    final enabled = lists.where((l) => l.enabled).fold<int>(0, (n, l) => n + l.entries);
    return Scaffold(
      appBar: AppBar(
        title: Text(context.l10n.bypassTitle),
        actions: [
          PopupMenuButton<String>(
            tooltip: context.l10n.bypassAddTooltip,
            icon: const Icon(Icons.add),
            onSelected: (v) {
              switch (v) {
                case 'paste':
                  _fromClipboard(context);
                case 'file':
                  _fromFile(context);
                case 'url':
                  _fromUrl(context);
                case 'type':
                  _edit(context, name: context.l10n.bypassDefaultName);
              }
            },
            itemBuilder: (_) => [
              PopupMenuItem(value: 'paste', child: Text(context.l10n.bypassFromClipboard)),
              PopupMenuItem(value: 'file', child: Text(context.l10n.bypassFromFile)),
              PopupMenuItem(value: 'url', child: Text(context.l10n.bypassFromUrl)),
              PopupMenuItem(value: 'type', child: Text(context.l10n.bypassType)),
            ],
          ),
        ],
      ),
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 560),
          child: ListView(
            padding: const EdgeInsets.only(bottom: 32),
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 4, 16, 12),
                child: Text(context.l10n.bypassIntro),
              ),
              if (lists.isEmpty)
                Padding(
                  padding: const EdgeInsets.all(24),
                  child: Column(
                    children: [
                      Text(context.l10n.bypassNoLists),
                      const SizedBox(height: 12),
                      FilledButton.icon(
                        onPressed: () => _fromFile(context),
                        icon: const Icon(Icons.file_open_outlined),
                        label: Text(context.l10n.bypassFromFile),
                      ),
                      TextButton(
                        onPressed: () => _edit(context, name: context.l10n.bypassDefaultName),
                        child: Text(context.l10n.bypassTypeList),
                      ),
                    ],
                  ),
                ),
              for (final l in lists)
                SwitchListTile(
                  title: Text(l.name, overflow: TextOverflow.ellipsis),
                  subtitle: Text(context.l10n.entriesCount(l.entries)),
                  value: l.enabled,
                  secondary: PopupMenuButton<String>(
                    tooltip: context.l10n.tooltipMore,
                    onSelected: (v) {
                      if (v == 'edit') _edit(context, list: l);
                      if (v == 'delete') ref.read(bypassListsProvider.notifier).remove(l.id);
                    },
                    itemBuilder: (_) => [
                      PopupMenuItem(value: 'edit', child: Text(context.l10n.actionEdit)),
                      PopupMenuItem(value: 'delete', child: Text(context.l10n.actionDelete)),
                    ],
                  ),
                  onChanged: (v) => ref.read(bypassListsProvider.notifier).update(l.copyWith(enabled: v)),
                ),
              const Divider(height: 32),
              SwitchListTile(
                title: Text(context.l10n.bypassOnlyTitle),
                subtitle: Text(
                  settings.bypassOnly ? context.l10n.bypassOnlyOn : context.l10n.bypassOnlyOff,
                ),
                value: settings.bypassOnly,
                onChanged: (v) => ref.read(settingsProvider.notifier).update((s) => s.copyWith(bypassOnly: v)),
              ),
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 8, 16, 0),
                child: Text(
                  enabled == 0 ? context.l10n.bypassNothing : context.l10n.bypassInUse(enabled),
                  style: Theme.of(context).textTheme.bodySmall?.copyWith(color: scheme.onSurfaceVariant),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// Downloads a list from a link (plain text, up to 5 MB).
Future<String> downloadList(AppLocalizations l, String url) async {
  final uri = Uri.parse(url.trim());
  if (!(uri.isScheme('http') || uri.isScheme('https')) || uri.host.isEmpty) {
    throw l.bypassHttpLink;
  }
  final client = HttpClient()..connectionTimeout = const Duration(seconds: 15);
  try {
    final request = await client.getUrl(uri).timeout(const Duration(seconds: 20));
    final response = await request.close().timeout(const Duration(seconds: 20));
    if (response.statusCode != 200) throw l.bypassServerAnswered(response.statusCode);
    final bytes = <int>[];
    await for (final chunk in response) {
      bytes.addAll(chunk);
      if (bytes.length > 5 << 20) throw l.bypassTooLarge;
    }
    return utf8.decode(bytes, allowMalformed: true);
  } finally {
    client.close(force: true);
  }
}

class _UrlDialog extends StatefulWidget {
  const _UrlDialog();

  @override
  State<_UrlDialog> createState() => _UrlDialogState();
}

class _UrlDialogState extends State<_UrlDialog> {
  final _controller = TextEditingController();

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text(context.l10n.bypassDownloadTitle),
      content: TextField(
        controller: _controller,
        autofocus: true,
        keyboardType: TextInputType.url,
        decoration: const InputDecoration(hintText: 'https://example.com/bypass.txt'),
      ),
      actions: [
        TextButton(onPressed: () => Navigator.pop(context), child: Text(context.l10n.actionCancel)),
        FilledButton(onPressed: () => Navigator.pop(context, _controller.text.trim()), child: Text(context.l10n.actionDownload)),
      ],
    );
  }
}

/// Edits one list. The core checks the text as it is typed: how many entries it
/// holds and which lines cannot be used.
class BypassEditScreen extends ConsumerStatefulWidget {
  const BypassEditScreen({super.key, this.existing, this.initialText = '', this.initialName = ''});

  final BypassList? existing;
  final String initialText;
  final String initialName;

  @override
  ConsumerState<BypassEditScreen> createState() => _BypassEditScreenState();
}

class _BypassEditScreenState extends ConsumerState<BypassEditScreen> {
  late final _name = TextEditingController(text: widget.existing?.name ?? widget.initialName);
  late final _text = TextEditingController(text: widget.existing?.text ?? widget.initialText);
  BypassReport? _report;
  String? _checkError;
  Timer? _debounce;
  int _check = 0;

  @override
  void initState() {
    super.initState();
    _checkNow();
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _name.dispose();
    _text.dispose();
    super.dispose();
  }

  void _changed() {
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 400), _checkNow);
  }

  Future<void> _checkNow() async {
    final n = ++_check;
    try {
      final report = await ref.read(engineProvider).parseBypass(_text.text);
      if (mounted && n == _check) {
        setState(() {
          _report = report;
          _checkError = null;
        });
      }
    } catch (e) {
      if (mounted && n == _check) setState(() => _checkError = e.toString());
    }
  }

  Future<void> _save() async {
    final defaultName = context.l10n.bypassDefaultName;
    await _checkNow();
    final report = _report;
    if (report == null || report.entries == 0) return;
    final name = _name.text.trim().isEmpty ? defaultName : _name.text.trim();
    final lists = ref.read(bypassListsProvider.notifier);
    final existing = widget.existing;
    if (existing == null) {
      lists.add(name: name, text: _text.text, entries: report.entries);
    } else {
      lists.update(existing.copyWith(name: name, text: _text.text, entries: report.entries));
    }
    if (!mounted) return;
    Navigator.of(context).pop();
    if (ref.read(connectionProvider.notifier).isActive) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(context.l10n.bypassReconnect)));
    }
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final report = _report;
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.existing == null ? context.l10n.bypassNewList : context.l10n.bypassEditList),
        actions: [
          TextButton(onPressed: report == null || report.entries == 0 ? null : _save, child: Text(context.l10n.actionSave)),
        ],
      ),
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 560),
          child: ListView(
            padding: const EdgeInsets.all(16),
            children: [
              TextField(
                controller: _name,
                decoration: InputDecoration(labelText: context.l10n.bypassName),
              ),
              const SizedBox(height: 16),
              TextField(
                controller: _text,
                onChanged: (_) => _changed(),
                minLines: 8,
                maxLines: 16,
                autocorrect: false,
                enableSuggestions: false,
                keyboardType: TextInputType.multiline,
                style: const TextStyle(fontFamily: 'monospace', fontSize: 13),
                decoration: InputDecoration(
                  alignLabelWithHint: true,
                  labelText: context.l10n.bypassEntriesLabel,
                  hintText: 'example.com\nfull:www.example.org\nkeyword:bank\n203.0.113.0/24',
                  border: OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 12),
              if (_checkError != null)
                Text(_checkError!, style: TextStyle(color: scheme.error))
              else if (report != null) ...[
                Text(
                  report.entries == 0
                      ? context.l10n.bypassNoUsable
                      : context.l10n.bypassSummary(
                          context.l10n.entriesCount(report.entries),
                          context.l10n.bypassNames(report.domains),
                          context.l10n.bypassRanges(report.ranges),
                        ),
                ),
                if (report.issues.isNotEmpty) ...[
                  const SizedBox(height: 8),
                  Text(
                    context.l10n.bypassIgnored(report.issues.length),
                    style: TextStyle(color: scheme.error),
                  ),
                  for (final i in report.issues.take(8))
                    Text('  ${i.line}: ${i.text} — ${i.message}', style: Theme.of(context).textTheme.bodySmall),
                  if (report.issues.length > 8) Text('  ${context.l10n.bypassMoreIssues(report.issues.length - 8)}'),
                ],
              ],
            ],
          ),
        ),
      ),
    );
  }
}
