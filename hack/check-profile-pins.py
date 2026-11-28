#!/usr/bin/env python3
"""Every template a deployed profile serves is pinned to a released module.

A profile entry floating on `refs/heads/main` of stuttgart-things/kcl lets a
kcl merge go live on every claim-machinery-api within minutes (the profile
watcher re-hashes template URLs every RELOAD_INTERVAL), past any review here
or in the cluster repo (#153). So each entry must be

    https://raw.githubusercontent.com/stuttgart-things/kcl/<module>-<X.Y.Z>/<path>

-- the git tag kcl sets when it publishes ghcr.io/stuttgart-things/<module>:<X.Y.Z>
(kcl#413) -- and the template behind it must render that same package:
`spec.source: oci://ghcr.io/stuttgart-things/<module>`, `spec.tag: <X.Y.Z>`.
Branch refs, bare SHAs and local paths are refused; Renovate bumps the tags.

Checked: profiles/*.yaml and the built-in profile of the kustomize artifact
(tests/kcl-deploy-profile.yaml, key config.templateProfiles). Not checked:
tests/profile.yaml, the local `go run` default, never deployed.

Usage: hack/check-profile-pins.py [--offline] [file...]
  --offline  only check the URL form, do not fetch the templates
"""
import re
import sys
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DEFAULT = sorted((ROOT / "profiles").glob("*.yaml")) + [ROOT / "tests/kcl-deploy-profile.yaml"]

PINNED = re.compile(
    r"^https://raw\.githubusercontent\.com/stuttgart-things/kcl/"
    r"(?P<module>[a-z0-9]+(?:-[a-z0-9]+)*?)-(?P<version>\d+\.\d+\.\d+)/"
    r"(?P<path>[^\s#]+\.ya?ml)$")
# A list item under `templates:` (profiles) or `config.templateProfiles:`
# (built-in). Comment lines never match: they start with `#`.
ENTRY = re.compile(r"^\s*-\s+(?P<value>\S+)\s*$")


def entries(path):
    """[(line number, value)] of the template list(s) in a profile file."""
    out, inside = [], False
    for n, line in enumerate(path.read_text().splitlines(), 1):
        if re.match(r"^(templates|config\.templateProfiles):\s*$", line):
            inside = True
            continue
        if inside and re.match(r"^\S", line) and not line.startswith("#"):
            inside = False  # next top-level key
        if inside and (m := ENTRY.match(line)):
            out.append((n, m.group("value").strip("\"'")))
    return out


def spec_of(url):
    """(spec.source, spec.tag) of the ClaimTemplate at url."""
    with urllib.request.urlopen(url, timeout=20) as r:
        text = r.read().decode()
    spec = text.split("\nspec:", 1)[-1]
    src = re.search(r"^\s+source:\s*[\"']?([^\"'\s]+)", spec, re.M)
    tag = re.search(r"^\s+tag:\s*[\"']?([^\"'\s]+)", spec, re.M)
    return (src.group(1) if src else None, tag.group(1) if tag else None)


def main(argv):
    offline = "--offline" in argv
    files = [Path(a) for a in argv if a != "--offline"] or DEFAULT
    fail, ok = [], 0
    for f in files:
        rel = f.resolve().relative_to(ROOT) if f.resolve().is_relative_to(ROOT) else f
        found = entries(f)
        if not found:
            fail.append(f"{rel}: no template entries found")
        for n, url in found:
            where = f"{rel}:{n}"
            m = PINNED.match(url)
            if not m:
                fail.append(f"{where}: {url} is not pinned to a kcl module tag "
                            "(…/kcl/<module>-<X.Y.Z>/<path>)")
                continue
            module, version = m["module"], m["version"]
            if f"/{module}/templates/" not in f"/{m['path']}":
                fail.append(f"{where}: tag {module}-{version} does not belong to {m['path']}")
                continue
            if not offline:
                try:
                    src, tag = spec_of(url)
                except Exception as e:  # noqa: BLE001 -- report, keep checking
                    fail.append(f"{where}: cannot fetch {url}: {e}")
                    continue
                want = f"oci://ghcr.io/stuttgart-things/{module}"
                if src != want:
                    fail.append(f"{where}: spec.source {src} != {want}")
                if tag != version:
                    fail.append(f"{where}: spec.tag {tag} != {version} (the ref's version)")
            ok += 1
    for line in fail:
        print(f"FAIL {line}", file=sys.stderr)
    print(f"{ok} pinned template entr{'y' if ok == 1 else 'ies'} in {len(files)} file(s)"
          f"{' (offline)' if offline else ''}, {len(fail)} problem(s)")
    return 1 if fail else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
