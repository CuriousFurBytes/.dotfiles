# sandbox

Per-project podman sandboxes with VM-level isolation via `krun` and an overlay
mount strategy that keeps the host filesystem untouched until you explicitly
sync changes back.

## Files

- `~/.local/bin/sandbox` &mdash; main CLI (`create`, `sh`, `run`, `list`, `prune`, `sync`, `rebuild`, ...)
- `~/.local/bin/sbx` &mdash; short wrapper with tool shortcuts (`sbx claude`, `sbx codex`, `sbx copilot`, `sbx commit`, ...)
- `~/.local/share/sandbox/Containerfile` &mdash; base image spec
- `~/.local/share/sandbox/overlays/<sandbox>/upper` &mdash; per-sandbox overlay write layer
- `~/.local/state/sandbox/<sandbox>.json` &mdash; per-sandbox metadata

## Prerequisites

Every dependency is declared in `packages.json` and picked up by this repo's
installer. If you ran `chezmoi apply`, the tools below were already installed
and `.chezmoiscripts/run_onchange_after_56-setup-sandbox.sh.tmpl` did the
per-OS setup.

| OS | What gets installed | krun (VM isolation)? |
|----|--------------------|------------------------|
| **Fedora**        | `podman`, `crun-krun`, `rsync`, `fuse-overlayfs`, `slirp4netns` | yes, native |
| **Ubuntu / Pop!_OS** | `podman`, `rsync`, `fuse-overlayfs`, `slirp4netns` | no &mdash; falls back to `crun`* |
| **macOS**         | `podman` (via brew), `rsync` | n/a &mdash; the whole podman runtime is already a VM |

*On Ubuntu, `crun-krun` is not in the default apt repos. `sandbox` logs a
warning and uses `crun` (regular rootless containers, shared host kernel). To
get real VM isolation, build `crun-krun` from
<https://github.com/containers/crun/tree/main/krun> and drop the binary into
`~/.local/bin/` or `/usr/local/bin/`.

### Manual install (if you don't use this dotfiles repo)

```bash
# Fedora
sudo dnf install -y podman crun-krun rsync fuse-overlayfs slirp4netns

# Ubuntu / Pop!_OS
sudo apt install -y podman rsync fuse-overlayfs slirp4netns

# macOS
brew install podman rsync
podman machine init --now   # one-time: start the Linux VM podman uses
```

## First-time setup

```bash
# 1. Build the base image
sandbox build

# 2. (Recommended) Log into every tool once and bake the auth state into
#    the base image so future sandboxes start already authenticated.
sandbox rebuild
# ^ drops you into a shell; run:
#     gh auth login
#     claude        # log in, then ctrl-c
#     codex         # log in, then ctrl-c
#     copilot       # GitHub Copilot CLI first-run login
#     git config --global user.name  "Your Name"
#     git config --global user.email "you@example.com"
#   then 'exit' - the container is committed back into
#   `localhost/sandbox-base:latest`.
```

## Daily workflow

```bash
cd ~/Documents/projects/my-project

sandbox create       # spin up a sandbox for this folder (krun VM)
sandbox sh           # drop into a shell in /workspace
sandbox run npm test # run anything ad hoc
sbx claude           # shorthand for `sandbox run claude`
sbx codex
sbx copilot
sbx commit           # runs `gitscribe commit` inside the sandbox

sandbox list         # see all sandboxes
sandbox sync         # rsync the overlay's writes back to the host folder
sandbox rm           # remove the sandbox for $PWD
sandbox prune        # nuke every sandbox + overlay dir
```

### What `sync` does

Changes inside the sandbox never touch your host files directly. They go into
an overlay *upper layer* at `~/.local/share/sandbox/overlays/<name>/upper`.
`sandbox sync` pauses the container and `rsync`'s that upper layer into the
original host folder. After sync the host reflects the sandbox state; the
overlay is preserved (so the container keeps running with a consistent view
of its writes).

If you want a one-way "throwaway" sandbox, just never call `sync` &mdash; when
you run `sandbox rm` or `sandbox prune`, the overlay is deleted and the host
folder is untouched.

## Rebuilding / re-baking the base image

The base image at `localhost/sandbox-base:latest` is stored in your local
podman storage. You have three ways to rebuild it:

### 1. `sandbox build` &mdash; rebuild from the Containerfile

Rebuilds fresh from `~/.local/share/sandbox/Containerfile`. Use this when you
edited the Containerfile (adding a package, updating a tool version). All
previous auth state is lost because the layers are rebuilt.

```bash
sandbox build
```

### 2. `sandbox rebuild` &mdash; rebuild + interactive auth bake

Rebuilds the image, then opens an interactive shell in a scratch container.
Log in to every tool you care about; when you `exit`, the container is
committed back to `localhost/sandbox-base:latest`. Future sandboxes spawn
already authenticated.

```bash
sandbox rebuild
```

### 3. Manual tweak then commit

Useful when you want to add something quickly without re-baking everything:

```bash
podman run --name sbx-tweak -it --runtime krun \
    localhost/sandbox-base:latest /bin/bash
# ... install stuff, log in, configure ...
# exit
podman commit sbx-tweak localhost/sandbox-base:latest
podman rm sbx-tweak
```

## Using a sandbox from your IDE

The goal is to point your IDE's language server / tasks / terminal at the
running sandbox container instead of the host.

### Zed

Zed supports remote development targets via **SSH**. Easiest path:

1. Start the sandbox and open a sh session once: `sandbox create && sandbox sh`
2. In a second terminal, grab the sandbox's short name: `sandbox path`
3. Add an SSH config block so Zed can ProxyCommand through podman:

    ```sshconfig
    # ~/.ssh/config
    Host sbx-*
        User dev
        # Use podman exec as the transport (no sshd needed inside the container)
        ProxyCommand podman exec -i %h /bin/bash
        RequestTTY yes
        StrictHostKeyChecking no
        UserKnownHostsFile /dev/null
    ```

4. In Zed, `cmd-shift-p` &rarr; `remote: open folder` &rarr; connect to
   `sbx-<your-folder>-<hash>` and open `/workspace`.

Alternatively, use Zed's tasks (`.zed/tasks.json`) to shell out to
`sandbox run ...`:

```json
[
  { "label": "sandbox: test",  "command": "sandbox run npm test" },
  { "label": "sandbox: lint",  "command": "sandbox run npm run lint" },
  { "label": "sandbox: shell", "command": "sandbox sh" }
]
```

### VS Code

Use the **Dev Containers** extension's "Attach to Running Container" command:

1. `sandbox create` in your project folder.
2. In VS Code: `F1` &rarr; `Dev Containers: Attach to Running Container...`
3. Pick `sbx-<folder>-<hash>`. VS Code installs its server into the
   container and opens `/workspace`.
4. Install your usual extensions inside the container; they are cached in
   the overlay and survive `sandbox sync` / container restarts.

If you want the config persisted across sandboxes, add the extensions to the
base image during `sandbox rebuild`:

```bash
# inside the rebuild shell:
code --install-extension dbaeumer.vscode-eslint
```

For Dev Containers' "Open Folder in Container" workflow, point at the
Containerfile:

```jsonc
// .devcontainer/devcontainer.json
{
  "name": "sandbox",
  "build": { "dockerfile": "../.local/share/sandbox/Containerfile" },
  "runArgs": ["--runtime=krun", "--userns=keep-id"],
  "workspaceMount": "type=overlay,source=${localWorkspaceFolder},destination=/workspace,upperdir=${localEnv:HOME}/.local/share/sandbox/overlays/devcontainer/upper,workdir=${localEnv:HOME}/.local/share/sandbox/overlays/devcontainer/work",
  "workspaceFolder": "/workspace"
}
```

### JetBrains / Helix / anything else

Anything that can run a command works with `sandbox run`:

```bash
sandbox run cargo build
sandbox run pytest -q
sandbox run hx .   # yes, even your editor
```

## Environment knobs

| Var | Default | Meaning |
|-----|---------|---------|
| `SANDBOX_IMAGE`   | `localhost/sandbox-base:latest` | base image ref |
| `SANDBOX_RUNTIME` | `krun` | OCI runtime (falls back to `crun`) |
| `SANDBOX_NET`     | `slirp4netns:enable_ipv6=true` | `--network` value |
| `SANDBOX_MEMORY`  | `4g` | `--memory` value |
| `SANDBOX_CPUS`    | `4`  | `--cpus` value |

## Troubleshooting

- **`Error: overlay is not supported over ...`** &mdash; the overlay mount
  needs a kernel that supports rootless overlay (5.11+). If you're on an
  older kernel, set `SANDBOX_RUNTIME=crun` and edit `sandbox` to replace the
  `--mount type=overlay,...` flag with `-v "$abs:/workspace:O"` (uses
  podman's transient overlay; `sync` won't work because the upper layer is
  anonymous).
- **`krun runtime not available`** &mdash; install `crun-krun`. The script
  will fall back to `crun` with a warning but you lose kernel isolation.
- **permission denied on overlay upperdir** &mdash; `--userns=keep-id` maps
  the host UID into the container. If you originally created the upperdir
  as root, `rm -rf ~/.local/share/sandbox/overlays/<name>` and recreate.
- **gitscribe missing** &mdash; the Containerfile tries the release asset
  then `go install`. If both fail, install it manually during
  `sandbox rebuild` &mdash; the committed image keeps the binary.
