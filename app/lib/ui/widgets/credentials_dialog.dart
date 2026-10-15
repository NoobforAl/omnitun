import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../engine/engine.dart';
import '../../state/providers.dart';

/// Asks for the username and password of [profile] and saves them.
/// Returns true when saved.
Future<bool> askCredentials(BuildContext context, WidgetRef ref, ProfileInfo profile) async {
  final user = TextEditingController(text: profile.raw['username'] as String? ?? '');
  final pass = TextEditingController(text: profile.raw['password'] as String? ?? '');
  var obscure = true;
  final ok = await showDialog<bool>(
    context: context,
    builder: (context) => StatefulBuilder(
      builder: (context, setState) => AlertDialog(
        title: const Text('Sign in'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text('${profile.name} needs a username and password.'),
            const SizedBox(height: 12),
            TextField(
              controller: user,
              autofocus: true,
              autocorrect: false,
              decoration: const InputDecoration(labelText: 'Username'),
            ),
            TextField(
              controller: pass,
              obscureText: obscure,
              autocorrect: false,
              decoration: InputDecoration(
                labelText: 'Password',
                suffixIcon: IconButton(
                  icon: Icon(obscure ? Icons.visibility : Icons.visibility_off),
                  onPressed: () => setState(() => obscure = !obscure),
                ),
              ),
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: const Text('Cancel')),
          FilledButton(
            onPressed: () {
              if (user.text.trim().isEmpty || pass.text.isEmpty) return;
              Navigator.pop(context, true);
            },
            child: const Text('Save'),
          ),
        ],
      ),
    ),
  );
  if (ok != true) return false;
  ref.read(profilesProvider.notifier).setCredentials(profile.id, user.text.trim(), pass.text);
  return true;
}
