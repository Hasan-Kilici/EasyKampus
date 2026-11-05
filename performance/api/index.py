"""Vercel Python function entrypoint for the performance service."""

import os
import sys

# Vercel çalışma dizini/sys.path her zaman performance/ klasörünü içermez;
# `app` paketinin bulunması için üst dizini açıkça ekle.
_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
if _ROOT not in sys.path:
    sys.path.insert(0, _ROOT)

from app.main import app  # noqa: E402

__all__ = ["app"]
