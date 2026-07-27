"""Require evidence that the agent ran the lint task successfully."""

from rewardkit import criteria

criteria.lint_validation(weight=1.0)
