from pygments.token import (
    Comment, Error, Generic, Keyword, Name, Number,
    Operator, Punctuation, String, Token,
)
from IPython.utils.PyColorize import Theme, theme_table

# Rosé Pine palette
_MUTED  = "#6e6a86"
_SUBTLE = "#908caa"
_TEXT   = "#e0def4"
_LOVE   = "#eb6f92"
_GOLD   = "#f6c177"
_ROSE   = "#ebbcba"
_PINE   = "#31748f"
_FOAM   = "#9ccfd8"
_IRIS   = "#c4a7e7"

theme_table["rose-pine"] = Theme(
    "rose-pine",
    base=None,
    extra_style={
        Token:                      _TEXT,
        Comment:                    f"italic {_MUTED}",
        Keyword:                    f"italic {_LOVE}",
        Keyword.Constant:           f"italic {_IRIS}",
        Keyword.Declaration:        f"italic {_LOVE}",
        Keyword.Namespace:          _FOAM,
        Keyword.Type:               f"italic {_GOLD}",
        Name:                       _TEXT,
        Name.Attribute:             _ROSE,
        Name.Builtin:               _FOAM,
        Name.Builtin.Pseudo:        f"italic {_FOAM}",
        Name.Class:                 _FOAM,
        Name.Decorator:             _IRIS,
        Name.Exception:             _LOVE,
        Name.Function:              f"italic {_PINE}",
        Name.Function.Magic:        f"italic {_PINE}",
        Number:                     _GOLD,
        Operator:                   f"bold {_FOAM}",
        Operator.Word:              f"bold italic {_LOVE}",
        Punctuation:                _SUBTLE,
        String:                     _PINE,
        String.Doc:                 f"italic {_MUTED}",
        String.Escape:              _ROSE,
        String.Interpol:            _ROSE,
        Error:                      _LOVE,
        Generic.Deleted:            _LOVE,
        Generic.Emph:               f"italic {_ROSE}",
        Generic.Error:              _LOVE,
        Generic.Heading:            f"bold {_PINE}",
        Generic.Inserted:           _PINE,
        Generic.Strong:             f"bold {_TEXT}",
        Generic.Subheading:         f"bold {_IRIS}",
        Generic.Traceback:          _LOVE,
        Token.Prompt:               _PINE,
        Token.PromptNum:            f"bold {_FOAM}",
        Token.OutPrompt:            _IRIS,
        Token.OutPromptNum:         f"bold {_IRIS}",
        Token.Lineno:               _SUBTLE,
        Token.LinenoEm:             f"bold {_SUBTLE}",
        Token.ValEm:                f"bold {_GOLD}",
        Token.VName:                _ROSE,
        Token.Filename:             _FOAM,
        Token.FilenameEm:           f"bold {_FOAM}",
        Token.ExcName:              f"bold {_LOVE}",
        Token.Topline:              _SUBTLE,
        Token.Caret:                "",
    },
    symbols={
        "arrow_body": "─",
        "arrow_head": "▶",
        "top_line": "─",
    },
)

c = get_config()  # noqa

# Theme
c.InteractiveShell.colors = 'rose-pine'

# Enable truecolor (Ghostty supports it)
c.TerminalInteractiveShell.true_color = True

# Use vi editing mode (consistent with Helix)
c.TerminalInteractiveShell.editing_mode = 'vi'

# Auto-close brackets and quotes
c.TerminalInteractiveShell.auto_match = True

# Don't prompt on Ctrl-D exit
c.TerminalInteractiveShell.confirm_exit = False

# Show assigned values too (e.g. `x = 1` prints the value)
c.InteractiveShell.ast_node_interactivity = 'last_expr_or_assign'

# Autoreload all modules before execution
c.InteractiveShellApp.extensions = ['autoreload']
c.InteractiveShellApp.exec_lines = ['%autoreload 2']

# Warn if IPython is not installed in the active venv
c.InteractiveShell.warn_venv = True
