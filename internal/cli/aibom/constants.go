package aibom

const (
	// PassthroughAnnotation marks a Cobra command whose flags, args, and --help
	// are delegated to the embedded cisco-aibom analyzer (DisableFlagParsing).
	// The in-process smoke test skips annotated commands because their help is
	// rendered by the Python analyzer, which is only present in embedded builds.
	PassthroughAnnotation = "aibom_passthrough"

	// HelpOrderAnnotation carries a zero-padded sort index so the panel help
	// lists subcommands in a defined order (analyzer passthroughs first, in the
	// analyzer's own order; aid_cli cloud-management commands grouped last)
	// instead of alphabetically. The generic renderer falls back to
	// alphabetical for commands without it.
	HelpOrderAnnotation = "aibom_help_order"

	// aibomIntegrationID is the config.spec.yaml section id for local AIBOM
	// settings.
	aibomIntegrationID = "aibom"

	// progName is the single source of truth for the user-facing AI BOM program
	// name. It drives the usage lines (Typer prog_name), the help-panel titles,
	// and the help wording — change it here to rebrand the whole surface.
	progName = "cisco-aid-cli ai-bom"

	// envHelpPreamble, envHelpEpilogue, and envProgName carry the program name
	// and aid_cli help framing to the bootstrap. The Python side renders the
	// framing as Rich panels (only on a help request) so they match the rest of
	// the help formatting (same box style and width).
	envHelpPreamble = "AID_BOM_HELP_PREAMBLE"
	envHelpEpilogue = "AID_BOM_HELP_EPILOGUE"
	envProgName     = "AID_BOM_PROG_NAME"
)

// showHelpFraming gates the aid_cli-owned preamble/epilogue panels around the
// passthrough analyzer help. It is OFF: we currently let the analyzer's own
// --help stand alone, and the parent `ai-bom --help` (rendered by the Go panel
// renderer) shows no framing either, so this keeps all ai-bom help uniform.
// Flip to true to re-enable the helpPreamble/helpEpilogue panels below.
const showHelpFraming = false

// helpPreamble / helpEpilogue are aid_cli-owned context shown around the command
// help when showHelpFraming is enabled. They intentionally read as native Cisco
// AI Defense text (no mention of an embedded/third-party analyzer). Retained
// (even while disabled) so the framing can be restored with a one-line flip.
const helpPreamble = "Cisco AI Defense — generate and inspect an AI Bill of Materials. " +
	"Credentials and settings are resolved from flags, environment, and config.toml; " +
	"any flag you pass takes precedence."

const helpEpilogue = "Related commands:\n" +
	"  " + progName + " list | info | components | summary | delete   (AI Defense inventory)\n" +
	"  " + progName + " report upload <report.json>                    (endpoint + key from config)"

// aibomBootstrap runs the embedded Typer app with an explicit program name
// (from AID_BOM_PROG_NAME) so usage lines read "<progName> <cmd> ..." instead of
// the embedded tool's own name. On a help request it brackets the help with
// aid_cli context, rendered via Rich panels (titled with the program name) so
// the framing matches the rest of the help. setrecursionlimit mirrors the
// embedded cli_entry_point. The original exit code is preserved and re-raised.
const aibomBootstrap = `import os, sys
from aibom.cli import app

_args = sys.argv[1:]
_is_help = ("--help" in _args) or ("-h" in _args)
_prog = os.environ.get("AID_BOM_PROG_NAME", "cisco-aid-cli ai-bom")


def _panel(env_key, title):
    body = os.environ.get(env_key)
    if not body:
        return
    try:
        from rich.console import Console
        from rich.panel import Panel
        Console().print(Panel(body, title=title, title_align="left", border_style="dim"))
    except Exception:
        print(body)


if _is_help:
    _panel("AID_BOM_HELP_PREAMBLE", _prog)

sys.setrecursionlimit(max(sys.getrecursionlimit(), 5000))
_code = 0
try:
    app(prog_name=_prog)
except SystemExit as exc:
    _code = exc.code if isinstance(exc.code, int) else (0 if exc.code is None else 1)

if _is_help:
    _panel("AID_BOM_HELP_EPILOGUE", _prog)

sys.exit(_code)`