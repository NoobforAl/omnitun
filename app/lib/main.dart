import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'state/providers.dart';
import 'state/store.dart';
import 'state/subscriptions.dart';
import 'ui/app_shell.dart';
import 'ui/theme.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  final store = await AppStore.open();
  runApp(ProviderScope(
    overrides: [storeProvider.overrideWithValue(store)],
    child: const OmnitunApp(),
  ));
}

class OmnitunApp extends ConsumerStatefulWidget {
  const OmnitunApp({super.key});

  @override
  ConsumerState<OmnitunApp> createState() => _OmnitunAppState();
}

class _OmnitunAppState extends ConsumerState<OmnitunApp> {
  @override
  void initState() {
    super.initState();
    // Update subscriptions whose refresh interval has passed.
    WidgetsBinding.instance.addPostFrameCallback((_) => ref.read(subscriptionsProvider.notifier).refreshDue());
  }

  @override
  Widget build(BuildContext context) {
    // Collect core logs from startup, not only once the Logs tab is opened.
    ref.listen(logsProvider, (_, _) {});
    return MaterialApp(
      title: 'Omnitun',
      debugShowCheckedModeBanner: false,
      theme: buildTheme(Brightness.light),
      darkTheme: buildTheme(Brightness.dark),
      themeMode: ThemeMode.system,
      home: const AppShell(),
    );
  }
}
