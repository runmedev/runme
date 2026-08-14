#!/usr/bin/env sh
set -eu

cat > textstats.py <<'PY'
"""Text statistics utilities."""

from collections import Counter


def word_count(text: str) -> int:
    """Return the number of words in text."""
    return len(text.split())


def most_common(text: str) -> str:
    """Return the most common word in lowercase."""
    words = text.lower().split()
    if not words:
        return ""
    return Counter(words).most_common(1)[0][0]
PY

cat > analyze.py <<'PY'
"""Analyze sample.txt and write results.json."""

import json
from pathlib import Path

from textstats import most_common, word_count


text = Path("sample.txt").read_text()
results = {
    "word_count": word_count(text),
    "most_common": most_common(text),
}
Path("results.json").write_text(json.dumps(results, indent=2))
PY

python3 analyze.py
python3 -m py_compile textstats.py analyze.py

# Harbor's oracle runner does not produce an agent trajectory. Record the
# successful validation so the workflow criterion can grade the reference
# solution using the same evidence format as an interactive agent run.
agent_log_dir=${RUNME_AGENT_LOG_DIR:-/logs/agent}
trajectory_path=${RUNME_AGENT_TRAJECTORY:-"$agent_log_dir/trajectory.json"}
cat > "$trajectory_path" <<'JSON'
{
  "schema_version": "ATIF-v1.7",
  "agent": {
    "name": "oracle",
    "version": "1.0.0"
  },
  "steps": [
    {
      "step_id": 1,
      "source": "agent",
      "message": "Ran the required validation command.",
      "llm_call_count": 0,
      "tool_calls": [
        {
          "tool_call_id": "compile-validation",
          "function_name": "shell",
          "arguments": {
            "command": "python3 -m py_compile textstats.py analyze.py"
          }
        }
      ],
      "observation": {
        "results": [
          {
            "source_call_id": "compile-validation",
            "content": "Process exited with code 0"
          }
        ]
      }
    }
  ]
}
JSON
