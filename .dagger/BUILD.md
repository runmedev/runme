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

```sh {"name":"BuildEnv"}
### Exported in runme.dev as BuildEnv
. --target-platform $TARGET_PLATFORM
```

Check out what the module has to offer.

```sh
BuildEnv | .help
```

## Local builds

Create a build from the local source directory.

```sh
BuildEnv |
    with-source . |
    build
```

Run the tests.

```sh
BuildEnv |
    with-source . |
    test |
    stdout
```

## Remote builds

Testing latest `main` branch.

```sh
BuildEnv |
    # test --pkgs "github.com/runmedev/runme/v3/pkg/document/editor/editorservice" |
    test |
    stdout
```

Build the binary.

```sh {"name":"BuildBinary"}
### Exported in runme.dev as BuildBinary
BuildEnv |
    binary
```

Export it to local file.

```sh
BuildBinary | export /tmp/runme-binary
```

## Releases

```sh
. | release | entries
```

```sh
. | release-files linux/arm64 | entries
```
