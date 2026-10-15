import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../state/subscriptions.dart';

/// Adds a subscription (or edits one when [existing] is given).
class SubscriptionScreen extends ConsumerStatefulWidget {
  const SubscriptionScreen({super.key, this.initialUrl = '', this.existing});

  final String initialUrl;
  final Subscription? existing;

  @override
  ConsumerState<SubscriptionScreen> createState() => _SubscriptionScreenState();
}

class _SubscriptionScreenState extends ConsumerState<SubscriptionScreen> {
  static const _authTypes = {
    'none': 'None',
    'basic': 'Username & password (Basic)',
    'bearer': 'Bearer token',
    'header': 'Custom header',
    'query': 'Token in URL',
  };

  late final _name = TextEditingController(text: widget.existing?.name ?? '');
  late final _url = TextEditingController(text: widget.existing?.url ?? widget.initialUrl);
  late final _username = TextEditingController(text: widget.existing?.auth.username ?? '');
  late final _password = TextEditingController(text: widget.existing?.auth.password ?? '');
  late final _token = TextEditingController(text: widget.existing?.auth.token ?? '');
  late final _header = TextEditingController(text: widget.existing?.auth.header ?? '');
  late final _queryKey = TextEditingController(text: widget.existing?.auth.queryKey ?? 'token');
  late String _authType = widget.existing?.auth.type ?? 'none';
  bool _busy = false;
  bool _showSecret = false;
  String? _error;

  @override
  void dispose() {
    for (final c in [_name, _url, _username, _password, _token, _header, _queryKey]) {
      c.dispose();
    }
    super.dispose();
  }

  SubscriptionAuth get _auth => SubscriptionAuth(
        type: _authType,
        username: _authType == 'basic' ? _username.text.trim() : '',
        password: _authType == 'basic' ? _password.text : '',
        token: const ['bearer', 'header', 'query'].contains(_authType) ? _token.text.trim() : '',
        header: _authType == 'header' ? _header.text.trim() : '',
        queryKey: _authType == 'query' ? _queryKey.text.trim() : '',
      );

  String? _validate() {
    final uri = Uri.tryParse(_url.text.trim());
    // nordvpn://<access token>?country=DE gives NordLynx (WireGuard) servers.
    if (uri == null || !(uri.isScheme('http') || uri.isScheme('https') || uri.isScheme('nordvpn')) || uri.host.isEmpty) {
      return 'Enter an http(s) subscription URL (or nordvpn://<access token>)';
    }
    if (_authType == 'header' && _header.text.trim().isEmpty) return 'Enter the header name';
    if (const ['bearer', 'header', 'query'].contains(_authType) && _token.text.trim().isEmpty) {
      return 'Enter the token';
    }
    return null;
  }

  Future<void> _save() async {
    final problem = _validate();
    if (problem != null) {
      setState(() => _error = problem);
      return;
    }
    setState(() {
      _busy = true;
      _error = null;
    });
    final ctrl = ref.read(subscriptionsProvider.notifier);
    try {
      final int count;
      final existing = widget.existing;
      if (existing == null) {
        count = await ctrl.add(url: _url.text, name: _name.text.trim(), auth: _auth);
      } else {
        ctrl.update(existing.copyWith(name: _name.text.trim(), url: _url.text.trim(), auth: _auth));
        count = await ctrl.refresh(existing.id);
      }
      if (!mounted) return;
      Navigator.of(context).pop();
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Subscription updated: $count profile(s)')));
    } catch (e) {
      if (mounted) {
        setState(() {
          _busy = false;
          _error = e.toString().replaceFirst(RegExp(r'^(Bad state|EngineException|[A-Z_]+): '), '');
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final secret = _showSecret ? null : '•';
    return Scaffold(
      appBar: AppBar(title: Text(widget.existing == null ? 'Add subscription' : 'Edit subscription')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          TextField(
            controller: _url,
            keyboardType: TextInputType.url,
            autofocus: widget.existing == null && widget.initialUrl.isEmpty,
            decoration: const InputDecoration(
              labelText: 'Subscription URL',
              hintText: 'https://provider.example/sub/… or nordvpn://<token>',
              border: OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _name,
            decoration: const InputDecoration(
              labelText: 'Name (optional)',
              helperText: 'Defaults to the provider\'s title or the host name',
              border: OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 20),
          DropdownButtonFormField<String>(
            initialValue: _authType,
            decoration: const InputDecoration(labelText: 'Authentication', border: OutlineInputBorder()),
            items: [
              for (final e in _authTypes.entries) DropdownMenuItem(value: e.key, child: Text(e.value)),
            ],
            onChanged: _busy ? null : (v) => setState(() => _authType = v ?? 'none'),
          ),
          const SizedBox(height: 12),
          if (_authType == 'basic') ...[
            TextField(
              controller: _username,
              decoration: const InputDecoration(labelText: 'Username', border: OutlineInputBorder()),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _password,
              obscureText: !_showSecret,
              obscuringCharacter: secret ?? '•',
              decoration: InputDecoration(
                labelText: 'Password',
                border: const OutlineInputBorder(),
                suffixIcon: IconButton(
                  onPressed: () => setState(() => _showSecret = !_showSecret),
                  icon: Icon(_showSecret ? Icons.visibility_off : Icons.visibility),
                ),
              ),
            ),
          ],
          if (_authType == 'header') ...[
            TextField(
              controller: _header,
              decoration: const InputDecoration(
                labelText: 'Header name',
                hintText: 'X-Api-Key',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 12),
          ],
          if (_authType == 'query') ...[
            TextField(
              controller: _queryKey,
              decoration: const InputDecoration(labelText: 'Parameter name', border: OutlineInputBorder()),
            ),
            const SizedBox(height: 12),
          ],
          if (const ['bearer', 'header', 'query'].contains(_authType))
            TextField(
              controller: _token,
              obscureText: !_showSecret,
              decoration: InputDecoration(
                labelText: _authType == 'header' ? 'Header value' : 'Token',
                border: const OutlineInputBorder(),
                suffixIcon: IconButton(
                  onPressed: () => setState(() => _showSecret = !_showSecret),
                  icon: Icon(_showSecret ? Icons.visibility_off : Icons.visibility),
                ),
              ),
            ),
          if (_error != null)
            Padding(
              padding: const EdgeInsets.only(top: 16),
              child: Text(_error!, style: TextStyle(color: Theme.of(context).colorScheme.error)),
            ),
          const SizedBox(height: 24),
          FilledButton.icon(
            onPressed: _busy ? null : _save,
            icon: _busy
                ? const SizedBox(width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2))
                : const Icon(Icons.cloud_download_outlined),
            label: Text(widget.existing == null ? 'Add and download' : 'Save and refresh'),
          ),
        ],
      ),
    );
  }
}
