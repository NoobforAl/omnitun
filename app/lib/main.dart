import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'ui/app_shell.dart';
import 'ui/theme.dart';

void main() {
  runApp(const ProviderScope(child: OmnitunApp()));
}

class OmnitunApp extends StatelessWidget {
  const OmnitunApp({super.key});

  @override
  Widget build(BuildContext context) {
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
