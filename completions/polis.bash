# Bash completion for polis CLI
# Source this file in your ~/.bashrc:
#   source /path/to/polis/completions/polis.bash
#
# Or copy to ~/.local/share/bash-completion/completions/polis for auto-loading

_polis_completion() {
    local cur prev
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # All top-level commands
    local commands="about blessing clone comment discover extract follow
        index init manifest migrate migrations notifications post preview
        rebuild register render republish rotate-key snippet unfollow
        unregister version"

    # Subcommands for specific commands
    local blessing_subcommands="beseech deny grant requests sync"
    local migrations_subcommands="apply"

    # Global options
    local global_opts="--json --help"

    case $COMP_CWORD in
        1)
            # First argument: complete commands or global options
            COMPREPLY=($(compgen -W "$commands $global_opts" -- "$cur"))
            ;;
        2)
            # Second argument: depends on first
            case $prev in
                blessing)
                    COMPREPLY=($(compgen -W "$blessing_subcommands" -- "$cur"))
                    ;;
                migrations)
                    COMPREPLY=($(compgen -W "$migrations_subcommands" -- "$cur"))
                    ;;
                --json)
                    # After --json, complete with commands
                    COMPREPLY=($(compgen -W "$commands" -- "$cur"))
                    ;;
                *)
                    # For other commands, offer --json if not already used
                    COMPREPLY=($(compgen -W "--json" -- "$cur"))
                    ;;
            esac
            ;;
        3)
            # Third argument: handle --json <command> case
            if [[ "${COMP_WORDS[1]}" == "--json" ]]; then
                case $prev in
                    blessing)
                        COMPREPLY=($(compgen -W "$blessing_subcommands" -- "$cur"))
                        ;;
                    migrations)
                        COMPREPLY=($(compgen -W "$migrations_subcommands" -- "$cur"))
                        ;;
                esac
            fi
            ;;
    esac
}

complete -F _polis_completion polis
