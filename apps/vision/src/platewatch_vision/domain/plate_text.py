from __future__ import annotations

import unicodedata


def canonicalize_plate_text(value: str) -> str:
    """Create a country-agnostic matching key while preserving Unicode letters/numbers."""
    normalized = unicodedata.normalize("NFKC", value).strip().upper()
    return "".join(character for character in normalized if character.isalnum())
