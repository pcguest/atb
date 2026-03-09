"""Compatibility shim for the LangChain integration path.

Use ``atb.langchain_callback.ATBCallbackHandler`` for new code.
"""

from __future__ import annotations

import warnings
from typing import Any

from atb.langchain_callback import ATBCallbackHandler as _ATBCallbackHandler


class ATBCallbackHandler(_ATBCallbackHandler):
    """Deprecated import path compatibility wrapper.

    This class preserves the existing ``atb.integrations.langchain`` path and
    forwards to ``atb.langchain_callback.ATBCallbackHandler``.
    """

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        warnings.warn(
            "Importing ATBCallbackHandler from atb.integrations.langchain is deprecated; "
            "use atb.langchain_callback.ATBCallbackHandler instead.",
            DeprecationWarning,
            stacklevel=2,
        )
        super().__init__(*args, **kwargs)


__all__ = ["ATBCallbackHandler"]
