import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:omnitun/state/incoming.dart';

void main() {
  test('what the command line asks to open: links as they are, files by their contents', () async {
    final tmp = Directory.systemTemp.createTempSync('incoming');
    addTearDown(() => tmp.deleteSync(recursive: true));
    final file = File('${tmp.path}/servers.txt')..writeAsStringSync('trojan://pw@a.example.com:443#A\n');
    final big = File('${tmp.path}/big.bin')..writeAsBytesSync(List.filled(IncomingShare.maxBytes + 1, 65));

    final got = await IncomingShare.fromArguments([
      '--some-flag', // flags are not things to open
      'vless://u@h.example.com:443?type=tcp#V',
      file.path,
      Uri.file(file.path).toString(), // file:// as some file managers pass it
      big.path, // too big to be a config
      '${tmp.path}/missing.ovpn',
      'plain words',
    ]);
    expect(got, [
      'vless://u@h.example.com:443?type=tcp#V',
      'trojan://pw@a.example.com:443#A\n',
      'trojan://pw@a.example.com:443#A\n',
    ]);
  });
}
