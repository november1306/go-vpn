# Test script to verify server key persistence

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

# Start server for the first time
Write-Host ""
Write-Host "=== Test 1: First startup (should generate new keys) ===" -ForegroundColor Yellow
$serverPath = Join-Path $PWD "build\server.exe"
$job1 = Start-Job -ScriptBlock { param($path) & $path 2>&1 } -ArgumentList $serverPath
Start-Sleep -Seconds 3
Stop-Job $job1
$output1 = Receive-Job $job1
Remove-Job $job1
$output1 | Out-File -FilePath "server-log-1.txt"

# Extract the public key from first run
$publicKey1 = ($output1 | Select-String "Server public key:").Line -replace ".*Server public key: ", ""
Write-Host "First startup public key: $publicKey1"

# Check if log contains "Generated new server key pair"
if ($output1 -match "Generated new server key pair") {
    Write-Host "✅ Server logged new key generation" -ForegroundColor Green
} else {
    Write-Host "❌ Server did not log new key generation" -ForegroundColor Red
}

# Check if server-key.json was created
if (Test-Path "data/server-key.json") {
    Write-Host "✅ Key file created: data/server-key.json" -ForegroundColor Green
    Write-Host "Key file contents:"
    Get-Content "data/server-key.json"
} else {
    Write-Host "❌ Key file was not created" -ForegroundColor Red
    exit 1
}

# Start server for the second time
Write-Host ""
Write-Host "=== Test 2: Second startup (should load existing keys) ===" -ForegroundColor Yellow
$job2 = Start-Job -ScriptBlock { param($path) & $path 2>&1 } -ArgumentList $serverPath
Start-Sleep -Seconds 3
Stop-Job $job2
$output2 = Receive-Job $job2
Remove-Job $job2
$output2 | Out-File -FilePath "server-log-2.txt"

# Extract the public key from second run
$publicKey2 = ($output2 | Select-String "Server public key:").Line -replace ".*Server public key: ", ""
Write-Host "Second startup public key: $publicKey2"

# Compare public keys
if ($publicKey1 -eq $publicKey2) {
    Write-Host "✅ Public keys match - persistence working!" -ForegroundColor Green
} else {
    Write-Host "❌ Public keys don't match - persistence failed" -ForegroundColor Red
    Write-Host "First:  $publicKey1"
    Write-Host "Second: $publicKey2"
    exit 1
}

# Check for "Loaded existing server key" message
if ($output2 -match "Loaded existing server key") {
    Write-Host "✅ Server logged that it loaded existing keys" -ForegroundColor Green
} else {
    Write-Host "❌ Server did not log loading existing keys" -ForegroundColor Red
    exit 1
}

# Test 3: Corrupt the key file
Write-Host ""
Write-Host "=== Test 3: Corrupted key file (should fail with clear error) ===" -ForegroundColor Yellow
"invalid json" | Out-File -FilePath "data/server-key.json"
$job3 = Start-Job -ScriptBlock { param($path) & $path 2>&1 } -ArgumentList $serverPath
Start-Sleep -Seconds 3
Stop-Job $job3
$output3 = Receive-Job $job3
Remove-Job $job3
$output3 | Out-File -FilePath "server-log-3.txt"

if ($output3 -match "failed to load server keys") {
    Write-Host "✅ Server failed with clear error message for corrupted file" -ForegroundColor Green
} else {
    Write-Host "❌ Server did not handle corrupted file properly" -ForegroundColor Red
    Write-Host "Output:"
    $output3
    exit 1
}

# Clean up
Write-Host ""
Write-Host "=== Cleanup ===" -ForegroundColor Cyan
Remove-Item -Force server-log-*.txt -ErrorAction SilentlyContinue
Remove-Item -Recurse -Force data -ErrorAction SilentlyContinue
Remove-Item -Force build/server.exe -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "✅ All tests passed!" -ForegroundColor Green
