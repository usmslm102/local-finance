#ifndef AppVersion
  #define AppVersion "0.0.0"
#endif
#ifndef TargetArch
  #define TargetArch "amd64"
#endif
[Setup]
AppId={{DD9229C3-85E8-40E2-98DE-420C25C12596}
AppName=LocalFinance
AppVersion={#AppVersion}
DefaultDirName={localappdata}\Programs\LocalFinance
DefaultGroupName=LocalFinance
PrivilegesRequired=lowest
OutputDir=..\..\dist\desktop
OutputBaseFilename=LocalFinance-Desktop-windows-{#TargetArch}-setup
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
SetupIconFile=..\..\dist\desktop\AppIcon.ico
#if TargetArch == "arm64"
ArchitecturesAllowed=arm64
ArchitecturesInstallIn64BitMode=arm64
#else
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
#endif
UninstallDisplayIcon={app}\LocalFinance.exe
[Files]
Source: "..\..\dist\desktop\local-finance-desktop.exe"; DestDir: "{app}"; DestName: "LocalFinance.exe"; Flags: ignoreversion
[Icons]
Name: "{group}\LocalFinance"; Filename: "{app}\LocalFinance.exe"
Name: "{autodesktop}\LocalFinance"; Filename: "{app}\LocalFinance.exe"; Tasks: desktopicon
[Tasks]
Name: desktopicon; Description: "Create a desktop shortcut"; Flags: unchecked
[Run]
Filename: "{app}\LocalFinance.exe"; Description: "Open LocalFinance"; Flags: nowait postinstall skipifsilent
; Do not remove ~/.localfinance during uninstall. WebView2 must be installed
; beforehand; application startup never downloads it or any finance data.
