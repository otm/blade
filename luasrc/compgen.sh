_blade()
{
  local cur prev output header opts oifs

  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"

  # generate compleation on -f option
  if [[ ${prev} == -f ]]; then
    if  [[ $(declare -f _filedir) ]]; then
      _filedir
    else
      COMPREPLY=( $(compgen -f -- ${cur}) )
    fi
    return 0
  fi

  # extract -f flag if present and has an option
  flags=$(echo "${COMP_WORDS[@]}")
  flag=$(expr "${flags}" : '.*\(-f [^ ]* *\)')

  # change IFS from default to newline to get more stable parsing
  oifs=$IFS
  IFS=$'\n'

  # Call blade and parse out header and options
  output=$(blade $flag -compgen -comp-cwords $COMP_CWORD ${COMP_WORDS[@]})
  header=$(echo "$output" | awk '/### BEGIN COMPGEN INFO/ {p=1;next}; /### END COMPGEN INFO/ {p=0}; p')
  opts=$(echo "$output" | awk 'BEGIN{p=1} /### BEGIN COMPGEN INFO/ {p=0;next}; /### END COMPGEN INFO/ {p=1;next}; p')
  COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )

  # Parse options in header
  while read line
  do
    local foo key value
    IFS=$oifs
    read foo key value <<<$(IFS=":"; echo $line)
    IFS=$'\n'
    case $key in
      "mode" )
        case $value in
          "filedir" )
            type compopt >&/dev/null && compopt -o filenames 2> /dev/null || compgen -f /non-existing-dir/ > /dev/null
            ;;
          * )
            echo "unknown mode: " $value >&2
            ;;
        esac
        ;;
      "" )
        ;;
      * )
        echo "unknown option: " $key >&2
        ;;
    esac
  done <<< $header

  # reset IFS
  IFS=$oifs
  return 0
}
complete -F _blade blade
