import 'dart:math' as math;

import 'package:flutter/material.dart';

/// A still sky of small stars behind the home screen. Drawn once, never
/// animated, so it costs no battery. In the light theme it is faint gold dust.
class StarFieldPainter extends CustomPainter {
  const StarFieldPainter();

  @override
  void paint(Canvas canvas, Size size) {
    final rnd = math.Random(11); // the same sky every time
    final paint = Paint();
    for (var i = 0; i < 70; i++) {
      final x = rnd.nextDouble() * size.width;
      final y = rnd.nextDouble() * size.height * 0.75;
      final r = 0.5 + rnd.nextDouble() * 1.4;
      final a = 0.15 + rnd.nextDouble() * 0.5;
      // fade out towards the bottom panel
      final fade = 1 - (y / (size.height * 0.75)) * 0.7;
      paint.color = Color.fromRGBO(255, 244, 214, a * fade);
      canvas.drawCircle(Offset(x, y), r, paint);
      if (i % 9 == 0) {
        // a few twinkles: a small cross
        paint
          ..strokeWidth = 0.8
          ..style = PaintingStyle.stroke;
        canvas.drawLine(Offset(x - r * 3, y), Offset(x + r * 3, y), paint);
        canvas.drawLine(Offset(x, y - r * 3), Offset(x, y + r * 3), paint);
        paint.style = PaintingStyle.fill;
      }
    }
  }

  @override
  bool shouldRepaint(StarFieldPainter old) => false;
}
