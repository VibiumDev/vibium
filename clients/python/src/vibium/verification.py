"""Shared result types and semantic Verify command."""
from pathlib import Path
from typing import List, Literal, Optional, TypedDict
from .client import BiDiClient

class VerificationEvidence(TypedDict):
    type: Literal["observation"]
    summary: str

class VerificationResult(TypedDict):
    status: Literal["passed", "failed", "inconclusive"]
    claim: str
    summary: str
    evidence: List[VerificationEvidence]

async def send_verification(client: BiDiClient, claim: str, record: Optional[str] = None, context: Optional[str] = None) -> VerificationResult:
    params = {"claim": claim}
    if record is not None:
        if not record:
            raise ValueError("record must be a nonempty path")
        params["record"] = str(Path(record).resolve())
    elif context:
        params["context"] = context
    return await client.send("vibium:verify.run", params, timeout=210)
