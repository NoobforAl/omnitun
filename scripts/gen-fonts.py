"""Cuts static, subsetted weights out of the variable OFL fonts (Cinzel, Inter, Vazirmatn)."""
import os
import shutil
import sys

from fontTools import subset
from fontTools.ttLib import TTFont
from fontTools.varLib import instancer

SRC = sys.argv[1]   # folder with the downloaded variable fonts and OFL texts
OUT = sys.argv[2]   # app/assets/fonts

LATIN = list(range(0x20, 0x7F)) + list(range(0xA0, 0x180)) + [0x2013, 0x2014, 0x2018, 0x2019, 0x201C, 0x201D,
        0x2022, 0x2026, 0x2190, 0x2191, 0x2192, 0x2193, 0x20AC, 0x2212, 0x00B7, 0x2192]
CYRILLIC = list(range(0x400, 0x500))
ARABIC = (list(range(0x600, 0x700)) + list(range(0x750, 0x780)) + list(range(0xFB50, 0xFE00))
          + list(range(0xFE70, 0xFF00)) + [0x200C, 0x200D, 0x200E, 0x200F, 0x20, 0xA0, 0x2026, 0x00B7]
          + list(range(0x30, 0x3A)))

JOBS = [
    ("Cinzel", {"wght": [500, 700]}, LATIN, {}),
    ("Inter", {"wght": [400, 500, 600, 700]}, LATIN + CYRILLIC, {"opsz": 14}),
    ("Vazirmatn", {"wght": [400, 500, 600, 700]}, ARABIC, {}),
]

os.makedirs(OUT, exist_ok=True)
for family, axes, unicodes, pinned in JOBS:
    for w in axes["wght"]:
        font = TTFont(os.path.join(SRC, f"{family}.ttf"))
        font = instancer.instantiateVariableFont(font, {"wght": w, **pinned})
        tmp = os.path.join(OUT, f"_{family}-{w}.ttf")
        font.save(tmp)
        opts = subset.Options()
        opts.layout_features = ["*"]
        opts.name_IDs = [1, 2, 3, 4, 6]
        opts.notdef_outline = True
        f = subset.load_font(tmp, opts)
        s = subset.Subsetter(opts)
        s.populate(unicodes=unicodes)
        s.subset(f)
        dest = os.path.join(OUT, f"{family}-{w}.ttf")
        subset.save_font(f, dest, opts)
        f.close()
        font.close()
        os.remove(tmp)
        print(dest, os.path.getsize(dest) // 1024, "KB")

for fam in ("cinzel", "inter", "vazirmatn"):
    shutil.copy(os.path.join(SRC, f"OFL-{fam}.txt"), os.path.join(OUT, f"OFL-{fam}.txt"))
