"""Roll up text-stats scores into Harbor's primary reward key."""

from rewardkit import criteria

criteria.correctness(weight=1.0)
criteria.structure(weight=1.0)
