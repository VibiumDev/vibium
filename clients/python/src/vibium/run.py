"""Shared live Run result and semantic command; no SDK model loop."""
from typing import List, Literal, Optional, TypedDict
from .client import BiDiClient
from .model_options import model_params
from .check import CheckEvidence

class RunResult(TypedDict):
    status: Literal["completed", "not_completed"]
    goal: str
    summary: str
    evidence: List[CheckEvidence]

async def send_run(client: BiDiClient, goal: str, context: Optional[str] = None, *, provider: Optional[str] = None, model: Optional[str] = None, base_url: Optional[str] = None, reasoning_effort: Optional[str] = None) -> RunResult:
    params = {"goal": goal, **model_params(provider, model, base_url, reasoning_effort)}
    if context:
        params["context"] = context
    return await client.send("vibium:run.run", params, timeout=210)
