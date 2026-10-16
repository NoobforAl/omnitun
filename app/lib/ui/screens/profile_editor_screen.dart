import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../engine/engine.dart';
import '../../l10n/form_text.dart';
import '../../l10n/l10n.dart';
import '../../state/countries.dart';
import '../../state/providers.dart';
import '../widgets/country_flag.dart';
import 'location_screen.dart';
import 'qr_scan_screen.dart';

/// Step one of adding a server by hand: which protocol.
class NewServerScreen extends ConsumerWidget {
  const NewServerScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final forms = ref.watch(formsProvider);
    return Scaffold(
      appBar: AppBar(title: Text(context.l10n.addServerTitle)),
      body: forms.when(
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => Center(
          child: Padding(padding: const EdgeInsets.all(24), child: Text(messageOf(e))),
        ),
        data: (list) => ListView(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 4, 16, 8),
              child: Text(context.l10n.addServerIntro),
            ),
            for (final f in list)
              ListTile(
                title: Text(f.label),
                trailing: const Icon(Icons.chevron_right),
                onTap: () async {
                  final added = await Navigator.of(context)
                      .push<ProfileInfo>(MaterialPageRoute(builder: (_) => ProfileEditorScreen(protocol: f.protocol)));
                  if (added != null && context.mounted) Navigator.of(context).pop();
                },
              ),
          ],
        ),
      ),
    );
  }
}

/// What to tell the user about an error from the core.
String messageOf(Object e) => e is EngineException ? e.message : e.toString().replaceFirst(RegExp(r'^[A-Za-z]+: '), '');

/// Edits a profile, or creates one of [protocol]. The fields come from the core
/// (`forms`), so every protocol has the same kind of editor and the core checks the values.
class ProfileEditorScreen extends ConsumerStatefulWidget {
  const ProfileEditorScreen({super.key, this.protocol, this.profile}) : assert(protocol != null || profile != null);

  final String? protocol;
  final ProfileInfo? profile;

  @override
  ConsumerState<ProfileEditorScreen> createState() => _ProfileEditorScreenState();
}

class _ProfileEditorScreenState extends ConsumerState<ProfileEditorScreen> {
  ProtocolFormSpec? _form;
  final _text = <String, TextEditingController>{};
  final _values = <String, String>{}; // switches and choices
  final _hidden = <String>{}; // secret fields currently shown
  String? _loadError;
  String? _saveError;
  bool _saving = false;

  /// The country picked here: a code, '' for "work it out from the name", null = untouched.
  String? _country;

  String get _protocol => widget.protocol ?? widget.profile!.protocol;
  bool get _editing => widget.profile != null;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    for (final c in _text.values) {
      c.dispose();
    }
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final forms = await ref.read(formsProvider.future);
      final form = forms.firstWhere((f) => f.protocol == _protocol, orElse: () => throw _NoEditor());
      final values = _editing
          ? await ref.read(engineProvider).profileValues(widget.profile!)
          : {for (final f in form.fields) f.key: f.defaultValue};
      if (!mounted) return;
      setState(() {
        _form = form;
        for (final f in form.fields) {
          final v = values[f.key] ?? f.defaultValue;
          if (f.kind == 'bool' || f.kind == 'choice') {
            _values[f.key] = v;
          } else {
            _text[f.key] = TextEditingController(text: v);
          }
        }
      });
    } catch (e) {
      if (mounted) setState(() => _loadError = e is _NoEditor ? context.l10n.formNoEditor(_protocol) : messageOf(e));
    }
  }

  Future<void> _save() async {
    final form = _form;
    final alreadyInList = context.l10n.alreadyInList;
    if (form == null || _saving) return;
    setState(() {
      _saving = true;
      _saveError = null;
    });
    final values = <String, String>{for (final f in form.fields) f.key: _text[f.key]?.text ?? _values[f.key] ?? ''};
    try {
      final built = await ref
          .read(engineProvider)
          .buildProfile(values, protocol: _editing ? null : _protocol, profile: widget.profile);
      final profiles = ref.read(profilesProvider.notifier);
      if (_editing) {
        profiles.replace(widget.profile!.id, built);
      } else if (profiles.addAll([built]) == 0) {
        throw EngineException('CONFLICT', alreadyInList);
      }
      if (_country != null) {
        ref.read(countryOverridesProvider.notifier).set(built.id, _country!.isEmpty ? null : _country);
      }
      if (mounted) Navigator.of(context).pop(built);
    } catch (e) {
      if (mounted) {
        setState(() {
          _saving = false;
          _saveError = messageOf(e);
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final form = _form;
    final title = _editing
        ? (_protocol == 'tor' ? context.l10n.torBridgesTitle : context.l10n.editServerTitle)
        : context.l10n.newServerTitle(form?.label ?? '');
    return Scaffold(
      appBar: AppBar(
        title: Text(title),
        actions: [TextButton(onPressed: form == null || _saving ? null : _save, child: Text(context.l10n.actionSave))],
      ),
      body: _loadError != null
          ? Center(
              child: Padding(padding: const EdgeInsets.all(24), child: Text(_loadError!)),
            )
          : form == null
          ? const Center(child: CircularProgressIndicator())
          : _body(context, form),
    );
  }

  Widget _body(BuildContext context, ProtocolFormSpec form) {
    final scheme = Theme.of(context).colorScheme;
    final basic = form.fields.where((f) => !f.advanced).toList();
    final more = form.fields.where((f) => f.advanced).toList();
    final profile = widget.profile;
    final countryOf = ref.watch(profileCountryProvider);
    final code = _country != null
        ? (_country!.isEmpty ? null : _country)
        : (profile == null ? null : countryOf(profile));
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 560),
        child: ListView(
          padding: const EdgeInsets.fromLTRB(16, 4, 16, 32),
          children: [
            if (_saveError != null)
              Container(
                margin: const EdgeInsets.only(bottom: 12),
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(color: scheme.errorContainer, borderRadius: BorderRadius.circular(12)),
                child: Text(_saveError!, style: TextStyle(color: scheme.onErrorContainer)),
              ),
            if (profile != null && profile.group != ProfileInfo.defaultGroup)
              Padding(
                padding: const EdgeInsets.only(bottom: 12),
                child: Text(context.l10n.subscriptionEditNote),
              ),
            for (final f in basic) _field(context, f),
            if (_protocol != 'tor')
              ListTile(
                contentPadding: EdgeInsets.zero,
                leading: CountryFlag(code),
                title: Text(context.l10n.countryRow),
                subtitle: Text(code == null ? context.l10n.countryAuto : countryNames[code] ?? code),
                trailing: const Icon(Icons.chevron_right),
                onTap: () async {
                  final picked = await pickCountry(context, code);
                  if (picked != null) setState(() => _country = picked);
                },
              ),
            if (more.isNotEmpty)
              ExpansionTile(
                tilePadding: EdgeInsets.zero,
                title: Text(context.l10n.moreOptions),
                childrenPadding: EdgeInsets.zero,
                children: [for (final f in more) _field(context, f)],
              ),
          ],
        ),
      ),
    );
  }

  Widget _field(BuildContext context, FormFieldSpec f) {
    final l = context.l10n;
    final fieldLabel = formLabel(l, f.label);
    final label = f.required ? l.fieldRequired(fieldLabel) : fieldLabel;
    final helper = f.help.isEmpty ? null : formHelp(l, f.help);
    Widget child;
    switch (f.kind) {
      case 'bool':
        child = SwitchListTile(
          contentPadding: EdgeInsets.zero,
          title: Text(fieldLabel),
          subtitle: helper == null ? null : Text(helper),
          value: _values[f.key] == 'true',
          onChanged: (v) => setState(() => _values[f.key] = v ? 'true' : 'false'),
        );
      case 'choice':
        final current = _values[f.key] ?? '';
        child = DropdownButtonFormField<String>(
          initialValue: f.choices.contains(current) ? current : null,
          isExpanded: true,
          decoration: InputDecoration(labelText: label, helperText: helper, helperMaxLines: 3),
          items: [for (final c in f.choices) DropdownMenuItem(value: c, child: Text(_choiceLabel(l, f, c)))],
          onChanged: (v) => setState(() => _values[f.key] = v ?? ''),
        );
      default:
        final controller = _text[f.key]!;
        final secret = f.kind == 'secret';
        final multiline = f.kind == 'multiline';
        child = Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            TextField(
              controller: controller,
              obscureText: secret && !_hidden.contains(f.key),
              enableSuggestions: !secret,
              autocorrect: false,
              minLines: multiline ? 3 : 1,
              maxLines: multiline ? 8 : 1,
              keyboardType: switch (f.kind) {
                'int' => TextInputType.number,
                'multiline' => TextInputType.multiline,
                _ => TextInputType.text,
              },
              decoration: InputDecoration(
                labelText: label,
                helperText: helper ?? (f.kind == 'list' ? l.listHelper : null),
                helperMaxLines: 3,
                suffixIcon: secret
                    ? IconButton(
                        tooltip: _hidden.contains(f.key) ? l.tooltipHide : l.tooltipShow,
                        icon: Icon(_hidden.contains(f.key) ? Icons.visibility_off : Icons.visibility),
                        onPressed: () =>
                            setState(() => _hidden.contains(f.key) ? _hidden.remove(f.key) : _hidden.add(f.key)),
                      )
                    : null,
              ),
            ),
            if (multiline) _FillButtons(controller: controller),
          ],
        );
    }
    return Padding(padding: const EdgeInsets.only(bottom: 14), child: child);
  }

  static String _choiceLabel(AppLocalizations l, FormFieldSpec f, String c) {
    if (f.key == 'tor.builtin') {
      return switch (c) {
        '' => l.torBridgesNone,
        'snowflake' => l.torBridgesSnowflake,
        'obfs4' => l.torBridgesObfs4,
        _ => c,
      };
    }
    return c.isEmpty ? l.choiceDefault : c;
  }
}

/// Paste, file and QR buttons under a multi-line field (certificates, bridge lines).
class _FillButtons extends StatelessWidget {
  const _FillButtons({required this.controller});

  final TextEditingController controller;

  void _add(String text) {
    final t = text.trim();
    if (t.isEmpty) return;
    controller.text = controller.text.trim().isEmpty ? t : '${controller.text.trimRight()}\n$t';
  }

  @override
  Widget build(BuildContext context) {
    return Align(
      alignment: Alignment.centerLeft,
      child: Wrap(
        spacing: 8,
        children: [
          TextButton.icon(
            onPressed: () async => _add((await Clipboard.getData(Clipboard.kTextPlain))?.text ?? ''),
            icon: const Icon(Icons.content_paste),
            label: Text(context.l10n.fillPaste),
          ),
          TextButton.icon(
            onPressed: () async {
              try {
                final file = await FilePicker.pickFile(dialogTitle: context.l10n.openTextFile);
                if (file != null) _add(await file.xFile.readAsString());
              } catch (e) {
                if (context.mounted) {
                  ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(context.l10n.couldNotReadFile('$e'))));
                }
              }
            },
            icon: const Icon(Icons.file_open_outlined),
            label: Text(context.l10n.fillFile),
          ),
          if (QrScanScreen.supported)
            TextButton.icon(
              onPressed: () async {
                final text = await Navigator.of(context)
                    .push<String>(MaterialPageRoute(builder: (_) => const QrScanScreen()));
                if (text != null) _add(text);
              },
              icon: const Icon(Icons.qr_code_scanner),
              label: Text(context.l10n.fillScanQr),
            ),
        ],
      ),
    );
  }
}

/// The core has no editor for the profile's protocol.
class _NoEditor implements Exception {}
