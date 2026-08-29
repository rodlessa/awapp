# bash completion for awapp
_awapp() {
    local cur prev
    cur="${COMP_WORDS[COMP_CWORD]}"
    local opts="
        --apikey --city --use-ip --no-ip-prompt --interval --fps
        -f --color --stars --light-key --moon --phase --eclipse
        --eclipse-duration --solar-eclipse --solar-eclipse-duration
        --size --season --leaves --save-config --list-config
        --config --version -h --help
    "
    COMPREPLY=( $(compgen -W "$opts" -- "$cur") )
}
complete -F _awapp awapp
