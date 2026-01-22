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
        index init migrate migrations notifications post preview
        rebuild register render republish rotate-key snippet unfollow
        unregister version"

    # Subcommands for specific commands
    local blessing_subcommands="beseech deny grant requests sync"
    local migrations_subcommands="apply"
    local notifications_subcommands="list read dismiss sync config"

    # Options for specific commands
    local notifications_list_opts="--type --all --json"
    local notifications_read_opts="--all --json"
    local notifications_dismiss_opts="--older-than --json"
    local notifications_sync_opts="--reset --json"
    local notifications_config_opts="--poll-interval --enable --disable --mute --unmute --json"
    local follow_opts="--announce --json"
    local unfollow_opts="--announce --json"
    local render_opts="--force --init-templates --json"
    local rebuild_opts="--posts --comments --notifications --all --json"
    local init_opts="--site-title --register --posts-dir --comments-dir --keys-dir --snippets-dir --versions-dir --themes-dir --json"
    local unregister_opts="--force --json"
    local clone_opts="--full --diff --json"
    local discover_opts="--author --since --json"
    local rotate_key_opts="--delete-old-key --json"
    local post_opts="--filename --title --json"
    local comment_opts="--filename --title --json"
    local snippet_opts="--filename --title --json"

    # Global options
    local global_opts="--json --help"

    # Determine actual command position (skip --json if it's first)
    local cmd_pos=1
    local actual_cmd=""
    if [[ "${COMP_WORDS[1]}" == "--json" ]]; then
        cmd_pos=2
    fi
    if [[ $COMP_CWORD -ge $cmd_pos ]]; then
        actual_cmd="${COMP_WORDS[$cmd_pos]}"
    fi

    # Calculate effective position relative to command
    local effective_pos=$((COMP_CWORD - cmd_pos))

    case $COMP_CWORD in
        1)
            # First argument: complete commands or global options
            COMPREPLY=($(compgen -W "$commands $global_opts" -- "$cur"))
            ;;
        *)
            # Handle --json as first argument
            if [[ "${COMP_WORDS[1]}" == "--json" && $COMP_CWORD -eq 2 ]]; then
                COMPREPLY=($(compgen -W "$commands" -- "$cur"))
                return
            fi

            # Handle command-specific completions
            case $actual_cmd in
                blessing)
                    if [[ $effective_pos -eq 1 ]]; then
                        COMPREPLY=($(compgen -W "$blessing_subcommands --json" -- "$cur"))
                    fi
                    ;;
                migrations)
                    if [[ $effective_pos -eq 1 ]]; then
                        COMPREPLY=($(compgen -W "$migrations_subcommands --json" -- "$cur"))
                    fi
                    ;;
                notifications)
                    if [[ $effective_pos -eq 1 ]]; then
                        COMPREPLY=($(compgen -W "$notifications_subcommands --json" -- "$cur"))
                    elif [[ $effective_pos -ge 2 ]]; then
                        local subcmd="${COMP_WORDS[$((cmd_pos + 1))]}"
                        case $subcmd in
                            list)
                                COMPREPLY=($(compgen -W "$notifications_list_opts" -- "$cur"))
                                ;;
                            read)
                                COMPREPLY=($(compgen -W "$notifications_read_opts" -- "$cur"))
                                ;;
                            dismiss)
                                COMPREPLY=($(compgen -W "$notifications_dismiss_opts" -- "$cur"))
                                ;;
                            sync)
                                COMPREPLY=($(compgen -W "$notifications_sync_opts" -- "$cur"))
                                ;;
                            config)
                                COMPREPLY=($(compgen -W "$notifications_config_opts" -- "$cur"))
                                ;;
                        esac
                    fi
                    ;;
                follow)
                    COMPREPLY=($(compgen -W "$follow_opts" -- "$cur"))
                    ;;
                unfollow)
                    COMPREPLY=($(compgen -W "$unfollow_opts" -- "$cur"))
                    ;;
                render)
                    COMPREPLY=($(compgen -W "$render_opts" -- "$cur"))
                    ;;
                rebuild)
                    COMPREPLY=($(compgen -W "$rebuild_opts" -- "$cur"))
                    ;;
                init)
                    COMPREPLY=($(compgen -W "$init_opts" -- "$cur"))
                    ;;
                unregister)
                    COMPREPLY=($(compgen -W "$unregister_opts" -- "$cur"))
                    ;;
                clone)
                    COMPREPLY=($(compgen -W "$clone_opts" -- "$cur"))
                    ;;
                discover)
                    COMPREPLY=($(compgen -W "$discover_opts" -- "$cur"))
                    ;;
                rotate-key)
                    COMPREPLY=($(compgen -W "$rotate_key_opts" -- "$cur"))
                    ;;
                post)
                    COMPREPLY=($(compgen -W "$post_opts" -- "$cur"))
                    ;;
                comment)
                    COMPREPLY=($(compgen -W "$comment_opts" -- "$cur"))
                    ;;
                snippet)
                    COMPREPLY=($(compgen -W "$snippet_opts" -- "$cur"))
                    ;;
                republish|preview|extract|migrate)
                    COMPREPLY=($(compgen -W "--json" -- "$cur"))
                    ;;
                about|version|index|register)
                    COMPREPLY=($(compgen -W "--json" -- "$cur"))
                    ;;
            esac
            ;;
    esac
}

complete -F _polis_completion polis
