import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../state/providers.dart';

/// Shown on the first start (when there are no servers yet): what the app does,
/// how to add a server, and what the system will ask for. Skippable at any time.
class OnboardingScreen extends ConsumerStatefulWidget {
  const OnboardingScreen({super.key});

  @override
  ConsumerState<OnboardingScreen> createState() => _OnboardingScreenState();
}

class _Page {
  const _Page(this.icon, this.title, this.text);

  final IconData icon;
  final String title;
  final String text;
}

class _OnboardingScreenState extends ConsumerState<OnboardingScreen> {
  final _controller = PageController();
  int _page = 0;

  static final _pages = [
    const _Page(
      Icons.lock_outline,
      'Private and simple',
      'Omnitun connects you through VPN, proxy and Tor protocols from one calm screen. '
          'Pick a place, press Connect.',
    ),
    const _Page(
      Icons.dns_outlined,
      'Add your servers',
      'Import a share link, a QR code, a config file or a subscription from your provider. '
          'You can also add a server by hand.',
    ),
    _Page(
      Icons.verified_user_outlined,
      Platform.isAndroid ? 'Android will ask once' : 'Ready when you are',
      Platform.isAndroid
          ? 'The first time you connect, Android asks to allow a VPN connection. Omnitun can only protect '
                'your traffic after you accept. While connected, a notification shows the speed and a Disconnect button.'
          : 'Omnitun starts in proxy mode, which needs no special rights. Turn on VPN mode in Settings to capture all traffic.',
    ),
  ];

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  /// Closes the introduction; [addServer] tells the caller to open the import sheet next.
  void _finish({bool addServer = false}) {
    ref.read(onboardedProvider.notifier).done();
    Navigator.of(context).pop(addServer);
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final text = Theme.of(context).textTheme;
    final last = _page == _pages.length - 1;
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 480),
            child: Column(
              children: [
                Align(
                  alignment: Alignment.centerRight,
                  child: TextButton(onPressed: _finish, child: const Text('Skip')),
                ),
                Expanded(
                  child: PageView(
                    controller: _controller,
                    onPageChanged: (i) => setState(() => _page = i),
                    children: [
                      for (final p in _pages)
                        Padding(
                          padding: const EdgeInsets.symmetric(horizontal: 32),
                          child: Column(
                            mainAxisAlignment: MainAxisAlignment.center,
                            children: [
                              Container(
                                width: 120,
                                height: 120,
                                decoration: BoxDecoration(shape: BoxShape.circle, color: scheme.surfaceContainer),
                                child: Icon(p.icon, size: 56, color: scheme.primary),
                              ),
                              const SizedBox(height: 32),
                              Text(p.title, style: text.headlineSmall, textAlign: TextAlign.center),
                              const SizedBox(height: 12),
                              Text(
                                p.text,
                                style: text.bodyLarge?.copyWith(color: scheme.onSurfaceVariant),
                                textAlign: TextAlign.center,
                              ),
                            ],
                          ),
                        ),
                    ],
                  ),
                ),
                Row(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    for (var i = 0; i < _pages.length; i++)
                      Container(
                        margin: const EdgeInsets.all(4),
                        width: 8,
                        height: 8,
                        decoration: BoxDecoration(
                          shape: BoxShape.circle,
                          color: i == _page ? scheme.primary : scheme.outlineVariant,
                        ),
                      ),
                  ],
                ),
                Padding(
                  padding: const EdgeInsets.all(24),
                  child: SizedBox(
                    width: double.infinity,
                    height: 52,
                    child: FilledButton(
                      onPressed: () {
                        if (!last) {
                          _controller.nextPage(duration: const Duration(milliseconds: 250), curve: Curves.easeOut);
                          return;
                        }
                        // The last step ends in the thing to do: add a first server.
                        _finish(addServer: true);
                      },
                      child: Text(last ? 'Add my first server' : 'Next'),
                    ),
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
