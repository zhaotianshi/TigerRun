$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
Push-Location $Root

function Invoke-Native {
    param(
        [Parameter(Mandatory = $true)]
        [scriptblock]$Command
    )

    & $Command
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code $LASTEXITCODE"
    }
}

try {
    Invoke-Native { go run ./tools/icongen }
    Push-Location frontend
    try {
        Invoke-Native { npm install }
        Invoke-Native { npm run build }
    }
    finally {
        Pop-Location
    }
    Invoke-Native { go run ./tools/winres }
    New-Item -ItemType Directory -Force build/bin | Out-Null
    Invoke-Native { go build -tags production -trimpath -ldflags "-H windowsgui" -o "build/bin/老虎快跑.exe" . }
}
finally {
    Pop-Location
}
