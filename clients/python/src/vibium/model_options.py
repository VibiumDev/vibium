"""Public per-call model settings; credentials stay in the runtime environment."""
from typing import Optional


def model_params(provider: Optional[str], model: Optional[str], ai_base_url: Optional[str], reasoning_effort: Optional[str]) -> dict:
    return {key: value for key, value in {
        "provider": provider, "model": model, "aiBaseURL": ai_base_url,
        "reasoningEffort": reasoning_effort,
    }.items() if value is not None}
