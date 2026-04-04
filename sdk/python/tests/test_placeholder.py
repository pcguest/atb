"""Placeholder tests for ATB Python SDK."""

from pathlib import Path

try:
    import tomllib
except ModuleNotFoundError:  # pragma: no cover - Python < 3.11 fallback
    import tomli as tomllib  # type: ignore[no-redef]


def test_sdk_imports():
    """Verify the SDK can be imported."""
    import atb

    assert hasattr(atb, "__version__")
    pyproject = Path(__file__).resolve().parents[1] / "pyproject.toml"
    expected = tomllib.loads(pyproject.read_text(encoding="utf-8"))["project"]["version"]
    assert atb.__version__ == expected


def test_bundle_class_exists():
    """Verify Bundle class is available."""
    from atb import Bundle

    assert Bundle is not None


def test_canonicalize_module_exists():
    """Verify canonicalize module is available."""
    from atb import canonicalize

    assert canonicalize is not None
