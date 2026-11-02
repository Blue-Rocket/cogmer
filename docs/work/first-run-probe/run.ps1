# Runs one isolated `claude -p` turn with the probe plugin loaded from this folder, in a fresh
# working directory, with only project settings, and reports which hooks ran (the probe's own log),
# what the harness said of each hook (its outcome, exit code and stderr), and the model's answer.
# With -NoGit it first removes every PATH entry that holds a bash.exe or a git.exe, so that the run
# shows what Claude Code does on a Windows machine without Git for Windows. It touches no cogmer
# state and reads nothing of the user's. -Out names a folder that exists and is empty.
param([Parameter(Mandatory = $true)][string]$Out, [switch]$NoGit)
$here = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $Out
Remove-Item -ErrorAction SilentlyContinue cogmer-first-run-probe.log
if ($NoGit) {
    $kept = $env:PATH.Split(';') | Where-Object {
        $_ -and -not (Test-Path (Join-Path $_ 'bash.exe')) -and -not (Test-Path (Join-Path $_ 'git.exe'))
    }
    $env:PATH = $kept -join ';'
    "PATH without Git: bash.exe on PATH = $([bool](Get-Command bash.exe -ErrorAction SilentlyContinue))"
}
(& claude --version 2>&1) | Select-Object -First 1
$prompt = 'List, exactly as written, every line in your context that contains the text FRP-. If there are none, reply with only the word NONE.'
'' | & claude -p $prompt --plugin-dir $here --setting-sources project --output-format stream-json --verbose --include-hook-events > stream.json 2> stderr.txt
"exit status $LASTEXITCODE"
'--- the probe''s log (which hooks ran, in order)'
if (Test-Path cogmer-first-run-probe.log) { Get-Content cogmer-first-run-probe.log } else { '(no log: no hook of the probe ran)' }
'--- what the harness said of each hook'
$answer = ''
foreach ($line in Get-Content stream.json) {
    try { $e = $line | ConvertFrom-Json } catch { continue }
    if ($e.subtype -eq 'hook_response') {
        $err = "$($e.stderr)"
        if ($err.Length -gt 160) { $err = $err.Substring(0, 160) }
        "{0} outcome={1} exit={2} stderr={3}" -f $e.hook_event, $e.outcome, $e.exit_code, $err
    }
    if ($e.type -eq 'result') { $answer = "$($e.result)" }
}
'--- the model''s answer'
$answer
'--- stderr'
if (Test-Path stderr.txt) { (Get-Content stderr.txt -Raw) }
