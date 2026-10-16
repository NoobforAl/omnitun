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

/// What the globe shows at one moment: where it looks, and the two places it may mark —
/// where you are (red while unprotected) and where you exit (once connecting or connected).
@immutable
class _Scene {
  const _Scene({
    required this.lat,
    required this.lon,
    this.home,
    this.exit,
    this.homeAlpha = 0,
    this.exitAlpha = 0,
    required this.homeColor,
    required this.exitColor,
  });

  /// The point in the middle of the globe, degrees.
  final double lat, lon;
  final (double, double)? home, exit;

  /// How visible each marker is (0-1).
  final double homeAlpha, exitAlpha;
  final Color homeColor, exitColor;

  bool same(_Scene o) =>
      lat == o.lat &&
      lon == o.lon &&
      home == o.home &&
      exit == o.exit &&
      homeAlpha == o.homeAlpha &&
      exitAlpha == o.exitAlpha &&
      homeColor == o.homeColor &&
      exitColor == o.exitColor;

  /// In between [a] and [b]. A marker sits where [b] has it (or where [a] had it while it fades out).
  static _Scene lerp(_Scene a, _Scene b, double t) => _Scene(
    lat: a.lat + (b.lat - a.lat) * t,
    lon: a.lon + (b.lon - a.lon) * t,
    home: b.home ?? a.home,
    exit: b.exit ?? a.exit,
    homeAlpha: a.homeAlpha + (b.homeAlpha - a.homeAlpha) * t,
    exitAlpha: a.exitAlpha + (b.exitAlpha - a.exitAlpha) * t,
    homeColor: Color.lerp(a.homeColor, b.homeColor, t)!,
    exitColor: Color.lerp(a.exitColor, b.exitColor, t)!,
  );
}

/// A dotted Earth behind the home screen, in the manner of Mullvad's: it shows where you
/// are while unprotected, and on connecting it flies (zooming out on the way) to where you
/// exit, with an arc between the two; on disconnecting it flies back. It only moves during
/// the flight (about two seconds), so it costs next to nothing the rest of the time.
class EarthBackdrop extends StatefulWidget {
  const EarthBackdrop({super.key, this.home, this.exit, required this.homeColor, required this.exitColor});

  /// (latitude, longitude) of where you are, and of the exit; null when not known / not connecting.
  final (double, double)? home, exit;
  final Color homeColor, exitColor;

  @override
  State<EarthBackdrop> createState() => _EarthBackdropState();
}

class _EarthBackdropState extends State<EarthBackdrop> with SingleTickerProviderStateMixin {
  /// The globe at rest, when no place is known: Europe, Africa and the Middle East.
  static const _restLat = 18.0, _restLon = 20.0;

  /// The marked place is shown this many degrees above the middle, clear of the power ring.
  static const _lift = 14.0;

  late final AnimationController _fly = AnimationController(vsync: this, duration: const Duration(milliseconds: 2000));
  late _Scene _from;
  late _Scene _to;

  _Scene _target(double previousLon) {
    final home = widget.home, exit = widget.exit;
    final focus = exit ?? home;
    return _Scene(
      lat: focus == null ? _restLat : (focus.$1 - _lift).clamp(-60.0, 60.0),
      // Turn the short way round.
      lon: focus == null ? previousLon : previousLon + ((focus.$2 - previousLon + 540) % 360) - 180,
      home: home,
      exit: exit,
      // Once there is an exit, where you are fades back and the exit takes over.
      homeAlpha: home == null ? 0 : (exit == null ? 1 : 0.55),
      exitAlpha: exit == null ? 0 : 1,
      homeColor: widget.homeColor,
      exitColor: widget.exitColor,
    );
  }

  @override
  void initState() {
    super.initState();
    _to = _target(_restLon);
    _from = _to;
    _fly.value = 1;
  }

  @override
  void didUpdateWidget(EarthBackdrop old) {
    super.didUpdateWidget(old);
    // Start from what is on screen now, so a change in mid-flight does not jump.
    final now = _Scene.lerp(_from, _to, Curves.easeInOutCubic.transform(_fly.value));
    final next = _target(now.lon);
    if (next.same(_to)) return;
    _from = now;
    _to = next;
    _fly.forward(from: 0);
  }

  @override
  void dispose() {
    _fly.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final dark = Theme.of(context).colorScheme.brightness == Brightness.dark;
    return IgnorePointer(
      child: RepaintBoundary(
        child: AnimatedBuilder(
          animation: _fly,
          builder: (context, _) {
            final t = _fly.value;
            return CustomPaint(
              painter: _GlobePainter(
                scene: _Scene.lerp(_from, _to, Curves.easeInOutCubic.transform(t)),
                // Pull back in the middle of a flight, then settle: the camera "travels".
                zoom: 1 - 0.18 * math.sin(math.pi * t),
                ocean: dark ? const Color(0xFF0B1422) : const Color(0xFFD3DCE8),
                oceanDeep: dark ? const Color(0xFF060A12) : const Color(0xFFC3CEDD),
                land: dark ? const Color(0xFF7FA3D6) : const Color(0xFF4B6A99),
                rim: dark ? const Color(0xFF86A8D8) : const Color(0xFF6E8CB8),
              ),
            );
          },
        ),
      ),
    );
  }
}

class _GlobePainter extends CustomPainter {
  _GlobePainter({
    required this.scene,
    required this.zoom,
    required this.ocean,
    required this.oceanDeep,
    required this.land,
    required this.rim,
  });

  final _Scene scene;
  final double zoom;
  final Color ocean, oceanDeep, land, rim;

  static const _deg = math.pi / 180;

  @override
  void paint(Canvas canvas, Size size) {
    final r = size.shortestSide / 2 * zoom;
    final c = size.center(Offset.zero);

    // The sea: a dark sphere, lit from above on the left.
    canvas.drawCircle(
      c,
      r,
      Paint()..shader = ui.Gradient.radial(c + Offset(-r * 0.35, -r * 0.4), r * 1.5, [ocean, oceanDeep]),
    );

    final cosLon = math.cos(scene.lon * _deg), sinLon = math.sin(scene.lon * _deg);
    final cosLat = math.cos(scene.lat * _deg), sinLat = math.sin(scene.lat * _deg);

    /// Screen position and depth (z > 0 faces the viewer) of a point at [k] times the globe's radius.
    (Offset, double) project(double x, double y, double z, [double k = 1]) {
      final x1 = x * cosLon - z * sinLon;
      final z1 = x * sinLon + z * cosLon;
      final y2 = y * cosLat - z1 * sinLat;
      final z2 = y * sinLat + z1 * cosLat;
      return (Offset(c.dx + r * k * x1, c.dy - r * k * y2), z2);
    }

    List<double> unit(double latDeg, double lonDeg) {
      final lat = latDeg * _deg, lon = lonDeg * _deg;
      return [math.cos(lat) * math.sin(lon), math.sin(lat), math.cos(lat) * math.cos(lon)];
    }

    final land3 = _Land.instance;
    final n = land3.count;
    // Dots in three brightness steps: the rim of the globe is dimmer.
    final near = Float32List(n * 2), mid = Float32List(n * 2), far = Float32List(n * 2);
    var nn = 0, nm = 0, nf = 0;
    final v = land3.xyz;
    for (var i = 0; i < n; i++) {
      final (p, z) = project(v[i * 3], v[i * 3 + 1], v[i * 3 + 2]);
      if (z <= 0.02) continue;
      if (z > 0.6) {
        near[nn++] = p.dx;
        near[nn++] = p.dy;
      } else if (z > 0.28) {
        mid[nm++] = p.dx;
        mid[nm++] = p.dy;
      } else {
        far[nf++] = p.dx;
        far[nf++] = p.dy;
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

    final home = scene.home, exit = scene.exit;
    final s = (r / 360).clamp(0.6, 1.2);

    // The path between where you are and where you exit: a dotted arc rising above the surface.
    if (home != null && exit != null) {
      final a = math.min(scene.homeAlpha, scene.exitAlpha);
      final h = unit(home.$1, home.$2), e = unit(exit.$1, exit.$2);
      final omega = math.acos((h[0] * e[0] + h[1] * e[1] + h[2] * e[2]).clamp(-1.0, 1.0));
      if (a > 0.01 && omega > 0.02) {
        final paint = Paint()..color = scene.exitColor.withValues(alpha: 0.8 * a);
        const steps = 56;
        for (var i = 0; i <= steps; i += 2) {
          final u = i / steps;
          final k1 = math.sin((1 - u) * omega) / math.sin(omega), k2 = math.sin(u * omega) / math.sin(omega);
          final lift = 1 + 0.16 * math.sin(math.pi * u) * (omega / math.pi).clamp(0.25, 1.0);
          final (p, z) = project(k1 * h[0] + k2 * e[0], k1 * h[1] + k2 * e[1], k1 * h[2] + k2 * e[2], lift);
          if (z > 0) canvas.drawCircle(p, 1.7 * s, paint);
        }
      }
    }

    void mark((double, double)? at, double alpha, Color color, double size) {
      if (at == null || alpha < 0.01) return;
      final u = unit(at.$1, at.$2);
      final (p, z) = project(u[0], u[1], u[2]);
      if (z <= 0.05) return;
      final glow = 46 * s * size;
      canvas.drawCircle(
        p,
        glow,
        Paint()..shader = ui.Gradient.radial(p, glow, [color.withValues(alpha: 0.7 * alpha), color.withValues(alpha: 0)]),
      );
      canvas.drawCircle(
        p,
        13 * s * size,
        Paint()
          ..style = PaintingStyle.stroke
          ..strokeWidth = 1.6
          ..color = color.withValues(alpha: 0.9 * alpha),
      );
      canvas.drawCircle(
        p,
        4.2 * s * size,
        Paint()..color = Color.lerp(color, Colors.white, 0.5)!.withValues(alpha: alpha),
      );
    }

    // Where you are (smaller once you are somewhere else), then where you exit.
    mark(home, scene.homeAlpha, scene.homeColor, exit == null ? 1 : 0.7);
    mark(exit, scene.exitAlpha, scene.exitColor, 1);
  }

  @override
  bool shouldRepaint(_GlobePainter old) => true;
}
