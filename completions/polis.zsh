#compdef polis
# Zsh completion for polis CLI
# Add to your fpath or source in ~/.zshrc:
#   fpath=(/path/to/polis/completions $fpath)
#   autoload -Uz compinit && compinit
#
# Or copy to ~/.zsh/completions/_polis (create dir if needed)

_polis() {
    local -a commands blessing_subcommands migrations_subcommands

    commands=(
        'about:Show site, versions, config, keys, discovery info'
        'blessing:Manage comment blessings'
        'clone:Clone a remote polis site'
        'comment:Create a comment on a post'
        'discover:Check followed authors for new content'
        'extract:Reconstruct a specific version of a file'
        'follow:Follow an author'
        'index:View content index'
        'init:Initialize Polis directory structure'
        'manifest:Regenerate manifest.json'
        'migrate:Migrate content to a new domain'
        'migrations:Apply discovered domain migrations'
        'notifications:Show pending actions'
        'post:Create a new post'
        'preview:Preview a post or comment with signature verification'
        'rebuild:Rebuild local indexes'
        'register:Register site with discovery service'
        'render:Render markdown to HTML'
        'republish:Update an already-published file'
        'rotate-key:Generate new keypair and re-sign content'
        'snippet:Publish a reusable template snippet'
        'unfollow:Unfollow an author'
        'unregister:Unregister site from discovery service'
        'version:Print CLI version'
    )

    blessing_subcommands=(
        'beseech:Re-request blessing by content hash'
        'deny:Deny a blessing request'
        'grant:Grant a blessing request'
        'requests:List pending blessing requests'
        'sync:Sync auto-blessed comments from discovery service'
    )

    migrations_subcommands=(
        'apply:Apply discovered domain migrations'
    )

    _arguments -C \
        '--json[Output results in JSON format]' \
        '--help[Show help]' \
        '1: :->command' \
        '2: :->subcommand' \
        '*: :->args'

    case $state in
        command)
            _describe -t commands 'polis commands' commands
            ;;
        subcommand)
            case $words[2] in
                blessing)
                    _describe -t subcommands 'blessing subcommands' blessing_subcommands
                    ;;
                migrations)
                    _describe -t subcommands 'migrations subcommands' migrations_subcommands
                    ;;
            esac
            ;;
    esac
}

_polis "$@"
