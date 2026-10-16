import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../engine/engine.dart';
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
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('The clipboard is empty')));
      return;
    }
    _edit(context, text: text, name: 'Pasted list');
  }

  Future<void> _fromFile(BuildContext context) async {
    try {
      final file = await FilePicker.pickFile(dialogTitle: 'Open a bypass list (.txt)');
      if (file == null || !context.mounted) return;
      final text = await file.xFile.readAsString();
      if (!context.mounted) return;
      _edit(context, text: text, name: file.name.replaceFirst(RegExp(r'\.[A-Za-z0-9]+$'), ''));
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Could not read the file: $e')));
      }
    }
  }

  Future<void> _fromUrl(BuildContext context) async {
    final url = await showDialog<String>(context: context, builder: (_) => const _UrlDialog());
    if (url == null || !context.mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    try {
      final text = await downloadList(url);
      if (!context.mounted) return;
      _edit(context, text: text, name: Uri.parse(url).host);
    } catch (e) {
      messenger.showSnackBar(SnackBar(content: Text('Could not download the list: $e')));
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
        title: const Text('Bypass lists'),
        actions: [
          PopupMenuButton<String>(
            tooltip: 'Add a list',
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
                  _edit(context, name: 'My list');
              }
            },
            itemBuilder: (_) => const [
              PopupMenuItem(value: 'paste', child: Text('Paste from the clipboard')),
              PopupMenuItem(value: 'file', child: Text('Open a .txt file')),
              PopupMenuItem(value: 'url', child: Text('Download from a link')),
              PopupMenuItem(value: 'type', child: Text('Type it')),
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
              const Padding(
                padding: EdgeInsets.fromLTRB(16, 4, 16, 12),
                child: Text(
                  'Websites and addresses that should not use the VPN, for example your bank or local services. '
                  'One entry per line: example.com (and its subdomains), full:www.example.com, keyword:bank, '
                  'regexp:…, 203.0.113.7 or 10.0.0.0/8. Lines starting with # are ignored.',
                ),
              ),
              if (lists.isEmpty)
                Padding(
                  padding: const EdgeInsets.all(24),
                  child: Column(
                    children: [
                      const Text('No lists yet.'),
                      const SizedBox(height: 12),
                      FilledButton.icon(
                        onPressed: () => _fromFile(context),
                        icon: const Icon(Icons.file_open_outlined),
                        label: const Text('Open a .txt file'),
                      ),
                      TextButton(
                        onPressed: () => _edit(context, name: 'My list'),
                        child: const Text('Type a list'),
                      ),
                    ],
                  ),
                ),
              for (final l in lists)
                SwitchListTile(
                  title: Text(l.name, overflow: TextOverflow.ellipsis),
                  subtitle: Text('${l.entries} entr${l.entries == 1 ? 'y' : 'ies'}'),
                  value: l.enabled,
                  secondary: PopupMenuButton<String>(
                    tooltip: 'More',
                    onSelected: (v) {
                      if (v == 'edit') _edit(context, list: l);
                      if (v == 'delete') ref.read(bypassListsProvider.notifier).remove(l.id);
                    },
                    itemBuilder: (_) => const [
                      PopupMenuItem(value: 'edit', child: Text('Edit')),
                      PopupMenuItem(value: 'delete', child: Text('Delete')),
                    ],
                  ),
                  onChanged: (v) => ref.read(bypassListsProvider.notifier).update(l.copyWith(enabled: v)),
                ),
              const Divider(height: 32),
              SwitchListTile(
                title: const Text('Only the listed destinations use the VPN'),
                subtitle: Text(
                  settings.bypassOnly
                      ? 'Everything else goes direct, outside the VPN.'
                      : 'Off: the listed destinations go direct and everything else uses the VPN.',
                ),
                value: settings.bypassOnly,
                onChanged: (v) => ref.read(settingsProvider.notifier).update((s) => s.copyWith(bypassOnly: v)),
              ),
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 8, 16, 0),
                child: Text(
                  enabled == 0
                      ? 'Nothing is bypassed until a list is on.'
                      : '$enabled entr${enabled == 1 ? 'y is' : 'ies are'} in use. Changes apply the next time you connect.',
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
Future<String> downloadList(String url) async {
  final uri = Uri.parse(url.trim());
  if (!(uri.isScheme('http') || uri.isScheme('https')) || uri.host.isEmpty) {
    throw 'Enter an http(s) link';
  }
  final client = HttpClient()..connectionTimeout = const Duration(seconds: 15);
  try {
    final request = await client.getUrl(uri).timeout(const Duration(seconds: 20));
    final response = await request.close().timeout(const Duration(seconds: 20));
    if (response.statusCode != 200) throw 'the server answered ${response.statusCode}';
    final bytes = <int>[];
    await for (final chunk in response) {
      bytes.addAll(chunk);
      if (bytes.length > 5 << 20) throw 'the list is larger than 5 MB';
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
      title: const Text('Download a list'),
      content: TextField(
        controller: _controller,
        autofocus: true,
        keyboardType: TextInputType.url,
        decoration: const InputDecoration(hintText: 'https://example.com/bypass.txt'),
      ),
      actions: [
        TextButton(onPressed: () => Navigator.pop(context), child: const Text('Cancel')),
        FilledButton(onPressed: () => Navigator.pop(context, _controller.text.trim()), child: const Text('Download')),
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
    await _checkNow();
    final report = _report;
    if (report == null || report.entries == 0) return;
    final name = _name.text.trim().isEmpty ? 'My list' : _name.text.trim();
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
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('Reconnect to use the new list')));
    }
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final report = _report;
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.existing == null ? 'New list' : 'Edit list'),
        actions: [
          TextButton(onPressed: report == null || report.entries == 0 ? null : _save, child: const Text('Save')),
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
                decoration: const InputDecoration(labelText: 'Name'),
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
                decoration: const InputDecoration(
                  alignLabelWithHint: true,
                  labelText: 'Entries, one per line',
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
                      ? 'No usable entries yet.'
                      : '${report.entries} entr${report.entries == 1 ? 'y' : 'ies'}: '
                            '${report.domains} name${report.domains == 1 ? '' : 's'}, '
                            '${report.ranges} address${report.ranges == 1 ? '' : 'es'} or range${report.ranges == 1 ? '' : 's'}.',
                ),
                if (report.issues.isNotEmpty) ...[
                  const SizedBox(height: 8),
                  Text(
                    '${report.issues.length} line${report.issues.length == 1 ? '' : 's'} will be ignored:',
                    style: TextStyle(color: scheme.error),
                  ),
                  for (final i in report.issues.take(8))
                    Text('  ${i.line}: ${i.text} — ${i.message}', style: Theme.of(context).textTheme.bodySmall),
                  if (report.issues.length > 8) Text('  … and ${report.issues.length - 8} more'),
                ],
              ],
            ],
          ),
        ),
      ),
    );
  }
}
