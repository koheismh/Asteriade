param([string]$OutputDirectory = '.local\validation')
$ErrorActionPreference = 'Stop'
$RepoRoot = Split-Path -Parent $PSScriptRoot
Push-Location $RepoRoot
$PreviousCGO = $env:CGO_ENABLED
try {
    $GoVersion = (& go version)
    if ($LASTEXITCODE -ne 0 -or $GoVersion -notmatch 'go1\.27\.2 windows/amd64') {
        throw 'Go 1.27.2 for Windows amd64 is required. Install the official verified distribution first.'
    }
    New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
    New-Item -ItemType Directory -Force -Path 'bin' | Out-Null
    $env:CGO_ENABLED = '0'
    & go mod verify
    if ($LASTEXITCODE -ne 0) { throw 'Dependency verification failed.' }
    $TestOutput = @(& go test -json ./...)
    $TestExit = $LASTEXITCODE
    $TestOutput | Set-Content -Encoding UTF8 (Join-Path $OutputDirectory 'go-test.jsonl')
    if ($TestExit -ne 0) { throw 'Go tests failed. See the result file.' }
    & go build -trimpath -o 'bin\asteriade.exe' ./cmd/asteriade
    if ($LASTEXITCODE -ne 0) { throw 'Windows build failed.' }
    $Diagnosis = (& '.\bin\asteriade.exe' -diagnose | ConvertFrom-Json)
    if ($LASTEXITCODE -ne 0 -or $Diagnosis.execution_enabled -ne $false -or $Diagnosis.runner.status -ne 'unverified') {
        throw 'Fail-closed runner diagnosis did not match the expected baseline.'
    }
    $OS = Get-CimInstance Win32_OperatingSystem
    $TestEvents = @($TestOutput | ForEach-Object { $_ | ConvertFrom-Json })
    $Summary = [ordered]@{
        recorded_at = [DateTime]::UtcNow.ToString('o')
        windows_caption = $OS.Caption
        windows_version = $OS.Version
        windows_build = $OS.BuildNumber
        go_version = $GoVersion
        binary_sha256 = (Get-FileHash 'bin\asteriade.exe' -Algorithm SHA256).Hash
        passed_test_events = @($TestEvents | Where-Object { $_.Action -eq 'pass' -and $_.Test }).Count
        skipped_test_events = @($TestEvents | Where-Object { $_.Action -eq 'skip' -and $_.Test }).Count
        platform_unit_tests = 'passed'
        runner_status = 'BLOCKED'
        execution_enabled = $false
        windows_sandbox_negative_tests = 'unrun'
        codex_auth_and_descendants = 'unrun'
        note = 'Unit tests/build are not proof of OS isolation. Follow docs/WINDOWS_VALIDATION.md.'
    }
    $Summary | ConvertTo-Json -Depth 5 | Set-Content -Encoding UTF8 (Join-Path $OutputDirectory 'manifest.json')
    Write-Output 'Windows baseline tests/build recorded. Runner safety gate remains BLOCKED.'
} finally {
    $env:CGO_ENABLED = $PreviousCGO
    Pop-Location
}
