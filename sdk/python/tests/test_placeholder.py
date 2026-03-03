"""Placeholder tests for ATB Python SDK."""

def test_sdk_imports():
    """Verify the SDK can be imported."""
    import atb
    assert hasattr(atb, '__version__')
    assert atb.__version__ == "0.1.0"

def test_bundle_class_exists():
    """Verify Bundle class is available."""
    from atb import Bundle
    assert Bundle is not None
