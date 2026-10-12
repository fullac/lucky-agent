#ifndef SourceRoot
  #define SourceRoot "..\..\dist\windows-installer"
#endif
#ifndef MyAppVersion
  #define MyAppVersion "dev"
#endif

[Setup]
AppId={{F7E5D335-EDC5-437B-9E6B-51FBBEA89A03}
AppName=LuckyAgent
AppVersion={#MyAppVersion}
AppPublisher=LuckyAgent
DefaultDirName={localappdata}\LuckyAgent
DefaultGroupName=LuckyAgent
DisableProgramGroupPage=yes
OutputBaseFilename=LuckyAgent-Setup-{#MyAppVersion}-x64
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
PrivilegesRequired=lowest
UninstallDisplayIcon={app}\LuckyAgent.ico
SetupIconFile=LuckyAgent.ico
ChangesEnvironment=yes

[Files]
Source: "{#SourceRoot}\lh.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "LuckyAgent.ico"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#SourceRoot}\Install-Portable.ps1"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\LuckyAgent CLI"; Filename: "{app}\lh.exe"; WorkingDir: "{app}"; IconFilename: "{app}\LuckyAgent.ico"

[Run]
Filename: "{app}\lh.exe"; Parameters: "init"; WorkingDir: "{app}"; Flags: runhidden waituntilterminated

[UninstallDelete]
Type: filesandordirs; Name: "{app}"

[Code]
const
  UserEnvironmentKey = 'Environment';

function PathContains(const Value, Entry: String): Boolean;
begin
  Result := Pos(';' + Lowercase(Entry) + ';', ';' + Lowercase(Value) + ';') > 0;
end;

procedure AddUserPath(const Entry: String);
var
  Existing, Updated: String;
begin
  if not RegQueryStringValue(HKCU, UserEnvironmentKey, 'Path', Existing) then
    Existing := '';
  if PathContains(Existing, Entry) then
    Exit;
  if Existing = '' then
    Updated := Entry
  else
    Updated := Existing + ';' + Entry;
  RegWriteExpandStringValue(HKCU, UserEnvironmentKey, 'Path', Updated);
end;

procedure RemoveUserPath(const Entry: String);
var
  Existing, Updated: String;
begin
  if not RegQueryStringValue(HKCU, UserEnvironmentKey, 'Path', Existing) then
    Exit;
  Updated := ';' + Existing + ';';
  StringChangeEx(Updated, ';' + Entry + ';', ';', True);
  while Pos(';;', Updated) > 0 do
    StringChangeEx(Updated, ';;', ';', True);
  if Updated = ';' then
    Updated := ''
  else
    Updated := Copy(Updated, 2, Length(Updated) - 2);
  RegWriteExpandStringValue(HKCU, UserEnvironmentKey, 'Path', Updated);
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then begin
    AddUserPath(ExpandConstant('{app}'));
  end;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usUninstall then begin
    RemoveUserPath(ExpandConstant('{app}'));
  end;
end;
