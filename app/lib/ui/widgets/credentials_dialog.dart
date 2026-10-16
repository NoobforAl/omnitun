import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../engine/engine.dart';
import '../../l10n/l10n.dart';
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
        title: Text(context.l10n.credTitle),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(context.l10n.credNeeds(profile.name)),
            const SizedBox(height: 12),
            TextField(
              controller: user,
              autofocus: true,
              autocorrect: false,
              decoration: InputDecoration(labelText: context.l10n.credUsername),
            ),
            TextField(
              controller: pass,
              obscureText: obscure,
              autocorrect: false,
              decoration: InputDecoration(
                labelText: context.l10n.credPassword,
                suffixIcon: IconButton(
                  icon: Icon(obscure ? Icons.visibility : Icons.visibility_off),
                  onPressed: () => setState(() => obscure = !obscure),
                ),
              ),
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(context, false), child: Text(context.l10n.actionCancel)),
          FilledButton(
            onPressed: () {
              if (user.text.trim().isEmpty || pass.text.isEmpty) return;
              Navigator.pop(context, true);
            },
            child: Text(context.l10n.actionSave),
          ),
        ],
      ),
    ),
  );
  if (ok != true) return false;
  ref.read(profilesProvider.notifier).setCredentials(profile.id, user.text.trim(), pass.text);
  return true;
}
