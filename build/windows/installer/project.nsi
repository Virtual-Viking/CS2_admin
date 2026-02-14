!include "MUI2.nsh"
!include "LogicLib.nsh"

Name "CS2 Admin"
OutFile "CS2Admin-Setup.exe"
InstallDir "$PROGRAMFILES64\CS2Admin"
InstallDirRegKey HKLM "Software\CS2Admin" "InstallDir"
RequestExecutionLevel admin

; Modern UI pages
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Section "Install"
    SetOutPath $INSTDIR
    
    ; Copy main executable
    File "CS2Admin.exe"
    
    ; Create start menu shortcuts
    CreateDirectory "$SMPROGRAMS\CS2 Admin"
    CreateShortcut "$SMPROGRAMS\CS2 Admin\CS2 Admin.lnk" "$INSTDIR\CS2Admin.exe"
    CreateShortcut "$DESKTOP\CS2 Admin.lnk" "$INSTDIR\CS2Admin.exe"
    
    ; Write uninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"
    
    ; Write registry keys
    WriteRegStr HKLM "Software\CS2Admin" "InstallDir" "$INSTDIR"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\CS2Admin" "DisplayName" "CS2 Admin"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\CS2Admin" "UninstallString" '"$INSTDIR\uninstall.exe"'
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\CS2Admin" "DisplayIcon" "$INSTDIR\CS2Admin.exe"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\CS2Admin" "Publisher" "CS2Admin"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\CS2Admin" "DisplayVersion" "${VERSION}"
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\CS2Admin" "NoModify" 1
    WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\CS2Admin" "NoRepair" 1
    
    ; Check for WebView2 Runtime
    ReadRegStr $0 HKLM "SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" "pv"
    ${If} $0 == ""
        MessageBox MB_YESNO "WebView2 Runtime is required. Download and install it now?" IDYES downloadwv2 IDNO skipwv2
        downloadwv2:
            ExecShell "open" "https://go.microsoft.com/fwlink/p/?LinkId=2124703"
        skipwv2:
    ${EndIf}
SectionEnd

Section "Uninstall"
    Delete "$INSTDIR\CS2Admin.exe"
    Delete "$INSTDIR\uninstall.exe"
    RMDir "$INSTDIR"
    
    Delete "$SMPROGRAMS\CS2 Admin\CS2 Admin.lnk"
    RMDir "$SMPROGRAMS\CS2 Admin"
    Delete "$DESKTOP\CS2 Admin.lnk"
    
    DeleteRegKey HKLM "Software\CS2Admin"
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\CS2Admin"
SectionEnd
