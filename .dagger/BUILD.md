---
cwd: ..
shell: dagger shell
terminalRows: 20
---

# Building Runme

Initialize the Runme dagger module. Make sure `direnv` is setup.

```sh {"name":"LinuxBuildEnv"}
### Exported in runme.dev as LinuxBuildEnv
. --target-platform "linux/amd64"
```

Check out what the module has to offer.

```sh
$LinuxBuildEnv | .help
```

## Local builds

Create a build from the local source directory.

```sh
$LinuxBuildEnv |
    with-source . |
    build
```

Run the tests.

```sh
$LinuxBuildEnv |
    with-source . |
    test |
    stdout
```

## Remote builds

Testing latest `main` branch.

```sh
$LinuxBuildEnv |
    test |
    stdout
```

Build the binary.

```sh {"interpreter":"bash","terminalRows":"4"}
echo "Target platform: $TARGET_PLATFORM"
```

```sh {"name":"BuildBinary"}
### Exported in runme.dev as BuildBinary
. --target-platform $TARGET_PLATFORM |
    binary
```

Export it to local file.

```sh
$BuildBinary | export runme-build-binary
```
