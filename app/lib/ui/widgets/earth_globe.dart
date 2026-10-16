import 'dart:convert';
import 'dart:math' as math;
import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:flutter/material.dart';

import '../../state/geo_data.dart';

/// The land of the Earth as unit vectors, one per land cell of the grid
/// (x east, y north, z towards the viewer when the cell is at 0°N 0°E).
class _Land {
  _Land._(this.xyz);

  final Float32List xyz;

  int get count => xyz.length ~/ 3;

  static final _Land instance = _Land._(_decode());

  static Float32List _decode() {
    final bytes = base64Decode(landGridBase64);
    final out = <double>[];
    for (var row = 0; row < landGridHeight; row++) {
      final lat = (90 - (row + 0.5) * landGridStepDegrees) * math.pi / 180;
      for (var col = 0; col < landGridWidth; col++) {
        final i = row * landGridWidth + col;
        if ((bytes[i >> 3] >> (7 - (i & 7))) & 1 == 0) continue;
        final lon = ((col + 0.5) * landGridStepDegrees - 180) * math.pi / 180;
        out
          ..add(math.cos(lat) * math.sin(lon))
          ..add(math.sin(lat))
          ..add(math.cos(lat) * math.cos(lon));
      }
    }
    return Float32List.fromList(out);
  }
}

/// Where the globe looks and what it marks.
@immutable
class _View {
  const _View(this.lat, this.lon, this.markLat, this.markLon, this.mark);

  /// The point in the middle of the globe, degrees.
  final double lat, lon;

  /// The exit location and how visible its marker is (0-1).
  final double markLat, markLon, mark;

  _View lerp(_View o, double t) => _View(
    lat + (o.lat - lat) * t,
    lon + (o.lon - lon) * t,
    o.markLat,
    o.markLon,
    mark + (o.mark - mark) * t,
  );
}

class _ViewTween extends Tween<_View> {
  _ViewTween({super.end});

  @override
  _View lerp(double t) => begin!.lerp(end!, t);
}

/// A dotted Earth behind the home screen. When the VPN is connected it turns to
/// the exit country and marks it with a glow; otherwise it rests. It moves only
/// while turning (about a second and a half), so it costs next to nothing.
class EarthBackdrop extends StatefulWidget {
  const EarthBackdrop({super.key, this.target, required this.markerColor});

  /// (latitude, longitude) of the exit, or null when not connected or unknown.
  final (double, double)? target;
  final Color markerColor;

  @override
  State<EarthBackdrop> createState() => _EarthBackdropState();
}

class _EarthBackdropState extends State<EarthBackdrop> {
  /// The globe at rest: Europe, Africa and the Middle East.
  static const _restLat = 18.0, _restLon = 20.0;

  /// The exit is shown this many degrees above the middle, clear of the power ring.
  static const _lift = 34.0;

  double _lon = _restLon;
  double _markLat = 0, _markLon = 0;

  _View _view() {
    final t = widget.target;
    if (t == null) return _View(_restLat, _lon, _markLat, _markLon, 0);
    _markLat = t.$1;
    _markLon = t.$2;
    // Turn the short way round.
    _lon += ((t.$2 - _lon + 540) % 360) - 180;
    return _View((t.$1 - _lift).clamp(-60.0, 60.0), _lon, t.$1, t.$2, 1);
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final dark = scheme.brightness == Brightness.dark;
    final view = _view();
    return IgnorePointer(
      child: RepaintBoundary(
        child: TweenAnimationBuilder<_View>(
          tween: _ViewTween(end: view),
          duration: const Duration(milliseconds: 1500),
          curve: Curves.easeInOutCubic,
          builder: (context, v, _) => CustomPaint(
            painter: _GlobePainter(
              view: v,
              ocean: dark ? const Color(0xFF0B1422) : const Color(0xFFD3DCE8),
              oceanDeep: dark ? const Color(0xFF060A12) : const Color(0xFFC3CEDD),
              land: dark ? const Color(0xFF7FA3D6) : const Color(0xFF4B6A99),
              rim: dark ? const Color(0xFF86A8D8) : const Color(0xFF6E8CB8),
              marker: widget.markerColor,
            ),
          ),
        ),
      ),
    );
  }
}

class _GlobePainter extends CustomPainter {
  _GlobePainter({
    required this.view,
    required this.ocean,
    required this.oceanDeep,
    required this.land,
    required this.rim,
    required this.marker,
  });

  final _View view;
  final Color ocean, oceanDeep, land, rim, marker;

  static const _deg = math.pi / 180;

  @override
  void paint(Canvas canvas, Size size) {
    final r = size.shortestSide / 2;
    final c = size.center(Offset.zero);

    // The sea: a dark sphere, lit from above on the left.
    canvas.drawCircle(
      c,
      r,
      Paint()
        ..shader = ui.Gradient.radial(
          c + Offset(-r * 0.35, -r * 0.4),
          r * 1.5,
          [ocean, oceanDeep],
        ),
    );

    final cosLon = math.cos(view.lon * _deg), sinLon = math.sin(view.lon * _deg);
    final cosLat = math.cos(view.lat * _deg), sinLat = math.sin(view.lat * _deg);
    final land3 = _Land.instance;
    final n = land3.count;
    // Dots in three brightness steps: the rim of the globe is dimmer.
    final near = Float32List(n * 2), mid = Float32List(n * 2), far = Float32List(n * 2);
    var nn = 0, nm = 0, nf = 0;
    final v = land3.xyz;
    for (var i = 0; i < n; i++) {
      final x = v[i * 3], y = v[i * 3 + 1], z = v[i * 3 + 2];
      final x1 = x * cosLon - z * sinLon;
      final z1 = x * sinLon + z * cosLon;
      final y2 = y * cosLat - z1 * sinLat;
      final z2 = y * sinLat + z1 * cosLat;
      if (z2 <= 0.02) continue;
      final px = c.dx + r * x1, py = c.dy - r * y2;
      if (z2 > 0.6) {
        near[nn++] = px;
        near[nn++] = py;
      } else if (z2 > 0.28) {
        mid[nm++] = px;
        mid[nm++] = py;
      } else {
        far[nf++] = px;
        far[nf++] = py;
      }
    }
    final dot = (r / 150).clamp(1.6, 3.4);
    void draw(Float32List pts, int len, double alpha) {
      if (len == 0) return;
      canvas.drawRawPoints(
        ui.PointMode.points,
        Float32List.sublistView(pts, 0, len),
        Paint()
          ..color = land.withValues(alpha: alpha)
          ..strokeWidth = dot
          ..strokeCap = StrokeCap.round,
      );
    }

    draw(far, nf, 0.20);
    draw(mid, nm, 0.38);
    draw(near, nn, 0.58);

    // A cold halo on the rim: the air.
    canvas.drawCircle(
      c,
      r,
      Paint()
        ..style = PaintingStyle.stroke
        ..strokeWidth = 1.5
        ..color = rim.withValues(alpha: 0.45),
    );
    canvas.drawCircle(
      c,
      r + 3,
      Paint()
        ..style = PaintingStyle.stroke
        ..strokeWidth = 10
        ..color = rim.withValues(alpha: 0.10)
        ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 10),
    );

    // The exit: a glow, a ring and a point.
    if (view.mark > 0.01) {
      final lat = view.markLat * _deg, lon = view.markLon * _deg;
      final x = math.cos(lat) * math.sin(lon), y = math.sin(lat), z = math.cos(lat) * math.cos(lon);
      final x1 = x * cosLon - z * sinLon;
      final z1 = x * sinLon + z * cosLon;
      final y2 = y * cosLat - z1 * sinLat;
      final z2 = y * sinLat + z1 * cosLat;
      if (z2 > 0.05) {
        final p = Offset(c.dx + r * x1, c.dy - r * y2);
        final a = view.mark;
        final s = (r / 360).clamp(0.6, 1.2);
        canvas.drawCircle(
          p,
          46 * s,
          Paint()..shader = ui.Gradient.radial(p, 46 * s, [marker.withValues(alpha: 0.7 * a), marker.withValues(alpha: 0)]),
        );
        canvas.drawCircle(
          p,
          13 * s,
          Paint()
            ..style = PaintingStyle.stroke
            ..strokeWidth = 1.6
            ..color = marker.withValues(alpha: 0.9 * a),
        );
        canvas.drawCircle(p, 4.2 * s, Paint()..color = Color.lerp(marker, Colors.white, 0.5)!.withValues(alpha: a));
      }
    }
  }

  @override
  bool shouldRepaint(_GlobePainter old) =>
      old.view.lat != view.lat ||
      old.view.lon != view.lon ||
      old.view.mark != view.mark ||
      old.view.markLat != view.markLat ||
      old.view.markLon != view.markLon ||
      old.marker != marker ||
      old.land != land ||
      old.ocean != ocean;
}
