#!/usr/bin/env python3
"""Bootstrap script for dotfiles installation."""

import os
import platform
import shutil
import subprocess
import sys
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

REPO_URL = "https://github.com/CuriousFurBytes/.dotfiles.git"
CHEZMOI_DIR = Path.home() / ".local" / "share" / "chezmoi"


def run(cmd: str, check: bool = False, capture: bool = False) -> subprocess.CompletedProcess:
    return subprocess.run(
        cmd, shell=True, check=check, capture_output=capture, text=True
    )


def command_exists(name: str) -> bool:
    return shutil.which(name) is not None


def detect_os() -> str:
    system = platform.system()
    if system == "Darwin":
        return "mac"
    elif system == "Linux":
        return "linux"
    return f"unknown:{system}"


def print_status(name: str, status: str) -> None:
    if status == "ok":
        console.print(f"  [green]\\[ok][/green] {name}")
    elif status == "done":
        console.print(f"  [blue]\\[done][/blue] {name}")
    elif status == "fail":
        console.print(f"  [red]\\[fail][/red] {name}")


def print_section(title: str) -> None:
    console.print()
    console.rule(f"[bold]{title}[/bold]")
    console.print()


def install_homebrew(machine: str) -> None:
    """Install Homebrew on macOS if missing."""
    if machine != "mac":
        return
    if command_exists("brew"):
        print_status("Homebrew", "ok")
        return
    with console.status("Installing [bold]Homebrew[/bold]..."):
        result = run(
            '/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"'
        )
    if result.returncode != 0:
        print_status("Homebrew", "fail")
        sys.exit(1)
    print_status("Homebrew", "done")


def install_chezmoi(machine: str) -> None:
    """Install chezmoi if missing."""
    if command_exists("chezmoi"):
        print_status("chezmoi", "ok")
        return
    if machine == "mac":
        with console.status("Installing [bold]chezmoi[/bold] (brew)..."):
            result = run("brew install chezmoi", capture=True)
    else:
        with console.status("Installing [bold]chezmoi[/bold]..."):
            result = run(
                'sh -c "$(curl -fsLS get.chezmoi.io)" -- -b "$HOME/.local/bin"',
                capture=True,
            )
            os.environ["PATH"] = f"{Path.home() / '.local' / 'bin'}:{os.environ['PATH']}"
    if result.returncode != 0:
        print_status("chezmoi", "fail")
        sys.exit(1)
    print_status("chezmoi", "done")


def install_pass_cli(machine: str) -> None:
    """Install Proton Pass CLI if missing."""
    if command_exists("pass-cli"):
        print_status("Proton Pass CLI", "ok")
        return
    if machine == "mac":
        with console.status("Installing [bold]Proton Pass CLI[/bold] (brew)..."):
            run("brew tap protonpass/tap", capture=True)
            result = run("brew install protonpass/tap/pass-cli", capture=True)
    else:
        with console.status("Installing [bold]Proton Pass CLI[/bold]..."):
            result = run(
                "curl -fsSL https://proton.me/download/pass-cli/install.sh | bash",
                capture=True,
            )
    if result.returncode != 0:
        print_status("Proton Pass CLI", "fail")
        sys.exit(1)
    print_status("Proton Pass CLI", "done")


def check_auth() -> None:
    """Check that Proton Pass CLI is authenticated."""
    with console.status("Checking [bold]Proton Pass CLI[/bold] authentication..."):
        result = run("pass-cli vault list", capture=True)
    if result.returncode != 0:
        console.print()
        console.print(
            Panel(
                "[bold red]Proton Pass CLI is not authenticated.[/bold red]\n\n"
                "Please run:\n"
                "  [bold]pass-cli login[/bold]\n\n"
                "Then re-run this script.",
                title="[bold red]Authentication Required[/bold red]",
                border_style="red",
            )
        )
        sys.exit(1)
    print_status("Proton Pass CLI authenticated", "ok")


def init_chezmoi() -> None:
    """Initialize chezmoi with the dotfiles repo."""
    if (CHEZMOI_DIR / ".git").is_dir():
        print_status("Dotfiles already initialized", "ok")
        return
    with console.status(f"Initializing [bold]chezmoi[/bold] with {REPO_URL}..."):
        result = run(f"chezmoi init {REPO_URL}", capture=True)
    if result.returncode != 0:
        print_status("chezmoi init", "fail")
        sys.exit(1)
    print_status("chezmoi init", "done")


def apply_chezmoi() -> None:
    """Apply dotfiles via chezmoi (streams output)."""
    console.print("  Running [bold]chezmoi apply -v[/bold]...")
    console.print()
    result = run("chezmoi apply -v")
    if result.returncode != 0:
        print_status("chezmoi apply", "fail")
        sys.exit(1)
    print_status("chezmoi apply", "done")


def main() -> None:
    machine = detect_os()

    console.print(
        Panel(
            f"[bold]Detected OS:[/bold] {machine}",
            title="[bold blue]Dotfiles Installer[/bold blue]",
            border_style="blue",
        )
    )

    # Prerequisites
    print_section("Prerequisites")
    install_homebrew(machine)
    install_chezmoi(machine)
    install_pass_cli(machine)

    # Authentication
    print_section("Authentication")
    check_auth()

    # Initialize & apply
    print_section("Dotfiles")
    init_chezmoi()
    apply_chezmoi()

    # Done
    console.print()
    console.print(
        Panel(
            "[bold green]Installation complete![/bold green]\n\n"
            "[bold]Next steps:[/bold]\n"
            "  1. Restart your shell or run: [bold]exec $SHELL[/bold]\n"
            "  2. Verify installations with: [bold]git config --list[/bold]\n"
            "  3. Check SSH key with: [bold]ssh-add -l[/bold]",
            title="[bold blue]Complete[/bold blue]",
            border_style="green",
        )
    )


if __name__ == "__main__":
    main()
