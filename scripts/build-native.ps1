#!/usr/bin/env pwsh
$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
if (-not $Root) { $Root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path }

$OutDir = if ($args.Count -gt 0) { $args[0] } else { Join-Path $Root "dist" }
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

$mingw = "C:\Users\Kishan\scoop\apps\mingw\current\bin"
if (Test-Path $mingw) {
  $env:Path = "$mingw;$env:Path"
}

$env:CGO_ENABLED = "1"
Set-Location $Root

$LibName = "logger.dll"
$OutLib = Join-Path $OutDir $LibName
$HeaderSrc = Join-Path $Root "native\include\logger.h"

Write-Host "Running ABI codegen"
go run ./cmd/codegen
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "Building native shared library -> $OutLib"
go build -buildmode=c-shared -o $OutLib ./native
Copy-Item $HeaderSrc (Join-Path $OutDir "logger.h") -Force
Remove-Item (Join-Path $OutDir "liblogger.h") -ErrorAction SilentlyContinue

Get-FileHash -Algorithm SHA256 @((Join-Path $OutDir $LibName), (Join-Path $OutDir "logger.h")) |
  ForEach-Object { "$($_.Hash.ToLower())  $(Split-Path $_.Path -Leaf)" } |
  Set-Content -Path (Join-Path $OutDir "checksums.sha256")

$targets = @(
  (Join-Path $Root "bindings\python\polyglot_logger\native"),
  (Join-Path $Root "bindings\node\native"),
  (Join-Path $Root "bindings\dotnet\Polyglot.Logger\native")
)
foreach ($t in $targets) {
  New-Item -ItemType Directory -Force -Path $t | Out-Null
  Copy-Item $OutLib $t -Force
}

Write-Host "Built $LibName"
