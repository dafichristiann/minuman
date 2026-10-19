<#
.SYNOPSIS
    Bootstrap lingkungan pengembangan KALENGO di Windows (padanan `make bootstrap`).

.DESCRIPTION
    Urutan kerja:
      1. Periksa prasyarat (docker, go, node, npm).
      2. Buat .env dari .env.example bila belum ada.
      3. Nyalakan infrastruktur (postgres, redis, mailhog, minio).
      4. Tunggu PostgreSQL benar-benar siap menerima koneksi.
      5. go mod tidy (WAJIB sekali agar go.sum terbentuk).
      6. gofmt (merapikan format hasil tulis tangan).
      7. Jalankan migrasi lewat container migrate/migrate.
      8. Isi data contoh.
      9. npm install untuk frontend.

    Skrip ini TIDAK merusak data: migrasi bersifat menambah, seed bersifat
    idempoten (ON CONFLICT DO NOTHING). Untuk mengulang dari nol, jalankan
    `.\scripts\bootstrap.ps1 -Reset` (menghapus volume Docker KALENGO).

.PARAMETER Reset
    Hapus container + volume Docker proyek ini sebelum bootstrap. DATA HILANG.

.PARAMETER SkipSeed
    Lewati pengisian data contoh.

.EXAMPLE
    .\scripts\bootstrap.ps1
    .\scripts\bootstrap.ps1 -Reset
#>

[CmdletBinding()]
param(
    [switch]$Reset,
    [switch]$SkipSeed
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$RepoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $RepoRoot

function Write-Step([string]$Message) {
    Write-Host ""
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Write-Ok([string]$Message) {
    Write-Host "  [ok] $Message" -ForegroundColor Green
}

function Write-Warn2([string]$Message) {
    Write-Host "  [!]  $Message" -ForegroundColor Yellow
}

function Assert-Command([string]$Name, [string]$Hint) {
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Perintah '$Name' tidak ditemukan. $Hint"
    }
    Write-Ok "$Name tersedia"
}

# ------------------------------------------------------------------
Write-Step "Memeriksa prasyarat"
# ------------------------------------------------------------------
Assert-Command 'docker' 'Pasang Docker Desktop: https://docs.docker.com/desktop/windows/'
Assert-Command 'go'     'Pasang Go 1.23+: https://go.dev/dl/'
Assert-Command 'node'   'Pasang Node.js 20.11+ LTS: https://nodejs.org/'
Assert-Command 'npm'    'Ikut terpasang bersama Node.js'

# docker compose (v2) vs docker-compose (v1)
$ComposeCmd = $null
try {
    & docker compose version *> $null
    $ComposeCmd = 'docker compose'
    Write-Ok "docker compose (v2) tersedia"
} catch {
    if (Get-Command 'docker-compose' -ErrorAction SilentlyContinue) {
        $ComposeCmd = 'docker-compose'
        Write-Warn2 "Memakai docker-compose v1 (lama). Disarankan upgrade ke Docker Compose v2."
    } else {
        throw "Docker Compose tidak ditemukan. Perbarui Docker Desktop."
    }
}

$goVersion = (& go version) -replace 'go version go([0-9.]+).*', '$1'
Write-Host "  Go $goVersion"
$nodeVersion = (& node --version).TrimStart('v')
Write-Host "  Node $nodeVersion"

# ------------------------------------------------------------------
Write-Step "Menyiapkan berkas lingkungan"
# ------------------------------------------------------------------
if (Test-Path '.env') {
    Write-Ok ".env sudah ada (tidak ditimpa)"
} else {
    Copy-Item '.env.example' '.env'
    Write-Ok ".env dibuat dari .env.example — periksa nilainya sebelum produksi"
}

# ------------------------------------------------------------------
if ($Reset) {
    Write-Step "Reset: menghapus container & volume proyek"
    Write-Warn2 "Semua data PostgreSQL/Redis/MinIO lokal akan hilang."
    Invoke-Expression "$ComposeCmd down -v"
    Write-Ok "Volume dihapus"
}

# ------------------------------------------------------------------
Write-Step "Menyalakan infrastruktur"
# ------------------------------------------------------------------
Invoke-Expression "$ComposeCmd up -d postgres redis mailhog minio minio-init"
if ($LASTEXITCODE -ne 0) { throw "Gagal menyalakan infrastruktur." }
Write-Ok "Container berjalan"

# ------------------------------------------------------------------
Write-Step "Menunggu PostgreSQL siap"
# ------------------------------------------------------------------
$deadline = (Get-Date).AddSeconds(90)
$ready = $false
while ((Get-Date) -lt $deadline) {
    & docker compose exec -T postgres pg_isready -U postgres -d kalengo *> $null
    if ($LASTEXITCODE -eq 0) { $ready = $true; break }
    Start-Sleep -Seconds 2
}
if (-not $ready) {
    throw "PostgreSQL tidak siap dalam 90 detik. Cek: $ComposeCmd logs postgres"
}
Write-Ok "PostgreSQL menerima koneksi"

# ------------------------------------------------------------------
Write-Step "Menyiapkan modul Go"
# ------------------------------------------------------------------
Push-Location 'apps/api'
try {
    & go mod tidy
    if ($LASTEXITCODE -ne 0) { throw "go mod tidy gagal." }
    Write-Ok "go.sum terbentuk / diperbarui"

    & go fmt ./...
    if ($LASTEXITCODE -ne 0) { throw "go fmt gagal." }
    Write-Ok "format Go dirapikan"
} finally {
    Pop-Location
}

# ------------------------------------------------------------------
Write-Step "Menjalankan migrasi"
# ------------------------------------------------------------------
$migrateUrl = 'postgres://kalengo_migrator:migrator_dev_password@postgres:5432/kalengo?sslmode=disable'
$network = (& docker compose config --format json | ConvertFrom-Json).name
if ([string]::IsNullOrWhiteSpace($network)) { $network = 'kalengo' }

docker run --rm `
    --network "${network}_default" `
    -v "${RepoRoot}/apps/api/db/migrations:/migrations:ro" `
    migrate/migrate:v4.18.1 `
    -path=/migrations -database "$migrateUrl" up

if ($LASTEXITCODE -ne 0) {
    throw "Migrasi gagal. Pastikan container postgres sehat dan role migrator sudah dibuat."
}
Write-Ok "Migrasi selesai"

# ------------------------------------------------------------------
if (-not $SkipSeed) {
    Write-Step "Mengisi data contoh"
    Get-Content 'apps/api/db/seed/dev_seed.sql' -Raw |
        docker compose exec -T postgres psql -U kalengo_migrator -d kalengo -v ON_ERROR_STOP=1
    if ($LASTEXITCODE -ne 0) { throw "Seed gagal." }
    Write-Ok "Data contoh siap"
}

# ------------------------------------------------------------------
Write-Step "Memasang dependensi frontend"
# ------------------------------------------------------------------
Push-Location 'apps/web'
try {
    & npm install --no-audit --no-fund
    if ($LASTEXITCODE -ne 0) { throw "npm install gagal." }
    Write-Ok "node_modules siap"
} finally {
    Pop-Location
}

# ------------------------------------------------------------------
Write-Host ""
Write-Host "Bootstrap selesai." -ForegroundColor Green
Write-Host ""
Write-Host "Langkah berikutnya:"
Write-Host "  go run .\apps\api\cmd\api        # API  -> http://localhost:8080/readyz"
Write-Host "  npm --prefix apps/web run dev    # Web  -> http://localhost:5173"
Write-Host "  .\scripts\bootstrap.ps1 -Reset   # ulang dari nol (data hilang)"
Write-Host ""
Write-Host "Alternatif Docker:"
Write-Host "  docker compose up -d             # seluruh stack termasuk web+api"
Write-Host ""
