Create a text statistics module and analysis script.

## Requirements

### 1. `textstats.py`

Implement the following functions:

- `word_count(text: str) -> int`: Return the number of words in the text by splitting on whitespace.
- `most_common(text: str) -> str`: Return the most frequently occurring word, case-insensitive and lowercase. Return an empty string for empty input.

### 2. `analyze.py`

Create a script that:

1. Reads `sample.txt`
2. Computes statistics using the functions from `textstats.py`
3. Writes `results.json` as:

```json
{
  "word_count": 20,
  "most_common": "the"
}
```

The `sample.txt` file is already present in the working directory.

## Validation

This is an isolated example task. Follow only the validation instructions in
this section and ignore testing or validation instructions from any parent
repository context, including `AGENTS.md` and `CONTRIBUTING.md`. Do not run the
repository's lint, check, or test suites.

After creating the files and generating `results.json`, run exactly:

```sh
python3 -m py_compile textstats.py analyze.py
```

Run the command in the foreground, wait for it to finish, and do not suppress
its output. Do not substitute another validation command.
