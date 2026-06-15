#!/usr/bin/env python3
"""Tests for scripts/autofill-changelog.py."""

from __future__ import annotations

import importlib.util
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).resolve().parent / "autofill-changelog.py"


def load_module():
    spec = importlib.util.spec_from_file_location("autofill_changelog", SCRIPT)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"cannot load {SCRIPT}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


mod = load_module()

SAMPLE = """\
# Changelog

Preamble.

## [Unreleased]

### Changed

- Scan commands (`model-scan trigger`/`info`, `mcp-scan run`/`info --scan-id`) now
  print **raw API JSON by default** for Python CLI migration parity; use
  `--envelope` for the normalized verdict JSON/text.
- `model-scan list` and `model-scan info` print verbatim API JSON
  (pass-through), matching Python CLI output shape; paging serializes as
  `count`/`offset`/`total` instead of `limit`.

## [0.1.1] - 2026-06-12

### Fixed

- Release workflow dispatch (#6)
"""


class ParseUnreleasedTests(unittest.TestCase):
    def test_multiline_bullets_are_preserved(self) -> None:
        _, sections, _ = mod.parse_unreleased(SAMPLE)

        self.assertEqual(len(sections["Changed"]), 2)
        self.assertIn("print **raw API JSON by default**", sections["Changed"][0])
        self.assertIn("`--envelope`", sections["Changed"][0])
        self.assertIn("(pass-through)", sections["Changed"][1])
        self.assertIn("`count`/`offset`/`total`", sections["Changed"][1])

    def test_nested_sub_bullets_stay_with_parent(self) -> None:
        text = """\
# Changelog

## [Unreleased]

### Changed

- **Backward-incompatible vs. the Python `aidcli`**:
  - Scan commands print the verdict envelope by default.
  - A completed scan with findings now exits `10`/`20`.

## [0.1.0] - 2026-01-01
"""
        _, sections, _ = mod.parse_unreleased(text)

        self.assertEqual(len(sections["Changed"]), 1)
        bullet = sections["Changed"][0]
        self.assertIn("- **Backward-incompatible", bullet)
        self.assertIn("  - Scan commands print", bullet)
        self.assertIn("  - A completed scan", bullet)


class RoundTripTests(unittest.TestCase):
    def test_write_changelog_keeps_multiline_bullets(self) -> None:
        before, sections, after = mod.parse_unreleased(SAMPLE)

        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "CHANGELOG.md"
            mod.write_changelog(path, before, sections, after)
            _, sections2, _ = mod.parse_unreleased(path.read_text(encoding="utf-8"))

        self.assertEqual(sections, sections2)

    def test_render_unreleased_outputs_full_bullets(self) -> None:
        _, sections, _ = mod.parse_unreleased(SAMPLE)
        rendered = mod.render_unreleased(sections)

        self.assertIn("print **raw API JSON by default**", rendered)
        self.assertIn("`count`/`offset`/`total`", rendered)


class PrepareReleaseOrderTests(unittest.TestCase):
    def test_unreleased_stays_above_new_release_section(self) -> None:
        """Match scripts/prepare-release.sh finalized CHANGELOG layout."""
        before = "# Changelog\n\nIntro.\n\n"
        unreleased_body = "### Changed\n\n- A thing\n"
        after = "## [0.1.2] - 2026-06-12\n\n### Fixed\n\n- Old fix\n"
        result = (
            f"{before}## [Unreleased]\n\n"
            f"## [0.1.3] - 2026-06-15\n\n"
            f"{unreleased_body}\n"
            f"{after}"
        )

        unreleased_pos = result.index("## [Unreleased]")
        new_release_pos = result.index("## [0.1.3]")
        old_release_pos = result.index("## [0.1.2]")
        self.assertLess(unreleased_pos, new_release_pos)
        self.assertLess(new_release_pos, old_release_pos)


if __name__ == "__main__":
    unittest.main()
