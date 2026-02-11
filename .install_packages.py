#!/usr/bin/env python3
"""Unified package installer that reads packages.json and installs for the current OS."""

import json
import os
import platform
import shutil
import subprocess
import sys
from pathlib import Path

# ── Install method priority for phase ordering ──────────────────────
SYSTEM_METHODS = ("brew", "cask", "apt", "dnf")
SECONDARY_METHODS = (
    "uv_tool",
    "cargo",
    "go_tool",
    "snap",
    "flatpak",
    "yay",
    "gh_extension",
    "eget",
    "manual",
)


def run(cmd: str, check: bool = False, capture: bool = False) -> subprocess.CompletedProcess:
    return subprocess.run(
        cmd, shell=True, check=check, capture_output=capture, text=True
    )


def command_exists(name: str) -> bool:
    return shutil.which(name) is not None


def detect_target() -> str:
    if platform.system() == "Darwin":
        return "darwin"
    try:
        with open("/etc/os-release") as f:
            for line in f:
                if line.startswith("ID="):
                    return line.strip().split("=")[1].strip('"')
    except FileNotFoundError:
        pass
    return "linux"


# ── Installed checks ────────────────────────────────────────────────

def is_brew_installed(pkg: str) -> bool:
    return run(f"brew list {pkg}", capture=True).returncode == 0


def is_cask_installed(pkg: str) -> bool:
    if run(f"brew list --cask {pkg}", capture=True).returncode == 0:
        return True
    # Check /Applications for apps that don't show in brew list
    app_name = pkg.replace("-", " ")
    for app_dir in (Path("/Applications"), Path.home() / "Applications"):
        if app_dir.exists():
            for entry in app_dir.iterdir():
                if app_name.lower() in entry.name.lower():
                    return True
    return False


def is_apt_installed(pkg: str) -> bool:
    result = run(f"dpkg -l {pkg} 2>/dev/null", capture=True)
    return any(line.startswith("ii") for line in result.stdout.splitlines())


def is_dnf_installed(pkg: str) -> bool:
    # pkg may be space-separated for groups like "gcc gcc-c++ make"
    for p in pkg.split():
        if run(f"rpm -q {p}", capture=True).returncode != 0:
            return False
    return True


def is_uv_tool_installed(pkg: str) -> bool:
    result = run("uv tool list", capture=True)
    return any(line.startswith(f"{pkg} ") for line in result.stdout.splitlines())


def is_cargo_installed(pkg: str) -> bool:
    result = run("cargo install --list", capture=True)
    return any(line.startswith(f"{pkg} ") for line in result.stdout.splitlines())


def is_go_tool_installed(pkg: str) -> bool:
    # Extract binary name: github.com/foo/bar@latest -> bar
    bin_name = pkg.rsplit("/", 1)[-1].split("@")[0]
    return command_exists(bin_name)


def is_snap_installed(name: str) -> bool:
    return run(f"snap list {name}", capture=True).returncode == 0


def is_flatpak_installed(app_id: str) -> bool:
    result = run("flatpak list", capture=True)
    return app_id in result.stdout


def is_yay_installed(pkg: str) -> bool:
    return run(f"yay -Qi {pkg}", capture=True).returncode == 0


def is_gh_extension_installed(repo: str) -> bool:
    ext_name = repo.rsplit("/", 1)[-1]
    result = run("gh extension list", capture=True)
    return ext_name in result.stdout


def is_eget_installed(repo: str) -> bool:
    tool_name = repo.rsplit("/", 1)[-1]
    return command_exists(tool_name)


def shell_expand(path: str) -> str:
    """Expand a path using the shell to handle bash syntax like ${VAR:-default}."""
    result = run(f'echo {path}', capture=True)
    if result.returncode == 0 and result.stdout.strip():
        return result.stdout.strip()
    return os.path.expandvars(path)


def is_manual_installed(manual: dict, pkg_name: str) -> bool:
    if "check_command" in manual:
        return command_exists(manual["check_command"])
    if "check_dir" in manual:
        return Path(shell_expand(manual["check_dir"])).is_dir()
    if "dest" in manual:
        return Path(shell_expand(manual["dest"])).is_dir()
    return command_exists(pkg_name)


def is_installed(pkg_name: str, method: str, value) -> bool:
    """Check if a package is already installed."""
    checkers = {
        "brew": lambda: is_brew_installed(value),
        "cask": lambda: is_cask_installed(value),
        "apt": lambda: is_apt_installed(value),
        "dnf": lambda: is_dnf_installed(value),
        "uv_tool": lambda: is_uv_tool_installed(value),
        "cargo": lambda: is_cargo_installed(value),
        "go_tool": lambda: is_go_tool_installed(value),
        "snap": lambda: is_snap_installed(value["name"] if isinstance(value, dict) else value),
        "flatpak": lambda: is_flatpak_installed(value),
        "yay": lambda: is_yay_installed(value),
        "gh_extension": lambda: is_gh_extension_installed(value),
        "eget": lambda: is_eget_installed(value),
        "manual": lambda: is_manual_installed(value, pkg_name),
    }
    checker = checkers.get(method)
    if checker and checker():
        return True
    # Fallback: check if the package name itself is a command
    return command_exists(pkg_name)


# ── Installers ──────────────────────────────────────────────────────

def install_brew(pkg_name: str, formula: str) -> None:
    print(f"  Installing {pkg_name} (brew)...")
    if run(f"brew install {formula}").returncode != 0:
        print(f"  Warning: Failed to install {pkg_name}")


def install_cask(pkg_name: str, cask: str) -> None:
    print(f"  Installing {pkg_name} (cask)...")
    result = run(f"brew install --cask {cask}", capture=True)
    if result.returncode != 0:
        if "It seems there is already an App at" in (result.stderr + result.stdout):
            print(f"  {pkg_name} already exists in /Applications, skipping")
        else:
            print(result.stdout)
            print(result.stderr)
            print(f"  Warning: Failed to install {pkg_name}")


def install_apt(pkg_name: str, pkg: str) -> None:
    print(f"  Installing {pkg_name} (apt)...")
    if run(f"sudo apt install -y {pkg}").returncode != 0:
        print(f"  Warning: Failed to install {pkg_name}")


def install_dnf(pkg_name: str, pkg: str) -> None:
    print(f"  Installing {pkg_name} (dnf)...")
    if run(f"sudo dnf install -y {pkg}").returncode != 0:
        print(f"  Warning: Failed to install {pkg_name}")


def install_uv_tool(pkg_name: str, tool: str) -> None:
    print(f"  Installing {pkg_name} (uv tool)...")
    if run(f"uv tool install {tool}").returncode != 0:
        print(f"  Warning: Failed to install {pkg_name}")


def install_cargo(pkg_name: str, crate: str) -> None:
    print(f"  Installing {pkg_name} (cargo)...")
    if run(f"cargo install {crate}").returncode != 0:
        print(f"  Warning: Failed to install {pkg_name}")


def install_go_tool(pkg_name: str, pkg_path: str) -> None:
    print(f"  Installing {pkg_name} (go install)...")
    if run(f"go install {pkg_path}").returncode != 0:
        print(f"  Warning: Failed to install {pkg_name}")


def install_snap(pkg_name: str, snap_spec) -> None:
    if isinstance(snap_spec, dict):
        name = snap_spec["name"]
        classic = snap_spec.get("classic", False)
    else:
        name = snap_spec
        classic = False
    print(f"  Installing {pkg_name} (snap)...")
    flag = " --classic" if classic else ""
    if run(f"sudo snap install {name}{flag}").returncode != 0:
        print(f"  Warning: Failed to install {pkg_name}")


def install_flatpak(pkg_name: str, app_id: str) -> None:
    print(f"  Installing {pkg_name} (flatpak)...")
    if run(f"flatpak install -y flathub {app_id}").returncode != 0:
        print(f"  Warning: Failed to install {pkg_name}")


def install_yay(pkg_name: str, pkg: str) -> None:
    print(f"  Installing {pkg_name} (yay)...")
    if run(f"yay -S --noconfirm {pkg}").returncode != 0:
        print(f"  Warning: Failed to install {pkg_name}")


def install_gh_extension(pkg_name: str, repo: str) -> None:
    if run("gh auth status", capture=True).returncode != 0:
        print(f"  Skipping {pkg_name} (gh not authenticated, run 'gh auth login' first)")
        return
    print(f"  Installing {pkg_name} (gh extension)...")
    if run(f"gh extension install {repo}").returncode != 0:
        print(f"  Warning: Failed to install {pkg_name}")


def install_eget(pkg_name: str, repo: str) -> None:
    print(f"  Installing {pkg_name} (eget)...")
    Path.home().joinpath(".local", "bin").mkdir(parents=True, exist_ok=True)
    if run(f"eget {repo} --to ~/.local/bin").returncode != 0:
        print(f"  Warning: Failed to install {pkg_name}")


def install_manual(pkg_name: str, manual: dict) -> None:
    install_type = manual["type"]
    url = manual["url"]

    if install_type == "script":
        args = manual.get("args", "")
        print(f"  Installing {pkg_name} (manual script)...")
        if args:
            cmd = f'sh -c "$(curl -fsSL {url})" "" {args}'
        else:
            cmd = f"curl -fsSL {url} | bash"
        if run(cmd).returncode != 0:
            print(f"  Warning: Failed to install {pkg_name}")

    elif install_type == "git_clone":
        dest = shell_expand(manual["dest"])
        print(f"  Installing {pkg_name} (git clone)...")
        Path(dest).parent.mkdir(parents=True, exist_ok=True)
        if run(f"git clone {url} {dest}").returncode != 0:
            print(f"  Warning: Failed to install {pkg_name}")



INSTALLERS = {
    "brew": install_brew,
    "cask": install_cask,
    "apt": install_apt,
    "dnf": install_dnf,
    "uv_tool": install_uv_tool,
    "cargo": install_cargo,
    "go_tool": install_go_tool,
    "snap": install_snap,
    "flatpak": install_flatpak,
    "yay": install_yay,
    "gh_extension": install_gh_extension,
    "eget": install_eget,
    "manual": install_manual,
}


# ── Main ────────────────────────────────────────────────────────────

def main() -> None:
    target = sys.argv[1] if len(sys.argv) > 1 else detect_target()
    packages_path = Path(__file__).parent / "packages.json"

    with open(packages_path) as f:
        packages = json.load(f)

    print(f"Installing packages for target: {target}")
    print()

    # Phase 1: Brew taps
    taps = packages.get("_brew_taps", [])
    if taps and command_exists("brew"):
        print("=== Adding Homebrew taps ===")
        for tap in taps:
            if run(f"brew tap {tap}", capture=True).returncode != 0:
                print(f"  Warning: Failed to tap {tap}")
            else:
                print(f"  {tap}")
        print()

    # Update package lists for Linux
    if target not in ("darwin",):
        if target in ("ubuntu", "pop"):
            print("Updating apt package lists...")
            run("sudo apt update")
            print()
        elif target == "fedora":
            run("sudo dnf check-update", capture=True)

    # Collect packages for this target
    pkg_items = []
    for name, pkg in packages.items():
        if name.startswith("_"):
            continue
        if not isinstance(pkg, dict) or "packages" not in pkg:
            continue
        target_config = pkg["packages"].get(target)
        if not target_config:
            continue
        pkg_items.append((name, target_config))

    # Phase 2: System packages (brew, cask, apt, dnf)
    print("=== Installing system packages ===")
    for name, target_config in pkg_items:
        for method in SYSTEM_METHODS:
            if method in target_config:
                value = target_config[method]
                if is_installed(name, method, value):
                    print(f"  [ok] {name}")
                else:
                    INSTALLERS[method](name, value)
                break
    print()

    # Phase 3: Secondary packages (uv_tool, cargo, go_tool, snap, flatpak, yay, gh_extension, eget, manual)
    print("=== Installing secondary packages ===")
    for name, target_config in pkg_items:
        # Skip if already handled in phase 2
        if any(m in target_config for m in SYSTEM_METHODS):
            continue
        for method in SECONDARY_METHODS:
            if method in target_config:
                value = target_config[method]
                if is_installed(name, method, value):
                    print(f"  [ok] {name}")
                else:
                    INSTALLERS[method](name, value)
                break
    print()

    print("Package installation complete!")


if __name__ == "__main__":
    main()
