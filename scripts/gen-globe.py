"""Builds the globe's data from Natural Earth (public domain):
  - a 1.5 degree land/sea grid (bit-packed, base64) for the dotted Earth,
  - a centre (lat, lon) for every country, to point the globe at the exit country.
Usage: make_geo.py land.geojson countries.geojson out.dart"""
import base64
import json
import sys

from PIL import Image, ImageDraw

land = json.load(open(sys.argv[1], encoding="utf-8"))
countries = json.load(open(sys.argv[2], encoding="utf-8"))

STEP = 1.5
W, H = int(360 / STEP), int(180 / STEP)  # 240 x 120
SCALE = 8                                 # rasterise finer, then vote per cell
img = Image.new("L", (W * SCALE, H * SCALE), 0)
d = ImageDraw.Draw(img)


def polys(geom):
    if geom["type"] == "Polygon":
        return [geom["coordinates"]]
    if geom["type"] == "MultiPolygon":
        return geom["coordinates"]
    return []


def px(lon, lat):
    return ((lon + 180) / 360 * W * SCALE, (90 - lat) / 180 * H * SCALE)


for f in land["features"]:
    for poly in polys(f["geometry"]):
        d.polygon([px(*p[:2]) for p in poly[0]], fill=255)
        for hole in poly[1:]:
            d.polygon([px(*p[:2]) for p in hole], fill=0)

small = img.resize((W, H), Image.BOX)
bits = []
for y in range(H):
    for x in range(W):
        bits.append(1 if small.getpixel((x, y)) >= 110 else 0)
while len(bits) % 8:
    bits.append(0)
packed = bytearray()
for i in range(0, len(bits), 8):
    v = 0
    for b in bits[i:i + 8]:
        v = (v << 1) | b
    packed.append(v)
b64 = base64.b64encode(bytes(packed)).decode()
print("land cells:", sum(bits), "of", W * H, "bytes:", len(packed))


def area_centroid(ring):
    a = cx = cy = 0.0
    for (x0, y0), (x1, y1) in zip(ring, ring[1:] + ring[:1]):
        c = x0 * y1 - x1 * y0
        a += c
        cx += (x0 + x1) * c
        cy += (y0 + y1) * c
    if abs(a) < 1e-9:
        return None, 0
    return (cx / (3 * a), cy / (3 * a)), abs(a) / 2


centers = {}
for f in countries["features"]:
    p = f["properties"]
    code = p.get("ISO_A2")
    if not code or code == "-99":
        code = p.get("ISO_A2_EH")
    if not code or code == "-99":
        continue
    best, best_area = None, 0
    for poly in polys(f["geometry"]):
        c, a = area_centroid([tuple(pt[:2]) for pt in poly[0]])
        if c and a > best_area:
            best, best_area = c, a
    if best:
        centers[code] = (round(best[1], 1), round(best[0], 1))

# Small states the 110 m map leaves out: approximate centres.
extra = {"SG": (1.35, 103.82), "HK": (22.3, 114.17), "MT": (35.9, 14.4), "MC": (43.74, 7.42), "LI": (47.16, 9.55),
         "AD": (42.55, 1.6), "SM": (43.94, 12.46), "BH": (26.07, 50.55), "MV": (3.2, 73.2), "MU": (-20.3, 57.6),
         "BB": (13.19, -59.54), "SC": (-4.68, 55.49), "LU": (49.8, 6.1), "VA": (41.9, 12.45),
         "AG": (17.06, -61.8), "CV": (16.0, -24.0), "DM": (15.4, -61.37), "GD": (12.1, -61.7), "KI": (1.45, 173.0),
         "KM": (-11.7, 43.3), "KN": (17.3, -62.7), "LC": (13.9, -60.98), "MO": (22.2, 113.55), "TO": (-21.18, -175.2),
         "TW": (23.7, 121.0), "VC": (13.25, -61.2), "WS": (-13.76, -172.1)}
for k, v in extra.items():
    centers.setdefault(k, v)

with open(sys.argv[3], "w", encoding="utf-8", newline="\n") as out:
    out.write("// Generated from Natural Earth (public domain): the land grid of the globe and the centre of each\n")
    out.write("// country. Not edited by hand.\n\n")
    out.write(f"/// Land (1) or sea (0) for every {STEP}-degree cell, row by row from 90N, as packed bits (base64).\n")
    out.write(f"const landGridWidth = {W};\nconst landGridHeight = {H};\nconst landGridStepDegrees = {STEP};\n")
    out.write("const landGridBase64 =\n")
    for i in range(0, len(b64), 96):
        out.write(f"    '{b64[i:i + 96]}'\n")
    out.write("    ;\n\n")
    out.write("/// (latitude, longitude) of the middle of each country, by ISO 3166-1 alpha-2 code.\n")
    out.write("const countryCenters = <String, (double, double)>{\n")
    for k in sorted(centers):
        out.write(f"  '{k}': ({centers[k][0]}, {centers[k][1]}),\n")
    out.write("};\n")
print("countries:", len(centers))
