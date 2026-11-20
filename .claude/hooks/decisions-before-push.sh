#!/usr/bin/env bash
# A PreToolUse hook on Bash. Before a `git push` that sends added or changed decisions,
# it asks for the decision-consistency skill unless that check has run for the commit
# being pushed. It fails open: anything it cannot work out lets the push through.
input=$(cat)
cmd=$(printf '%s' "$input" | python3 -I -c 'import json,sys; print(json.load(sys.stdin).get("tool_input",{}).get("command",""))' 2>/dev/null) || exit 0
case "$cmd" in *"git push"*) ;; *) exit 0 ;; esac

cd "${CLAUDE_PROJECT_DIR:-.}" 2>/dev/null || exit 0
git rev-parse --git-dir >/dev/null 2>&1 || exit 0
base=$(git rev-parse --abbrev-ref --symbolic-full-name '@{upstream}' 2>/dev/null) || base=origin/main
changed=$(git diff --name-only --diff-filter=AMR "$base"..HEAD -- docs/decisions/ 2>/dev/null |
  grep -v -e '/areas\.md$' -e '/README\.md$') || exit 0
[ -n "$changed" ] || exit 0
head=$(git rev-parse HEAD) || exit 0
[ -e "$(git rev-parse --git-dir)/decision-consistency/$head" ] && exit 0

list=$(printf '%s' "$changed" | tr '\n' ' ')
reason="This push changes decisions that have not been checked against the other active decisions: $list. Run the decision-consistency skill on them, which runs in a subagent and records that it ran, report its findings to the user, and push again only if the user still wants to."
python3 -I -c 'import json,sys; print(json.dumps({"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":sys.argv[1]}}))' "$reason"
