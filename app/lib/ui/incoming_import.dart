import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../l10n/errors.dart';
import '../l10n/l10n.dart';
import '../state/providers.dart';
import 'screens/subscription_screen.dart';

/// Imports what another app sent: a subscription link opens the subscription form, anything else
/// (share links, config files, bridge lines) goes through the importer and is saved at once.
Future<void> importShared(GlobalKey<NavigatorState> navigator, WidgetRef ref, String text) async {
  final context = navigator.currentContext;
  if (context == null) return;
  final l = context.l10n;
  final messenger = ScaffoldMessenger.of(context);
  text = text.trim();
  if (text.isEmpty) return;

  // A single http(s) or nordvpn:// URL is a subscription, not a config.
  if (!text.contains('\n') && RegExp(r'^(https?|nordvpn)://\S+$').hasMatch(text)) {
    navigator.currentState?.push(MaterialPageRoute<void>(builder: (_) => SubscriptionScreen(initialUrl: text)));
    return;
  }

  try {
    final result = await ref.read(engineProvider).importText(text);
    final added = ref.read(profilesProvider.notifier).addAll(result.profiles);
    final message = result.profiles.isNotEmpty
        ? (result.errors.isEmpty
              ? l.importedProfiles(added, result.profiles.length - added)
              : l.importPartial(added, result.errors.length))
        : l.importNothing(
            result.failure != null
                ? describeError(l, result.failure!)
                : (result.errors.isEmpty ? '–' : result.errors.first),
          );
    messenger.showSnackBar(SnackBar(content: Text(message)));
  } catch (e) {
    messenger.showSnackBar(SnackBar(content: Text(l.couldNotReadFile('$e'))));
  }
}
