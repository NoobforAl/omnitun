import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../l10n/l10n.dart';

/// One license text and the packages that use it, from scripts/gen-notices.py.
class NoticeGroup {
  const NoticeGroup(this.packages, this.text);

  final List<String> packages;
  final String text;
}

/// Reads the "License texts" part of THIRD_PARTY_NOTICES.md: a "### Used by: a, b"
/// line, then the text, for each distinct license text.
List<NoticeGroup> parseNotices(String markdown) {
  final start = markdown.indexOf('## License texts');
  if (start < 0) return const [];
  final groups = <NoticeGroup>[];
  final parts = markdown.substring(start).split(RegExp(r'^### Used by: ', multiLine: true));
  for (final part in parts.skip(1)) {
    final newline = part.indexOf('\n');
    if (newline < 0) continue;
    final packages = part.substring(0, newline).split(', ').map((p) => p.trim()).where((p) => p.isNotEmpty).toList();
    final text = part.substring(newline + 1).trim();
    if (packages.isNotEmpty && text.isNotEmpty) groups.add(NoticeGroup(packages, text));
  }
  return groups;
}

class NoticesScreen extends StatefulWidget {
  const NoticesScreen({super.key, this.load});

  /// Where the text comes from; the bundled asset unless a test says otherwise.
  final Future<String> Function()? load;

  @override
  State<NoticesScreen> createState() => _NoticesScreenState();
}

class _NoticesScreenState extends State<NoticesScreen> {
  late final Future<List<NoticeGroup>> _groups = (widget.load ?? _bundled)().then(parseNotices);

  static Future<String> _bundled() => rootBundle.loadString('assets/third_party_notices.md');

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    return Scaffold(
      appBar: AppBar(title: Text(l.noticesTitle)),
      body: FutureBuilder<List<NoticeGroup>>(
        future: _groups,
        builder: (context, snapshot) {
          if (snapshot.connectionState != ConnectionState.done) {
            return const Center(child: CircularProgressIndicator());
          }
          final groups = snapshot.data;
          if (groups == null || groups.isEmpty) return Center(child: Text(l.noticesLoadFailed));
          final total = groups.fold<int>(0, (n, g) => n + g.packages.length);
          return ListView.builder(
            itemCount: groups.length + 1,
            itemBuilder: (context, i) {
              if (i == 0) {
                return Padding(
                  padding: const EdgeInsets.all(16),
                  child: Text('${l.noticesIntro}\n${l.noticesPackages(total)}'),
                );
              }
              return _GroupTile(groups[i - 1]);
            },
          );
        },
      ),
    );
  }
}

class _GroupTile extends StatelessWidget {
  const _GroupTile(this.group);

  final NoticeGroup group;

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final shown = group.packages.take(2).join(', ');
    final more = group.packages.length - 2;
    return ExpansionTile(
      title: Directionality(textDirection: TextDirection.ltr, child: Text(shown)),
      subtitle: more > 0 ? Text(l.noticesMore(more)) : null,
      expandedCrossAxisAlignment: CrossAxisAlignment.start,
      childrenPadding: const EdgeInsets.fromLTRB(16, 0, 16, 16),
      children: [
        if (group.packages.length > 2)
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: SelectableText(group.packages.join(', '), textDirection: TextDirection.ltr),
          ),
        SelectableText(
          group.text,
          textDirection: TextDirection.ltr,
          style: const TextStyle(fontFamily: 'monospace', fontSize: 11),
        ),
      ],
    );
  }
}
