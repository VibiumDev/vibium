"""Shared result types and semantic Check command."""
from pathlib import Path
from typing import List, Literal, Optional, TypedDict
from .client import BiDiClient
from .model_options import model_params

class CheckEvidence(TypedDict):
    type: Literal["observation"]
    summary: str

class CheckResult(TypedDict):
    status: Literal["passed", "failed", "inconclusive"]
    claim: str
    summary: str
    evidence: List[CheckEvidence]

async def send_check(client: BiDiClient, claim: str, record: Optional[str] = None, context: Optional[str] = None, *, provider: Optional[str] = None, model: Optional[str] = None, base_url: Optional[str] = None, reasoning_effort: Optional[str] = None) -> CheckResult:
    params = {"claim": claim, **model_params(provider, model, base_url, reasoning_effort)}
    if record is not None:
        if not record:
            raise ValueError("record must be a nonempty path")
        params["record"] = str(Path(record).resolve())
    elif context:
        params["context"] = context
    return await client.send("vibium:check.run", params, timeout=210)
