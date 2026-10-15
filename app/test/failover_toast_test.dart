import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:omnitun/engine/engine.dart';
import 'package:omnitun/main.dart';
import 'package:omnitun/state/providers.dart';
import 'package:omnitun/state/store.dart';

/// Engine whose states are pushed by the test.
class ScriptedEngine implements VpnEngine {
  final controller = StreamController<EngineState>.broadcast();

  @override
  Stream<EngineState> get states => controller.stream;
  @override
  Stream<String> get logs => const Stream.empty();
  @override
  String get name => 'scripted';
  @override
  Future<void> connect(ProfileInfo profile,
      {List<ProfileInfo> fallbacks = const [], ConnectOptions options = const ConnectOptions()}) async {}
  @override
  Future<void> disconnect() async {}
  @override
  Future<ImportResult> importText(String text) async => const ImportResult([], []);
  @override
  Future<int> testLatency(ProfileInfo profile) async => 0;
  @override
  Future<SubscriptionResult> fetchSubscription(Map<String, dynamic> request) async =>
      const SubscriptionResult([], [], SubscriptionInfo());
  @override
  Future<void> setStopOnClose(bool value) async {}
  @override
  void dispose() => controller.close();
}

Map<String, dynamic> _p(String id, String name) =>
    {'id': id, 'name': name, 'protocol': 'shadowsocks', 'server': '$id.example.com', 'port': 8388};

void main() {
  testWidgets('a failover shows a toast naming both profiles and follows the switch', (tester) async {
    final engine = ScriptedEngine();
    final store = AppStore.memory({
      'profiles': [_p('dead', 'Dead-server'), _p('live', 'PC-dev')],
      'selected': 'dead',
    });
    await tester.pumpWidget(ProviderScope(
      overrides: [storeProvider.overrideWithValue(store), engineProvider.overrideWithValue(engine)],
      child: const OmnitunApp(),
    ));
    await tester.pump();

    final since = DateTime.now();
    engine.controller.add(EngineState(status: ConnStatus.connected, activeProfileId: 'dead', connectedSince: since));
    await tester.pump();
    engine.controller.add(EngineState(
      status: ConnStatus.connected,
      activeProfileId: 'live',
      connectedSince: since,
      failover: FailoverInfo(from: 'dead', to: 'live', reason: 'timeout', at: DateTime.now()),
    ));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));

    expect(find.textContaining('Switched to PC-dev'), findsOneWidget);
    expect(find.text('PC-dev'), findsOneWidget); // home card follows the active profile
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('connecting shows the progress ring, percent and current step', (tester) async {
    final engine = ScriptedEngine();
    final store = AppStore.memory({'profiles': [_p('live', 'PC-dev')]});
    await tester.pumpWidget(ProviderScope(
      overrides: [storeProvider.overrideWithValue(store), engineProvider.overrideWithValue(engine)],
      child: const OmnitunApp(),
    ));
    await tester.pump();

    engine.controller.add(const EngineState(
      status: ConnStatus.connecting,
      activeProfileId: 'live',
      progress: ConnectProgress('handshake', 85, profile: 'PC-dev'),
    ));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));
    expect(find.text('85%'), findsOneWidget);
    expect(find.text('Contacting server · PC-dev'), findsOneWidget);
    final ring = tester.widget<CircularProgressIndicator>(find.byType(CircularProgressIndicator));
    expect(ring.value, closeTo(0.85, 0.01));

    // A failure freezes the ring at the failed stage and shows the reason.
    engine.controller.add(const EngineState(
      status: ConnStatus.error,
      error: 'Contacting server failed: connection refused',
      progress: ConnectProgress('handshake', 85, profile: 'PC-dev'),
    ));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 500));
    expect(find.text('Contacting server failed: connection refused'), findsOneWidget);
    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    expect(find.text('85%'), findsNothing);
    await tester.pumpWidget(const SizedBox());
  });
}
