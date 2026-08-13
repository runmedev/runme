# /// script
# dependencies = [
#   "anthropic>=0.75.0",
#   "pydantic==2.12.5",
# ]
# ///

# Adapted from harbor-framework/harbor examples/tasks/llm-judge-example,
# licensed under Apache-2.0.

import json
import os
import sys
from pathlib import Path

from anthropic import Anthropic, transform_schema
from pydantic import BaseModel, Field


class HumorJudgment(BaseModel):
    """Structured LLM judgment for the poem."""

    funny_score: float = Field(
        ...,
        ge=0.0,
        le=1.0,
        description="How funny the poem is, from 0.0 to 1.0.",
    )
    reasoning: str = Field(
        ...,
        description="Brief explanation of the humor score.",
    )


def env_path(key: str, fallback: str) -> Path:
    return Path(os.environ.get(key, fallback))


def write_json(path: Path, value: object) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, indent=2) + "\n")


def read_poem(path: Path) -> str:
    try:
        return path.read_text().strip()
    except FileNotFoundError as exc:
        raise RuntimeError(f"missing poem file: {path}") from exc


def judge_poem(poem: str) -> HumorJudgment:
    api_key = os.environ.get("ANTHROPIC_API_KEY")
    if not api_key:
        raise RuntimeError("ANTHROPIC_API_KEY is required for the LLM judge verifier")

    model = os.environ.get("MODEL_NAME", "claude-haiku-4-5")
    client = Anthropic(api_key=api_key)
    schema = transform_schema(HumorJudgment.model_json_schema())

    response = client.messages.create(
        model=model,
        max_tokens=1024,
        output_config={"format": {"type": "json_schema", "schema": schema}},
        messages=[
            {
                "role": "user",
                "content": f"""Rate how funny this poem is from 0.0 to 1.0.

Consider whether it has a clear joke, surprise, wordplay, or a playful take on
developer workflow. Return only the requested structured JSON.

Poem:
{poem}""",
            }
        ],
    )

    return HumorJudgment.model_validate_json(response.content[0].text)


def reward_details(score: float, reasoning: str) -> dict[str, object]:
    return {
        "funny": {
            "score": score,
            "kind": "llm-judge",
            "criteria": [
                {
                    "name": "funny",
                    "value": score,
                    "raw": score,
                    "weight": 1.0,
                    "description": (
                        "LLM judgment of poem humor from 0.0 to 1.0. "
                        f"Reasoning: {reasoning}"
                    ),
                }
            ],
        }
    }


def main() -> None:
    poem_path = Path(sys.argv[1]) if len(sys.argv) > 1 else Path("poem.txt")
    verifier_dir = env_path("RUNME_VERIFIER_DIR", "/logs/verifier")
    reward_path = env_path("RUNME_REWARD_PATH", str(verifier_dir / "reward.json"))
    reward_details_path = env_path(
        "RUNME_REWARD_DETAILS_PATH",
        str(verifier_dir / "reward-details.json"),
    )

    poem = read_poem(poem_path)
    if not poem:
        raise RuntimeError(f"poem file is empty: {poem_path}")

    judgment = judge_poem(poem)
    score = judgment.funny_score

    write_json(reward_path, {"reward": score, "funny": score})
    write_json(reward_details_path, reward_details(score, judgment.reasoning))
    write_json(
        verifier_dir / "diagnostics.json",
        {
            "poem_path": str(poem_path),
            "model": os.environ.get("MODEL_NAME", "claude-haiku-4-5"),
            "reasoning": judgment.reasoning,
        },
    )

    print(f"Funny score: {score}")
    print(f"Reasoning: {judgment.reasoning}")


if __name__ == "__main__":
    main()
