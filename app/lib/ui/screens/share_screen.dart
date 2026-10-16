import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:qr_flutter/qr_flutter.dart';

import '../../engine/engine.dart';
import '../../state/providers.dart';
import 'profile_editor_screen.dart';

/// A server's share link as text and as a QR code, for another device.
class ShareScreen extends ConsumerStatefulWidget {
  const ShareScreen({super.key, required this.profile});

  final ProfileInfo profile;

  @override
  ConsumerState<ShareScreen> createState() => _ShareScreenState();
}

class _ShareScreenState extends ConsumerState<ShareScreen> {
  late final Future<String> _link = ref.read(engineProvider).exportLink(widget.profile);

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final text = Theme.of(context).textTheme;
    return Scaffold(
      appBar: AppBar(title: Text(widget.profile.name, overflow: TextOverflow.ellipsis)),
      body: FutureBuilder<String>(
        future: _link,
        builder: (context, snap) {
          if (snap.connectionState != ConnectionState.done) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snap.hasError) {
            return Center(
              child: Padding(
                padding: const EdgeInsets.all(24),
                child: Text(
                  'This server cannot be shared as a link: ${messageOf(snap.error!)}',
                  textAlign: TextAlign.center,
                ),
              ),
            );
          }
          final link = snap.data!;
          return Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 480),
              child: ListView(
                padding: const EdgeInsets.all(20),
                children: [
                  Center(
                    child: Container(
                      padding: const EdgeInsets.all(14),
                      decoration: BoxDecoration(color: Colors.white, borderRadius: BorderRadius.circular(16)),
                      child: QrImageView(
                        data: link,
                        size: 260,
                        backgroundColor: Colors.white,
                        errorStateBuilder: (_, _) => const SizedBox(
                          width: 260,
                          height: 260,
                          child: Center(
                            child: Text('Too long for a QR code: copy the link', textAlign: TextAlign.center),
                          ),
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(height: 16),
                  Text(
                    'Scan it with Omnitun on another device, or copy the link.',
                    style: text.bodyMedium,
                    textAlign: TextAlign.center,
                  ),
                  const SizedBox(height: 16),
                  Container(
                    padding: const EdgeInsets.all(12),
                    decoration: BoxDecoration(color: scheme.surfaceContainer, borderRadius: BorderRadius.circular(12)),
                    child: SelectableText(
                      link,
                      maxLines: 6,
                      style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
                    ),
                  ),
                  const SizedBox(height: 12),
                  FilledButton.icon(
                    onPressed: () async {
                      await Clipboard.setData(ClipboardData(text: link));
                      if (context.mounted) {
                        ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('Link copied')));
                      }
                    },
                    icon: const Icon(Icons.copy),
                    label: const Text('Copy link'),
                  ),
                  const SizedBox(height: 16),
                  Text(
                    'The link contains this server\'s password or key: only share it with people who may use the server.',
                    style: text.bodySmall?.copyWith(color: scheme.onSurfaceVariant),
                    textAlign: TextAlign.center,
                  ),
                ],
              ),
            ),
          );
        },
      ),
    );
  }
}
