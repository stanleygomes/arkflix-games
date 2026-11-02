; Instalador do arkflix-games pro Windows (NSIS). Instala na pasta do usuário
; (sem pedir administrador), cria atalhos no Menu Iniciar e na área de
; trabalho e registra o desinstalador. Chamado por bin/package.sh windows.
Unicode true
!include "MUI2.nsh"

!ifndef VERSION
  !define VERSION "0.0.0"
!endif

Name "arkflix games"
OutFile "${OUT}"
InstallDir "$LOCALAPPDATA\Programs\arkflix-games"
InstallDirRegKey HKCU "Software\arkflix-games" "InstallDir"
RequestExecutionLevel user
SetCompressor /SOLID lzma

VIProductVersion "${VERSION}.0"
VIAddVersionKey "ProductName" "arkflix games"
VIAddVersionKey "ProductVersion" "${VERSION}"
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "FileDescription" "arkflix games: biblioteca de jogos retrô"
VIAddVersionKey "LegalCopyright" "arkflix-games"

!define MUI_FINISHPAGE_RUN "$INSTDIR\arkflix-games.exe"
!define MUI_FINISHPAGE_RUN_TEXT "Abrir o arkflix games"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "PortugueseBR"

Section "arkflix games"
  SetOutPath "$INSTDIR"
  File "${SRC}\arkflix-games.exe"
  WriteUninstaller "$INSTDIR\desinstalar.exe"
  CreateDirectory "$SMPROGRAMS\arkflix games"
  CreateShortcut "$SMPROGRAMS\arkflix games\arkflix games.lnk" "$INSTDIR\arkflix-games.exe"
  CreateShortcut "$DESKTOP\arkflix games.lnk" "$INSTDIR\arkflix-games.exe"
  WriteRegStr HKCU "Software\arkflix-games" "InstallDir" "$INSTDIR"
  !define UNINST "Software\Microsoft\Windows\CurrentVersion\Uninstall\arkflix-games"
  WriteRegStr HKCU "${UNINST}" "DisplayName" "arkflix games"
  WriteRegStr HKCU "${UNINST}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKCU "${UNINST}" "DisplayIcon" "$INSTDIR\arkflix-games.exe"
  WriteRegStr HKCU "${UNINST}" "UninstallString" "$\"$INSTDIR\desinstalar.exe$\""
  WriteRegDWORD HKCU "${UNINST}" "NoModify" 1
  WriteRegDWORD HKCU "${UNINST}" "NoRepair" 1
SectionEnd

; Desinstalar tira o programa; configurações, perfis e saves
; (%AppData%\arkflix-games) ficam.
Section "Uninstall"
  Delete "$INSTDIR\arkflix-games.exe"
  Delete "$INSTDIR\desinstalar.exe"
  RMDir "$INSTDIR"
  Delete "$SMPROGRAMS\arkflix games\arkflix games.lnk"
  RMDir "$SMPROGRAMS\arkflix games"
  Delete "$DESKTOP\arkflix games.lnk"
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\arkflix-games"
  DeleteRegKey HKCU "Software\arkflix-games"
SectionEnd
