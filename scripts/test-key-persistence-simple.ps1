# Simple test script to verify server key persistence

Write-Host "=== Testing Server Key Persistence ===" -ForegroundColor Cyan
Write-Host ""

# Clean up any existing data directory
if (Test-Path "data") {
    Write-Host "Cleaning up existing data directory..."
    Remove-Item -Recurse -Force data
}

# Build the server
Write-Host "Building server..."
go build -o build/server.exe ./cmd/server
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Build failed" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "=== Manual Test Instructions ===" -ForegroundColor Yellow
Write-Host ""
Write-Host "1. Run the server: .\build\server.exe"
Write-Host "2. Note the 'Server public key' in the output"
Write-Host "3. Check that 'Generated new server key pair' appears in logs"
Write-Host "4. Check that data/server-key.json was created"
Write-Host "5. Stop the server (Ctrl+C)"
Write-Host "6. Run the server again: .\build\server.exe"
Write-Host "7. Verify the same public key appears"
Write-Host "8. Check that 'Loaded existing server key' appears in logs"
Write-Host ""
Write-Host "=== Automated Quick Test ===" -ForegroundColor Yellow
Write-Host ""

# Test 1: Check key file creation
Write-Host "Starting server briefly to generate keys..."
$process = Start-Process -FilePath ".\build\server.exe" -PassThru -NoNewWindow -RedirectStandardOutput "server-log-1.txt" -RedirectStandardError "server-err-1.txt"
Start-Sleep -Seconds 2
Stop-Process -Id $process.Id -Force
Start-Sleep -Seconds 1

$log1 = Get-Content "server-log-1.txt" -Raw
if ($log1 -match "Server public key: ([A-Za-z0-9+/=]+)") {
    $publicKey1 = $matches[1]
    Write-Host "✅ First startup public key: $publicKey1" -ForegroundColor Green
} else {
    Write-Host "❌ Could not extract public key from first run" -ForegroundColor Red
    Write-Host "Log contents:"
    Get-Content "server-log-1.txt"
    exit 1
}

if (Test-Path "data/server-key.json") {
    Write-Host "✅ Key file created: data/server-key.json" -ForegroundColor Green
    $keyContent = Get-Content "data/server-key.json" | ConvertFrom-Json
    Write-Host "   Public key in file: $($keyContent.publicKey)"
} else {
    Write-Host "❌ Key file was not created" -ForegroundColor Red
    exit 1
}

# Test 2: Check key persistence
Write-Host ""
Write-Host "Starting server again to test key loading..."
$process2 = Start-Process -FilePath ".\build\server.exe" -PassThru -NoNewWindow -RedirectStandardOutput "server-log-2.txt" -RedirectStandardError "server-err-2.txt"
Start-Sleep -Seconds 2
Stop-Process -Id $process2.Id -Force
Start-Sleep -Seconds 1

$log2 = Get-Content "server-log-2.txt" -Raw
if ($log2 -match "Server public key: ([A-Za-z0-9+/=]+)") {
    $publicKey2 = $matches[1]
    Write-Host "✅ Second startup public key: $publicKey2" -ForegroundColor Green
} else {
    Write-Host "❌ Could not extract public key from second run" -ForegroundColor Red
    exit 1
}

if ($publicKey1 -eq $publicKey2) {
    Write-Host "✅ Public keys match - persistence working!" -ForegroundColor Green
} else {
    Write-Host "❌ Public keys don't match - persistence failed" -ForegroundColor Red
    Write-Host "First:  $publicKey1"
    Write-Host "Second: $publicKey2"
    exit 1
}

# Clean up
Write-Host ""
Write-Host "Cleaning up test files..."
Remove-Item -Force server-log-*.txt, server-err-*.txt -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "✅ All automated tests passed!" -ForegroundColor Green
Write-Host ""
Write-Host "You can now test manually by running: .\build\server.exe" -ForegroundColor Cyan
