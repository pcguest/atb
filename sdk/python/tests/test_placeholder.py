"""Placeholder tests for ATB Python SDK."""

from importlib import metadata


def test_sdk_imports():
    """Verify the SDK can be imported."""
    import atb

    assert hasattr(atb, "__version__")
    assert atb.__version__ == metadata.version("atb-sdk")


def test_bundle_class_exists():
    """Verify Bundle class is available."""
    from atb import Bundle

    assert Bundle is not None


def test_canonicalize_module_exists():
    """Verify canonicalize module is available."""
    from atb import canonicalize

    assert canonicalize is not None
