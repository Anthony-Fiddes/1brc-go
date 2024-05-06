#!/usr/bin/fish

go build .

set -l commands
for revision in (seq 0 6)
    set -a commands "'./1brc-go --revision $revision measurements.txt'"
end

# no clue why it needs eval but it does. Otherwise it complains about the file
# not existing.
eval "hyperfine --warmup 3 --runs 10 $commands --export-markdown benchmark.md"
