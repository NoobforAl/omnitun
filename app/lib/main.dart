import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'l10n/l10n.dart';
import 'state/incoming.dart';
import 'state/providers.dart';
import 'state/store.dart';
import 'state/subscriptions.dart';
import 'ui/incoming_import.dart';
import 'ui/screens/home_screen.dart';
import 'ui/screens/onboarding_screen.dart';
import 'ui/screens/profiles_screen.dart';
import 'ui/theme.dart';

Future<void> main(List<String> args) async {
  WidgetsFlutterBinding.ensureInitialized();
  final store = await AppStore.open();
  // On a computer, opening a link or config file with Omnitun starts it with that as an argument.
  final startup = await IncomingShare.fromArguments(args);
  runApp(ProviderScope(
    overrides: [storeProvider.overrideWithValue(store), startupSharesProvider.overrideWithValue(startup)],
    child: const OmnitunApp(),
  ));
}

class OmnitunApp extends ConsumerStatefulWidget {
  const OmnitunApp({super.key});

  @override
  ConsumerState<OmnitunApp> createState() => _OmnitunAppState();
}

class _OmnitunAppState extends ConsumerState<OmnitunApp> {
  final _navigator = GlobalKey<NavigatorState>();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) async {
      // Update subscriptions whose refresh interval has passed.
      ref.read(subscriptionsProvider.notifier).refreshDue();
      // A link or file sent by another app comes first; the introduction steps aside for it.
      final shared = await IncomingShare.start(
        (text) => importShared(_navigator, ref, text),
        startup: ref.read(startupSharesProvider),
      );
      if (!shared) _introduction();
    });
  }

  /// The first start, with no servers yet, begins with a short introduction.
  Future<void> _introduction() async {
    if (ref.read(onboardedProvider)) return;
    if (ref.read(profilesProvider).isNotEmpty) return; // an existing user needs no introduction
    final navigator = _navigator.currentState;
    if (navigator == null) return;
    final addServer = await navigator.push<bool>(
      MaterialPageRoute(fullscreenDialog: true, builder: (_) => const OnboardingScreen()),
    );
    final overlay = _navigator.currentState?.overlay;
    if (addServer == true && overlay != null && overlay.mounted) showImportSheet(overlay.context, ref);
  }

  @override
  Widget build(BuildContext context) {
    // Collect core logs from startup, not only once the Logs tab is opened.
    ref.listen(logsProvider, (_, _) {});
    ref.watch(exitProvider); // looks up the exit address after each connection, when switched on
    final theme = ref.watch(settingsProvider.select((s) => s.theme));
    final language = ref.watch(settingsProvider.select((s) => s.language));
    return MaterialApp(
      navigatorKey: _navigator,
      title: 'Omnitun',
      debugShowCheckedModeBanner: false,
      theme: buildTheme(Brightness.light),
      darkTheme: buildTheme(Brightness.dark),
      themeMode: themeModeOf(theme),
      locale: language.isEmpty ? null : Locale(language),
      supportedLocales: AppLocalizations.supportedLocales,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      onGenerateTitle: (context) => context.l10n.appName,
      home: const HomeScreen(),
    );
  }
}
