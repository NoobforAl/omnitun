#!/usr/bin/env python3
"""Writes THIRD_PARTY_NOTICES.md: the licenses of everything that ships in Omnitun.

  Go:    the modules linked into cmd/* and core/mobile (with the project's build tags)
  Dart:  the packages in app/pubspec.lock (hosted ones, from the pub cache)
  Rust:  the crates of tor/ (cargo metadata)

Usage: python scripts/gen-notices.py [--out FILE] [--check]
  --check   only report; exit 1 if a license could not be identified
Run it from the repository root with go, flutter (pub cache) and cargo available.
"""
import argparse
import hashlib
import json
import os
import re
import subprocess
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
TAGS = "with_gvisor,with_quic,with_wireguard,with_utls,with_clash_api,with_openvpn,with_openconnect,with_naive_outbound,with_acme,badlinkname,tfogo_checklinkname0"

LICENSE_FILES = re.compile(r"^(licen[sc]e|copying|notice|unlicense)([-_.].*)?$", re.I)

# Licenses recognised by the title at the top of the file (a longer text may mention others).
TITLES = [
    ("Apache-2.0", r"^(.{0,80})?apache license.{0,40}version 2\.0"),
    ("MPL-2.0", r"^(.{0,40})?mozilla public license.{0,40}(version )?2\.0"),
    ("AGPL-3.0", r"^(.{0,40})?gnu affero general public license"),
    ("LGPL-3.0", r"^(.{0,40})?gnu lesser general public license.{0,60}version 3"),
    ("LGPL-2.1", r"^(.{0,40})?gnu lesser general public license.{0,60}version 2\.1"),
    ("GPL-3.0", r"^(.{0,40})?gnu general public license.{0,60}version 3"),
    ("GPL-2.0", r"^(.{0,40})?gnu general public license.{0,60}version 2"),
]
# Permissive licenses, recognised by their wording.
SIGNATURES = [
    ("GPL-3.0", r"free software: you can redistribute it and/or modify it under the terms of the gnu general public license as published by the free software foundation, either version 3"),
    ("ISC", r"permission to use, copy, modify,? (and/or|and) distribute this software for any purpose with or without fee"),
    ("MIT", r"permission is hereby granted, free of charge, to any person obtaining a copy"),
    ("BSD-3-Clause", r"redistributions of source code must retain.*redistributions in binary form must reproduce.*(neither the name|the names? of)"),
    ("BSD-2-Clause", r"redistributions of source code must retain.*redistributions in binary form must reproduce"),
    ("Unlicense", r"this is free and unencumbered software released into the public domain"),
    ("Zlib", r"this software is provided 'as-is', without any express or implied warranty"),
    ("CC0-1.0", r"creative commons.{0,40}cc0"),
]

# Packages whose module ships no license file; the repository's declared license, checked by hand.
OVERRIDES = {
    "github.com/mattn/go-localereader": ("MIT", "MIT (the module ships no license file; the upstream repository is MIT licensed)"),
}


def norm(text):
    return re.sub(r"\s+", " ", text.lower()).strip()


def guess(text):
    t = norm(text)
    head = t[:500]
    for spdx, rx in TITLES:
        if re.search(rx, head):
            return spdx
    for spdx, rx in SIGNATURES:
        if re.search(rx, t):
            return spdx
    return ""


def run(cmd, cwd=ROOT, env=None):
    e = dict(os.environ)
    if env:
        e.update(env)
    return subprocess.run(cmd, cwd=cwd, env=e, capture_output=True, text=True, encoding="utf-8", check=True).stdout


def license_files(directory):
    out = []
    if not directory or not os.path.isdir(directory):
        return out
    for name in sorted(os.listdir(directory)):
        full = os.path.join(directory, name)
        if os.path.isfile(full) and LICENSE_FILES.match(name):
            try:
                out.append((name, open(full, encoding="utf-8", errors="replace").read()))
            except OSError:
                pass
    return out


def go_modules():
    out = run(
        ["go", "list", "-deps", "-e", "-tags", TAGS, "-f", "{{with .Module}}{{if not .Main}}{{.Path}}\t{{.Version}}\t{{.Dir}}{{end}}{{end}}",
         "./cmd/...", "./core/mobile"],
    )
    seen = {}
    for line in out.splitlines():
        if not line.strip():
            continue
        path, version, directory = line.split("\t")
        seen[path] = (version, directory)
    # The vendored copy of veepin is a replace directive; go list reports its directory.
    return [(p, v, d) for p, (v, d) in sorted(seen.items())]


def dart_packages():
    lock = os.path.join(ROOT, "app", "pubspec.lock")
    text = open(lock, encoding="utf-8").read()
    cache = os.environ.get("PUB_CACHE") or os.path.join(os.environ.get("LOCALAPPDATA", os.path.expanduser("~")), "Pub", "Cache")
    if not os.path.isdir(os.path.join(cache, "hosted")):
        cache = os.path.join(os.path.expanduser("~"), ".pub-cache")
    pkgs = []
    for m in re.finditer(r"^  (\S+):\n    dependency: \"([^\"]+)\"\n(?:.*\n)*?    source: (\w+)\n    version: \"([^\"]+)\"", text, re.M):
        name, dep, source, version = m.groups()
        if dep == "direct dev" or source != "hosted":
            continue
        directory = os.path.join(cache, "hosted", "pub.dev", "%s-%s" % (name, version))
        pkgs.append((name, version, directory))
    return pkgs


def rust_crates():
    try:
        meta = json.loads(run(["cargo", "metadata", "--format-version", "1", "--locked"], cwd=os.path.join(ROOT, "tor")))
    except Exception as e:  # noqa: BLE001
        print("cargo metadata failed:", e, file=sys.stderr)
        return []
    out = []
    for p in meta["packages"]:
        if p.get("source") is None:
            continue
        out.append((p["name"], p["version"], os.path.dirname(p["manifest_path"]), p.get("license") or ""))
    return sorted(out)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", default=os.path.join(ROOT, "THIRD_PARTY_NOTICES.md"))
    ap.add_argument("--check", action="store_true")
    args = ap.parse_args()

    entries = []  # (ecosystem, name, version, spdx, [(file, text)])
    unknown = []
    for eco, items in (("Go", go_modules()), ("Dart", dart_packages())):
        for name, version, directory in items:
            files = license_files(directory)
            spdx = ""
            for _, text in files:
                spdx = guess(text) or spdx
                if spdx:
                    break
            if not files and name in OVERRIDES:
                spdx, note = OVERRIDES[name]
                files = [("(override)", note)]
            if not files:
                unknown.append("%s %s@%s: no license file in %s" % (eco, name, version, directory))
            elif not spdx:
                unknown.append("%s %s@%s: license not recognised" % (eco, name, version))
            entries.append((eco, name, version, spdx or "?", files))
    for name, version, directory, expr in rust_crates():
        files = license_files(directory)
        entries.append(("Rust", name, version, expr or "?", files))
        if not files and not expr:
            unknown.append("Rust %s@%s: no license" % (name, version))

    # Summary by license.
    counts = {}
    for _, _, _, spdx, _ in entries:
        counts[spdx] = counts.get(spdx, 0) + 1
    lines = ["# Third-party notices", "",
             "Omnitun is MIT licensed. It includes the software below under their own licenses. "
             "Generated by `scripts/gen-notices.py`; do not edit.", "", "## Summary", "", "| License | Packages |", "|---|---|"]
    for spdx, n in sorted(counts.items(), key=lambda x: (-x[1], x[0])):
        lines.append("| %s | %d |" % (spdx, n))
    lines += ["", "## Packages", "", "| Ecosystem | Package | Version | License |", "|---|---|---|---|"]
    for eco, name, version, spdx, _ in entries:
        lines.append("| %s | %s | %s | %s |" % (eco, name, version, spdx))

    # Full texts, each distinct text once.
    texts = {}
    for eco, name, version, spdx, files in entries:
        # A LICENSE file that only says "MIT OR Apache-2.0" points at the real texts next to it.
        files = [f for f in files if len(f[1].strip()) >= 40] or files
        for fname, text in files:
            key = hashlib.sha256(norm(text).encode()).hexdigest()
            texts.setdefault(key, (text, []))[1].append("%s %s %s" % (eco, name, version))
    lines += ["", "## License texts", ""]
    for key, (text, users) in sorted(texts.items(), key=lambda x: x[1][1][0]):
        lines += ["### Used by: " + ", ".join(users[:6]) + (" and %d more" % (len(users) - 6) if len(users) > 6 else ""), "", "```text", text.strip(), "```", ""]
    out = "\n".join(lines) + "\n"
    if not args.check:
        open(args.out, "w", encoding="utf-8", newline="\n").write(out)
    print("%d packages (%s), %d distinct license texts, %d KB" % (
        len(entries), ", ".join("%s %d" % kv for kv in sorted(counts.items(), key=lambda x: -x[1])[:8]), len(texts), len(out) // 1024))
    for u in unknown:
        print("  ?", u)
    return 1 if (args.check and unknown) else 0


if __name__ == "__main__":
    sys.exit(main())
