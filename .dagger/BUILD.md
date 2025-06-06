---
cwd: ..
shell: dagger shell
terminalRows: 20
---

# Building Runme

Initialize the Runme dagger module. Make sure `direnv` is setup.

```sh {"interpreter":"bash","terminalRows":"4"}
direnv allow
echo "Target platform: $TARGET_PLATFORM"
```

If `TARGET_PLATFORM` is not set, reset your Runme session. It's likely because direnv wasn't authorized yet.

Then, let's set the target platform on an instance of the Runme dagger module.

```sh {"name":"Runme"}
### Exported in runme.dev as Runme
. --target-platform $TARGET_PLATFORM
```

Check out what the module has to offer.

```sh
Runme | .help
```

## Local builds

Create a build from the local source directory. If `--source` is skipped, the build will default to https://github.com/runmedev/runme#main.

```sh
. --source . --target-platform $TARGET_PLATFORM | build
```

`Runme` uses sources from the `main` branch as per it's definition at the top.

```sh
Runme | binary
```

Run the tests.

```sh
Runme | test | stdout
```

## Remote builds

Testing latest `main` branch. Defaults to https://github.com/runmedev/runme#main without explicit `source`.

```sh
Runme |
    # test --pkgs "github.com/runmedev/runme/v3/document/editor/editorservice" |
    test |
    stdout
```

Build the binary.

```sh {"name":"BuildBinary"}
### Exported in runme.dev as BuildBinary
Runme | binary
```

Export it to local file.

```sh {"name":"LocalBinary"}
### Exported in runme.dev as LocalBinary
BuildBinary | export runme
```

## Releases

Access official pre-built releases (via goreleaser) stored in GitHub Releases.

```sh
. | list-release --version latest | entries
```

Access the files for a specific release on a particular platform.

```sh
. | link-release --version latest linux/arm64 | entries
```
