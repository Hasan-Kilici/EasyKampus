"""LMS performan servisi (Go/Fiber backend'in analiz sağlayıcısı)."""

from __future__ import annotations

import hmac
import os
from typing import Optional

from fastapi import Depends, FastAPI, Header, HTTPException, status

from . import models
from .analysis import analyze_student, analyze_teacher

app = FastAPI(
    title="LMS Performance Service",
    version="1.0.0",
    description=(
        "Öğrenci ve öğretmen performans analizi. Sözleşme sürümü "
        f"{models.CONTRACT_VERSION} (backend: pkg/domain/performance/contract.go)."
    ),
)


def require_internal_token(x_internal_token: Optional[str] = Header(default=None)) -> None:
    """Analiz uç noktalarını isteğe bağlı paylaşımlı token ile korur.

    Vercel'de bu servis herkese açık bir URL'de yayınlanır; token yoksa herkes
    ücretsiz kotayı tüketebilir. PERFORMANCE_INTERNAL_TOKEN tanımlıysa backend
    aynı değeri X-Internal-Token başlığıyla göndermelidir. Tanımsızsa davranış
    değişmez (yerel geliştirme). /api/health açık kalır.
    """
    expected = os.getenv("PERFORMANCE_INTERNAL_TOKEN", "")
    if not expected:
        return
    given = x_internal_token or ""
    if not hmac.compare_digest(given.encode("utf-8"), expected.encode("utf-8")):
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="invalid internal token")


@app.get("/api/health")
def health() -> dict:
    return {
        "status": "ok",
        "service": "performance",
        "contract_version": models.CONTRACT_VERSION,
    }


@app.post("/api/analyze/student", response_model=models.StudentResult, dependencies=[Depends(require_internal_token)])
def student(payload: models.StudentInput) -> models.StudentResult:
    return analyze_student(payload)


@app.post("/api/analyze/teacher", response_model=models.TeacherResult, dependencies=[Depends(require_internal_token)])
def teacher(payload: models.TeacherInput) -> models.TeacherResult:
    return analyze_teacher(payload)


def main() -> None:
    import uvicorn

    uvicorn.run(
        "app.main:app",
        host=os.getenv("PERFORMANCE_HOST", "0.0.0.0"),
        port=int(os.getenv("PERFORMANCE_PORT", "8000")),
        log_level=os.getenv("PERFORMANCE_LOG_LEVEL", "info"),
    )


if __name__ == "__main__":
    main()
