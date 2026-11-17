#!/usr/bin/env bash
# Prints the probe's log, and where it was written.
d="${HOME:-/tmp}/.cogmer-probe"
echo "log directory: $d"
ls -la "$d" 2>&1 | head -20
echo "-----"
cat "$d/log.txt" 2>&1 | tail -n 400
