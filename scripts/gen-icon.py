"""Omnitun app icon v2: an arcane portal gate (gold runed ring, teal-violet vortex, star
at its heart) in a midnight starfield. Writes the Android adaptive icon layers, legacy
PNGs, a Windows .ico, an in-app logo and a 1024 px master."""
import os
import sys

import numpy as np
from PIL import Image, ImageDraw

ROOT = sys.argv[1]
M = 1024          # master size
SS = 2            # supersampling for the masters
rng = np.random.default_rng(7)


def smooth(t):
    t = np.clip(t, 0, 1)
    return t * t * (3 - 2 * t)


def stops(r, xs, cols):
    return np.stack([np.interp(r, xs, [c[i] for c in cols]) for i in range(3)], axis=-1)


def to_img(rgb, a):
    arr = np.dstack([np.clip(rgb, 0, 255), np.clip(a, 0, 1) * 255]).astype(np.uint8)
    return Image.fromarray(arr, "RGBA")


def astroid(x, y, rx, ry):
    """0 outside, 1 inside a four-pointed star (soft edge); also returns the raw field."""
    f = (np.abs(x) / rx) ** (2 / 3) + (np.abs(y) / ry) ** (2 / 3)
    return smooth((1 - f) * 6), f


def portal(N, diam, mono=False):
    """The portal on a transparent canvas of N px; diam = its diameter as a fraction of N."""
    R = N * diam / 2
    yy, xx = np.mgrid[0:N, 0:N]
    x, y = xx + 0.5 - N / 2, yy + 0.5 - N / 2
    r = np.hypot(x, y) / R
    th = np.arctan2(y, x)
    aa = 1.6 / R
    out = Image.new("RGBA", (N, N), (0, 0, 0, 0))

    teal, violet, gold_d, gold_l = (70, 135, 195), (40, 78, 150), (86, 98, 118), (214, 223, 236)

    if not mono:
        # glow around the gate
        t = 0.5 + 0.5 * np.sin(th)
        glow_rgb = np.array(teal)[None, None] * (1 - t[..., None]) + np.array(violet)[None, None] * t[..., None]
        glow_a = 0.62 * np.exp(-np.clip(r - 0.92, 0, None) / 0.26)
        out = Image.alpha_composite(out, to_img(glow_rgb, glow_a))

        # the vortex
        disc = stops(r, [0, 0.12, 0.38, 0.68, 0.88],
                     [(226, 240, 252), (112, 172, 218), (54, 106, 176), (24, 52, 108), (8, 16, 38)])
        swirl = 0.80 + 0.30 * np.sin(3 * th + 9.5 * r) * smooth((r - 0.05) / 0.2)
        disc = disc * swirl[..., None]
        out = Image.alpha_composite(out, to_img(disc, smooth((0.885 - r) / aa)))

    # inner thin teal ring
    ring_in = np.exp(-(((r - 0.862) / 0.012) ** 2))
    if mono:
        out = Image.alpha_composite(out, to_img(np.full((N, N, 3), 255.0), ring_in * 0.9))
    else:
        out = Image.alpha_composite(out, to_img(np.full((N, N, 3), 190.0) * 0 + np.array((150, 205, 235)), ring_in * 0.9))

    # the gold ring with rune notches
    m = smooth((r - 0.885) / aa) * smooth((1.0 - r) / aa)
    seg = 2 * np.pi / 16
    k = np.floor(th / seg).astype(int)
    ph = np.mod(th, seg)
    d = np.abs(ph - seg / 2)
    long_notch = (d < 0.032) & (r > 0.905) & (r < 0.975)
    short_notch = (d < 0.032) & (r > 0.935) & (r < 0.975)
    notch = np.where(k % 2 == 0, long_notch, short_notch)
    if mono:
        a = m * np.where(notch, 0.0, 1.0)
        out = Image.alpha_composite(out, to_img(np.full((N, N, 3), 255.0), a))
    else:
        tt = 0.5 + 0.5 * np.cos(th + 2.356)
        gold = np.array(gold_d)[None, None] * (1 - tt[..., None]) + np.array(gold_l)[None, None] * tt[..., None]
        bevel = 0.80 + 0.30 * np.abs(r - 0.9425) / 0.0575
        gold = gold * bevel[..., None]
        gold = np.where(notch[..., None], gold * 0.30, gold)
        out = Image.alpha_composite(out, to_img(gold, m))

    # the star at the heart
    mask, f = astroid(x, y, R * 0.36, R * 0.36)
    halo = np.exp(-f * 1.6) * (0.55 if not mono else 0.0)
    star_rgb = np.full((N, N, 3), 255.0) * np.array((0.92, 0.96, 1.0))
    out = Image.alpha_composite(out, to_img(star_rgb, halo))
    out = Image.alpha_composite(out, to_img(star_rgb, mask))
    return out


def sparkles(N, pts, mono=False):
    """Small four-pointed stars at fractional positions: (fx, fy, size as a fraction of N)."""
    out = Image.new("RGBA", (N, N), (0, 0, 0, 0))
    for fx, fy, s in pts:
        cx, cy, r = fx * N, fy * N, s * N
        lo, hi = int(max(cx - r * 3, 0)), int(min(cx + r * 3, N))
        lo2, hi2 = int(max(cy - r * 3, 0)), int(min(cy + r * 3, N))
        yy, xx = np.mgrid[lo2:hi2, lo:hi]
        mask, f = astroid(xx + 0.5 - cx, yy + 0.5 - cy, r, r)
        a = np.maximum(mask, np.exp(-f * 2.2) * 0.45)
        rgb = np.full(a.shape + (3,), 255.0) * np.array((0.85, 0.92, 1.0))
        tile = to_img(rgb, a)
        layer = Image.new("RGBA", (N, N), (0, 0, 0, 0))
        layer.paste(tile, (lo, lo2))
        out = Image.alpha_composite(out, layer)
    return out


def background(N, stars=True):
    yy, xx = np.mgrid[0:N, 0:N]
    x, y = (xx + 0.5) / N, (yy + 0.5) / N
    r = np.hypot(x - 0.5, y - 0.45)
    base = stops(r, [0, 0.35, 0.75], [(26, 50, 90), (11, 20, 42), (4, 7, 15)])
    # nebulae
    for cx, cy, s, col, k in [(0.82, 0.86, 0.34, (40, 70, 130), 0.34), (0.14, 0.12, 0.30, (40, 110, 140), 0.24),
                              (0.9, 0.12, 0.25, (50, 70, 140), 0.2)]:
        g = np.exp(-(((x - cx) ** 2 + (y - cy) ** 2) / (2 * s * s))) * k
        base = base * (1 - g[..., None]) + np.array(col)[None, None] * g[..., None]
    img = Image.fromarray(np.clip(base, 0, 255).astype(np.uint8), "RGB").convert("RGBA")
    if stars:
        d = ImageDraw.Draw(img, "RGBA")
        for _ in range(110):
            px, py = rng.random() * N, rng.random() * N
            if np.hypot(px - N / 2, py - N / 2) < N * 0.30:
                continue
            s = (0.6 + rng.random() * 1.6) * N / 512
            a = int(90 + rng.random() * 150)
            d.ellipse((px - s, py - s, px + s, py + s), fill=(220, 232, 255, a))
    return img


def rounded(img, frac):
    N = img.size[0]
    m = Image.new("L", (N * 2, N * 2), 0)
    ImageDraw.Draw(m).rounded_rectangle((0, 0, N * 2 - 1, N * 2 - 1), radius=int(N * 2 * frac), fill=255)
    out = Image.new("RGBA", img.size, (0, 0, 0, 0))
    out.paste(img, (0, 0), m.resize(img.size, Image.LANCZOS))
    return out


def down(img, n):
    return img.resize((n, n), Image.LANCZOS)


def save(img, *parts):
    path = os.path.join(ROOT, *parts)
    os.makedirs(os.path.dirname(path), exist_ok=True)
    img.save(path)
    print("wrote", os.path.relpath(path, ROOT))


N = M * SS
SPARK = [(0.18, 0.2, 0.030), (0.84, 0.17, 0.022), (0.88, 0.7, 0.026), (0.12, 0.76, 0.02), (0.7, 0.9, 0.016)]
SPARK_FG = [(0.30, 0.30, 0.020), (0.72, 0.30, 0.015), (0.74, 0.70, 0.018), (0.28, 0.72, 0.014)]

# adaptive layers: 108 dp canvas, content inside the central 66 dp
fg = down(Image.alpha_composite(portal(N, 0.50), sparkles(N, SPARK_FG)), M)
mono = down(Image.alpha_composite(portal(N, 0.50, mono=True), sparkles(N, SPARK_FG, True)), M)
bg = down(background(N), M).convert("RGBA")

# legacy / store icon: everything together, the portal bigger
portal_big = Image.alpha_composite(portal(N, 0.70), sparkles(N, SPARK))
legacy = down(Image.alpha_composite(background(N), portal_big), M)

res = ("app", "android", "app", "src", "main", "res")
for name, k in {"mdpi": 1, "hdpi": 1.5, "xhdpi": 2, "xxhdpi": 3, "xxxhdpi": 4}.items():
    d = f"mipmap-{name}"
    save(down(rounded(legacy, 0.22), int(48 * k)), *res, d, "ic_launcher.png")
    save(down(rounded(legacy, 0.5), int(48 * k)), *res, d, "ic_launcher_round.png")
    save(down(fg, int(108 * k)), *res, d, "ic_launcher_foreground.png")
    save(down(mono, int(108 * k)), *res, d, "ic_launcher_monochrome.png")
    save(down(bg.convert("RGB"), int(108 * k)), *res, d, "ic_launcher_background.png")

master = rounded(legacy, 0.22)
save(master, "docs", "icon.png")
save(down(legacy, 256), "app", "assets", "logo.png")
ico = os.path.join(ROOT, "app", "windows", "runner", "resources", "app_icon.ico")
master.save(ico, format="ICO", sizes=[(s, s) for s in (16, 24, 32, 48, 64, 128, 256)])
print("wrote ico")
