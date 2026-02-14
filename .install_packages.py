#!/usr/bin/env python3
"""Unified package installer that reads packages.json and installs for the current OS."""

import json
import os
import platform
import shutil
import subprocess
import sys
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path


# Ensure rich is available — re-exec in a venv if needed
def _ensure_rich() -> None:
    try:
        import rich  # noqa: F401
        return
    except ImportError:
        pass

    venv_dir = Path.home() / ".cache" / "dotfiles-venv"
    venv_python = venv_dir / "bin" / "python3"

    if not venv_python.exists():
        print("Creating temporary venv for rich...")
        subprocess.run([sys.executable, "-m", "venv", str(venv_dir)], check=True)
        subprocess.run([str(venv_python), "-m", "pip", "install", "rich", "-q"], check=True)

    # Re-exec this script under the venv python
    os.execv(str(venv_python), [str(venv_python), "-B"] + sys.argv)

_ensure_rich()

from rich.console import Console  # noqa: E402
from rich.panel import Panel  # noqa: E402

console = Console()

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


def print_status(name: str, status: str) -> None:
    """Print a package status line with consistent formatting."""
    if status == "ok":
        console.print(f"  [green]\\[ok][/green] {name}")
    elif status == "done":
        console.print(f"  [blue]\\[done][/blue] {name}")
    elif status == "skip":
        console.print(f"  [yellow]\\[skip][/yellow] {name}")
    elif status == "fail":
        console.print(f"  [red]\\[fail][/red] {name}")


def print_section(title: str) -> None:
    """Print a section header."""
    console.print()
    console.rule(f"[bold]{title}[/bold]")
    console.print()


# ── Installed caches (populated once, checked many times) ───────────

_cache: dict[str, set[str]] = {}


def _get_cached(key: str, cmd: str, parse=None) -> set[str]:
    """Run a command once and cache the parsed output as a set."""
    if key not in _cache:
        result = run(cmd, capture=True)
        if result.returncode != 0:
            _cache[key] = set()
        elif parse:
            _cache[key] = parse(result.stdout)
        else:
            _cache[key] = set(result.stdout.split())
        # Also cache installed app names from /Applications
        if key == "cask":
            apps = set()
            for app_dir in (Path("/Applications"), Path.home() / "Applications"):
                if app_dir.exists():
                    for entry in app_dir.iterdir():
                        apps.add(entry.stem.lower().replace(" ", "-"))
            _cache["cask_apps"] = apps
    return _cache[key]


def _parse_lines(stdout: str) -> set[str]:
    return {line.strip() for line in stdout.splitlines() if line.strip()}


def _parse_pkg_names(stdout: str) -> set[str]:
    """Parse output where package name is the first word on each line."""
    return {line.split()[0] for line in stdout.splitlines() if line.strip()}


# ── Installed checks ────────────────────────────────────────────────

def is_brew_installed(pkg: str) -> bool:
    installed = _get_cached("brew", "brew list --formula -1", _parse_lines)
    return pkg in installed


def is_cask_installed(pkg: str) -> bool:
    installed = _get_cached("cask", "brew list --cask -1", _parse_lines)
    if pkg in installed:
        return True
    return pkg.lower() in _cache.get("cask_apps", set())


def is_apt_installed(pkg: str) -> bool:
    installed = _get_cached(
        "apt",
        "dpkg-query -W -f='${Package}\n' 2>/dev/null",
        _parse_lines,
    )
    return pkg in installed


def is_dnf_installed(pkg: str) -> bool:
    installed = _get_cached("dnf", "rpm -qa --qf '%{NAME}\n'", _parse_lines)
    return all(p in installed for p in pkg.split())


def is_uv_tool_installed(pkg: str) -> bool:
    installed = _get_cached("uv_tool", "uv tool list", _parse_pkg_names)
    return pkg in installed


def is_cargo_installed(pkg: str) -> bool:
    installed = _get_cached("cargo", "cargo install --list", _parse_pkg_names)
    return pkg in installed


def is_go_tool_installed(pkg: str) -> bool:
    bin_name = pkg.rsplit("/", 1)[-1].split("@")[0]
    return command_exists(bin_name)


def is_snap_installed(name: str) -> bool:
    installed = _get_cached("snap", "snap list 2>/dev/null", _parse_pkg_names)
    return name in installed


def is_flatpak_installed(app_id: str) -> bool:
    installed = _get_cached(
        "flatpak",
        "flatpak list --columns=application 2>/dev/null",
        _parse_lines,
    )
    return app_id in installed


def is_yay_installed(pkg: str) -> bool:
    installed = _get_cached("yay", "yay -Qq 2>/dev/null", _parse_lines)
    return pkg in installed


def is_gh_extension_installed(repo: str) -> bool:
    ext_name = repo.rsplit("/", 1)[-1]
    installed = _get_cached("gh_ext", "gh extension list 2>/dev/null", _parse_lines)
    return any(ext_name in entry for entry in installed)


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
    with console.status(f"Installing [bold]{pkg_name}[/bold] (brew)..."):
        if run(f"zb install {formula}", capture=True).returncode != 0:
            print_status(pkg_name, "fail")
        else:
            print_status(pkg_name, "done")


def install_cask(pkg_name: str, cask: str) -> None:
    with console.status(f"Installing [bold]{pkg_name}[/bold] (cask)..."):
        result = run(f"zb install --cask {cask}", capture=True)
    if result.returncode != 0:
        if "It seems there is already an App at" in (result.stderr + result.stdout):
            print_status(pkg_name, "skip")
        else:
            print_status(pkg_name, "fail")
    else:
        print_status(pkg_name, "done")


def install_apt(pkg_name: str, pkg: str) -> None:
    with console.status(f"Installing [bold]{pkg_name}[/bold] (apt)..."):
        if run(f"sudo apt install -y {pkg}", capture=True).returncode != 0:
            print_status(pkg_name, "fail")
        else:
            print_status(pkg_name, "done")


def install_dnf(pkg_name: str, pkg: str) -> None:
    with console.status(f"Installing [bold]{pkg_name}[/bold] (dnf)..."):
        if run(f"sudo dnf install -y {pkg}", capture=True).returncode != 0:
            print_status(pkg_name, "fail")
        else:
            print_status(pkg_name, "done")


def install_uv_tool(pkg_name: str, tool: str) -> None:
    with console.status(f"Installing [bold]{pkg_name}[/bold] (uv tool)..."):
        if run(f"uv tool install {tool}", capture=True).returncode != 0:
            print_status(pkg_name, "fail")
        else:
            print_status(pkg_name, "done")


def install_cargo(pkg_name: str, crate: str) -> None:
    with console.status(f"Installing [bold]{pkg_name}[/bold] (cargo)..."):
        if run(f"cargo install {crate}", capture=True).returncode != 0:
            print_status(pkg_name, "fail")
        else:
            print_status(pkg_name, "done")


def install_go_tool(pkg_name: str, pkg_path: str) -> None:
    with console.status(f"Installing [bold]{pkg_name}[/bold] (go install)..."):
        if run(f"go install {pkg_path}", capture=True).returncode != 0:
            print_status(pkg_name, "fail")
        else:
            print_status(pkg_name, "done")


def install_snap(pkg_name: str, snap_spec) -> None:
    if isinstance(snap_spec, dict):
        name = snap_spec["name"]
        classic = snap_spec.get("classic", False)
    else:
        name = snap_spec
        classic = False
    flag = " --classic" if classic else ""
    with console.status(f"Installing [bold]{pkg_name}[/bold] (snap)..."):
        if run(f"sudo snap install {name}{flag}", capture=True).returncode != 0:
            print_status(pkg_name, "fail")
        else:
            print_status(pkg_name, "done")


def install_flatpak(pkg_name: str, app_id: str) -> None:
    with console.status(f"Installing [bold]{pkg_name}[/bold] (flatpak)..."):
        if run(f"flatpak install -y flathub {app_id}", capture=True).returncode != 0:
            print_status(pkg_name, "fail")
        else:
            print_status(pkg_name, "done")


def install_yay(pkg_name: str, pkg: str) -> None:
    with console.status(f"Installing [bold]{pkg_name}[/bold] (yay)..."):
        if run(f"yay -S --noconfirm {pkg}", capture=True).returncode != 0:
            print_status(pkg_name, "fail")
        else:
            print_status(pkg_name, "done")


def install_gh_extension(pkg_name: str, repo: str) -> None:
    if run("gh auth status", capture=True).returncode != 0:
        print_status(f"{pkg_name} (gh not authenticated)", "skip")
        return
    with console.status(f"Installing [bold]{pkg_name}[/bold] (gh extension)..."):
        if run(f"gh extension install {repo}", capture=True).returncode != 0:
            print_status(pkg_name, "fail")
        else:
            print_status(pkg_name, "done")


def install_eget(pkg_name: str, repo: str) -> None:
    Path.home().joinpath(".local", "bin").mkdir(parents=True, exist_ok=True)
    with console.status(f"Installing [bold]{pkg_name}[/bold] (eget)..."):
        if run(f"eget {repo} --to ~/.local/bin", capture=True).returncode != 0:
            print_status(pkg_name, "fail")
        else:
            print_status(pkg_name, "done")


def install_manual(pkg_name: str, manual: dict) -> None:
    install_type = manual["type"]
    url = manual["url"]

    if install_type == "script":
        args = manual.get("args", "")
        if args:
            cmd = f'sh -c "$(curl -fsSL {url})" "" {args}'
        else:
            cmd = f"curl -fsSL {url} | bash"
        with console.status(f"Installing [bold]{pkg_name}[/bold] (script)..."):
            if run(cmd, capture=True).returncode != 0:
                print_status(pkg_name, "fail")
            else:
                print_status(pkg_name, "done")

    elif install_type == "git_clone":
        dest = shell_expand(manual["dest"])
        Path(dest).parent.mkdir(parents=True, exist_ok=True)
        with console.status(f"Installing [bold]{pkg_name}[/bold] (git clone)..."):
            if run(f"git clone {url} {dest}", capture=True).returncode != 0:
                print_status(pkg_name, "fail")
            else:
                print_status(pkg_name, "done")


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

def ensure_zerobrew() -> None:
    """Install zerobrew if not already available."""
    if command_exists("zb"):
        print_status("zerobrew", "ok")
        return
    with console.status("Installing [bold]zerobrew[/bold]..."):
        if run("curl -fsSL https://zerobrew.rs/install | bash", capture=True).returncode != 0:
            print_status("zerobrew", "fail")
        else:
            print_status("zerobrew", "done")


def install_package(name: str, target_config: dict, methods: tuple) -> tuple[str, str]:
    """Install a single package. Returns (name, status)."""
    for method in methods:
        if method in target_config:
            value = target_config[method]
            if is_installed(name, method, value):
                return (name, "ok")
            else:
                INSTALLERS[method](name, value)
                return (name, "done")
    return (name, "skip")


MAX_PARALLEL = 4


def main() -> None:
    target = sys.argv[1] if len(sys.argv) > 1 else detect_target()
    packages_path = Path(__file__).parent / "packages.json"

    with open(packages_path) as f:
        packages = json.load(f)

    console.print(
        Panel(
            f"[bold]Target:[/bold] {target}",
            title="[bold blue]Package Installer[/bold blue]",
            border_style="blue",
        )
    )

    # Phase 0: Ensure zerobrew is available (darwin only)
    if target == "darwin" and command_exists("brew"):
        print_section("Zerobrew")
        ensure_zerobrew()

    # Phase 1: Brew taps
    taps = packages.get("_brew_taps", [])
    if taps and command_exists("brew"):
        print_section("Homebrew Taps")
        for tap in taps:
            with console.status(f"Tapping [bold]{tap}[/bold]..."):
                if run(f"brew tap {tap}", capture=True).returncode != 0:
                    print_status(tap, "fail")
                else:
                    print_status(tap, "ok")

    # Update package lists for Linux
    if target not in ("darwin",):
        if target in ("ubuntu", "pop"):
            with console.status("Updating apt package lists..."):
                run("sudo apt update", capture=True)
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

    # Phase 2: System packages — batch by method into single commands
    print_section("System Packages")
    system_items = [
        (name, tc) for name, tc in pkg_items
        if any(m in tc for m in SYSTEM_METHODS)
    ]

    # Separate already-installed from to-install, grouped by method
    brew_to_install = []
    cask_to_install = []
    apt_to_install = []
    dnf_to_install = []

    with console.status("Checking installed packages..."):
        for name, tc in system_items:
            for method in SYSTEM_METHODS:
                if method in tc:
                    value = tc[method]
                    if is_installed(name, method, value):
                        print_status(name, "ok")
                    elif method == "brew":
                        brew_to_install.append((name, value))
                    elif method == "cask":
                        cask_to_install.append((name, value))
                    elif method == "apt":
                        apt_to_install.append((name, value))
                    elif method == "dnf":
                        dnf_to_install.append((name, value))
                    break

    if brew_to_install:
        formulas = " ".join(v for _, v in brew_to_install)
        names = ", ".join(f"[bold]{n}[/bold]" for n, _ in brew_to_install)
        with console.status(f"Installing brew packages: {names}"):
            if run(f"zb install {formulas}", capture=True).returncode != 0:
                console.print("  [red]Warning: Failed to install some brew packages[/red]")
            else:
                for n, _ in brew_to_install:
                    print_status(n, "done")

    if cask_to_install:
        casks = " ".join(v for _, v in cask_to_install)
        names = ", ".join(f"[bold]{n}[/bold]" for n, _ in cask_to_install)
        with console.status(f"Installing cask packages: {names}"):
            if run(f"zb install --cask {casks}", capture=True).returncode != 0:
                console.print("  [red]Warning: Failed to install some cask packages[/red]")
            else:
                for n, _ in cask_to_install:
                    print_status(n, "done")

    if apt_to_install:
        pkgs = " ".join(v for _, v in apt_to_install)
        names = ", ".join(f"[bold]{n}[/bold]" for n, _ in apt_to_install)
        with console.status(f"Installing apt packages: {names}"):
            if run(f"sudo apt install -y {pkgs}", capture=True).returncode != 0:
                console.print("  [red]Warning: Failed to install some apt packages[/red]")
            else:
                for n, _ in apt_to_install:
                    print_status(n, "done")

    if dnf_to_install:
        pkgs = " ".join(v for _, v in dnf_to_install)
        names = ", ".join(f"[bold]{n}[/bold]" for n, _ in dnf_to_install)
        with console.status(f"Installing dnf packages: {names}"):
            if run(f"sudo dnf install -y {pkgs}", capture=True).returncode != 0:
                console.print("  [red]Warning: Failed to install some dnf packages[/red]")
            else:
                for n, _ in dnf_to_install:
                    print_status(n, "done")

    # Phase 3: Secondary packages — parallel
    print_section("Secondary Packages")
    secondary_items = [
        (name, tc) for name, tc in pkg_items
        if not any(m in tc for m in SYSTEM_METHODS)
    ]
    with ThreadPoolExecutor(max_workers=MAX_PARALLEL) as pool:
        futures = {
            pool.submit(install_package, name, tc, SECONDARY_METHODS): name
            for name, tc in secondary_items
        }
        for future in as_completed(futures):
            pass  # status printed by install_package -> INSTALLERS

    # Summary
    total = len(pkg_items)
    console.print()
    console.print(
        Panel(
            f"[bold green]Processed {total} packages[/bold green]",
            title="[bold blue]Complete[/bold blue]",
            border_style="green",
        )
    )


if __name__ == "__main__":
    main()
