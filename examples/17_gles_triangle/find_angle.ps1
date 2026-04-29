# Search common locations for ANGLE DLLs
$searchRoots = @(
    "C:\Program Files",
    "C:\Program Files (x86)",
    "$env:LOCALAPPDATA",
    "$env:APPDATA"
)

Write-Host "Searching for libEGL.dll ..."
foreach ($root in $searchRoots) {
    if (Test-Path $root) {
        Get-ChildItem -Path $root -Filter "libEGL.dll" -Recurse -ErrorAction SilentlyContinue |
            Select-Object -ExpandProperty FullName |
            ForEach-Object { Write-Host "  FOUND: $_" }
    }
}
Write-Host "Done."
