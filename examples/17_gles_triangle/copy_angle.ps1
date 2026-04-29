$chrome = "C:\Program Files\Google\Chrome\Application"
$ver = Get-ChildItem $chrome | Where-Object { $_.Name -match '^\d' } | Sort-Object Name | Select-Object -Last 1
Write-Host "Found Chrome version: $($ver.Name)"
$src = Join-Path $chrome $ver.Name
Copy-Item (Join-Path $src "libEGL.dll")    -Destination . -Force
Copy-Item (Join-Path $src "libGLESv2.dll") -Destination . -Force
Write-Host "Copied libEGL.dll and libGLESv2.dll to current directory."
