"""Classify Claude Code changelog bullets that touch the capabilities cogmer relies on,
and date each by its npm publish time. Reads CHANGELOG.md and npm.json given as args;
writes a CSV of matching bullets and prints monthly counts per family and verb."""
import json, re, sys, csv, collections
cl_path, npm_path, out_csv = sys.argv[1:4]
times = json.load(open(npm_path))['time']
families = {
  'prompt_hook': r'UserPromptSubmit',
  'stop_hook': r'\bStop hook|Stop and SubagentStop|last_assistant_message|\bStop\b hooks?',
  'session_start': r'SessionStart',
  'hook_output': r'additionalContext|hook output|hook.{0,20}stdout|systemMessage|hook_success|hook_additional',
  'hook_ids': r'session_id|transcript_path|prompt_id|CLAUDE_CODE_SESSION_ID',
  'transcript': r'transcript|session file|\.jsonl|JSONL',
  'compaction': r'compact',
  'plugin_hooks': r'plugin.{0,40}hooks?|hooks?.{0,40}plugin',
  'mcp_tool_hook': r'mcp_tool',
  'hooks_general': r'\bhooks?\b',
}
rows=[]; ver=None
for line in open(cl_path, encoding='utf-8'):
    m=re.match(r'^## (\d+\.\d+\.\d+)', line)
    if m: ver=m.group(1); continue
    if not ver or not line.startswith('- '): continue
    text=line[2:].strip()
    fams=[f for f,p in families.items() if re.search(p,text,re.I)]
    if not fams: continue
    verb=text.split()[0].lower()
    verb={'added':'added','fixed':'fixed','changed':'changed','removed':'removed','deprecated':'removed','improved':'added','reverted':'changed'}.get(verb,'other')
    ts=times.get(ver,'')
    rows.append((ver,ts[:10],verb,'|'.join(fams),text))
with open(out_csv,'w',newline='') as f:
    w=csv.writer(f); w.writerow(['version','date','verb','families','text']); w.writerows(rows)
dated=[r for r in rows if r[1]]
print('bullets matched',len(rows),'dated',len(dated))
months=sorted({r[1][:7] for r in dated}); print('months',months[0],'..',months[-1],len(months))
c=collections.Counter()
for r in dated:
    for fam in r[3].split('|'): c[(fam,r[2])]+=1
for fam in families:
    print(fam.ljust(14), {v:c[(fam,v)] for v in ['added','changed','fixed','removed','other']})
# versions per month
vm=collections.Counter(t[:7] for v,t in times.items() if re.match(r'^\d+\.\d+\.\d+$',v))
print('releases per month (last 12):', sorted(vm.items())[-12:])
