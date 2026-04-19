"""CLI entrypoint: `python -m daily_tui` or `daily-tui-py`."""

from __future__ import annotations

import sys

from . import __version__


def main() -> int:
    if len(sys.argv) > 1 and sys.argv[1] in ("-v", "--version"):
        print(f"daily-tui-py {__version__}")
        return 0
    if len(sys.argv) > 1 and sys.argv[1] in ("-h", "--help"):
        print("daily-tui-py — Python TUI for Todo + Claude commands")
        print()
        print("Usage:")
        print("  daily-tui-py            Launch the TUI")
        print("  daily-tui-py --version  Print version")
        print("  daily-tui-py --help     Show this help")
        return 0

    from .app import DailyTuiApp

    DailyTuiApp().run()
    return 0


if __name__ == "__main__":
    sys.exit(main())
