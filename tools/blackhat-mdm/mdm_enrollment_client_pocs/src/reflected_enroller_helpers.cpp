#include "common.h"

MIDL_INTERFACE("EA03163E-5836-4FFD-84AF-9DFE0E58E7F9")
FindDiscoveryResults : public IInspectable{
  virtual HRESULT Proc1(HSTRING * p0) = 0;
  virtual HRESULT Proc2(BYTE* p0) = 0;
};

MIDL_INTERFACE("C7EBA020-AF80-4B6B-B2DC-329F13A8B101")
DiscoverEndpointsResults : public IInspectable{
  virtual HRESULT Proc1(int* p0) = 0;
  virtual HRESULT Proc2(int* p0) = 0;
  virtual HRESULT Proc3(HSTRING* p0) = 0;
  virtual HRESULT Proc4(HSTRING* p0) = 0;
  virtual HRESULT Proc5(HSTRING* p0) = 0;
};

MIDL_INTERFACE("C5466834-857E-45BF-920D-0E9C1E8CA703")
CustomAllDonePageResults : public IInspectable{
  virtual HRESULT Proc6(HSTRING * p0) = 0;
  virtual HRESULT Proc7(HSTRING* p0) = 0;
  virtual HRESULT Proc8(HSTRING* p0) = 0;
  virtual HRESULT Proc9(HSTRING* p0) = 0;
};

MIDL_INTERFACE("FD0D03BA-0AEE-4FF7-8990-689AFBF63A20")
ReflectedEnrollmentResult : public IInspectable{
  virtual HRESULT Proc1(GUID * p0) = 0;
  virtual HRESULT Proc2(HSTRING* p0) = 0;
  virtual HRESULT Proc3(int* p0) = 0;
  virtual HRESULT Proc4(HSTRING* p0) = 0;
  virtual HRESULT Proc5(int* p0) = 0;
};

template <>
MIDL_INTERFACE("FA7717FD-0583-5A8D-AE87-78C26D9BB9C8")
IAsyncOperationCompletedHandler<ReflectedEnrollmentResult> : public IInspectable{ 
  virtual HRESULT Proc1(IAsyncOperation<ReflectedEnrollmentResult>*p0, int p1) = 0;
};

template <>
MIDL_INTERFACE("E5BB59E8-7790-598A-BC6B-457E588370CE")
IAsyncOperation<ReflectedEnrollmentResult> : public IInspectable{
  virtual HRESULT Proc6(IAsyncOperationCompletedHandler<ReflectedEnrollmentResult>*p0) = 0;
  virtual HRESULT Proc7(IAsyncOperationCompletedHandler<ReflectedEnrollmentResult>** p0) = 0;
  virtual HRESULT Proc8(ReflectedEnrollmentResult** p0) = 0;
};

MIDL_INTERFACE("3490F9C9-9703-46D0-B778-1EC23B82F926")
ReflectedEnrollment : public IInspectable{
  virtual HRESULT FindDiscoveryServiceAsync(HSTRING p0, BYTE p1, IAsyncOperation<FindDiscoveryResults>**p2) = 0;
  virtual HRESULT DiscoverEndpointsAsync(HSTRING p0, HSTRING p1, BYTE p2, IAsyncOperation<DiscoverEndpointsResults>** p3) = 0;
  virtual HRESULT EnrollAsync(HSTRING p0, HSTRING p1, HSTRING p2, int p3, HSTRING p4, HSTRING p5, HSTRING p6, HSTRING p7, int p8, HSTRING p9, IAsyncOperation<ReflectedEnrollmentResult>** p10) = 0;
  virtual HRESULT AllowAuthUri(void* p0) = 0;
  virtual HRESULT RemoveAuthUriAllowList() = 0;
  virtual HRESULT EventWriteForEnrollment(int p0, int p1) = 0;
  virtual HRESULT RetrieveCustomAllDonePageAsync(IAsyncOperation<CustomAllDonePageResults>** p0) = 0;
  virtual HRESULT SetEnrollmentAsDormant(HSTRING p0, int p1, int p2, IAsyncAction** p3) = 0;
  virtual HRESULT CompleteMAMToMDMUpgrade(HSTRING p0, HSTRING p1, int p2, IAsyncAction** p3) = 0;
  virtual HRESULT GetEnrollment(int p0, IAsyncOperation<HSTRING>** p1) = 0;
  virtual HRESULT CreateCorrelationVector(IAsyncOperation<HSTRING>** p0) = 0;
  virtual HRESULT CheckForDomainControllerConnectivity(int p0, IAsyncAction** p1) = 0;
  virtual HRESULT ShowMdmSyncStatusPageAsync(int p0, IAsyncOperation<INT32>** p1) = 0;
  virtual HRESULT PollForExpectedPoliciesAndResources(int p0, int p1, int p2, void** p3) = 0;
  virtual HRESULT UpdateServerWithResult(int p0, int p1) = 0;
  virtual HRESULT StartPollingTask() = 0;
  virtual HRESULT ClearAutoLoginData() = 0;
  virtual HRESULT SetWasContinuedAnyway(int p0) = 0;
  virtual HRESULT CheckMDMProgressModeAsync(void** p0) = 0;
  virtual HRESULT CheckBlockingValueAsync(IAsyncOperation<INT32>** p0) = 0;
  virtual HRESULT ShouldShowCollectLogsAsync(int p0, IAsyncOperation<INT32>** p1) = 0;
  virtual HRESULT CollectLogs(HSTRING p0, IAsyncAction** p1) = 0;
  virtual HRESULT ResetProgressTimeout(int p0) = 0;
  virtual HRESULT RetrieveCustomErrorText(int p0, IAsyncOperation<HSTRING>** p1) = 0;
  virtual HRESULT AADEnrollAsync(HSTRING p0, HSTRING p1, HSTRING p2, HSTRING p3, int p4, HSTRING p5, HSTRING p6, HSTRING p7, IAsyncOperation<ReflectedEnrollmentResult>** p8) = 0;
  virtual HRESULT AADEnrollAsyncWithTenantId(HSTRING p0, HSTRING p1, HSTRING p2, HSTRING p3, int p4, HSTRING p5, HSTRING p6, HSTRING p7, HSTRING p8, void** p9) = 0;
  virtual HRESULT UnenrollAsync(HSTRING p0, IAsyncAction** p1) = 0;
  virtual HRESULT AADCredentialsEnrollAsync(int p0, IAsyncAction** p1) = 0;
  virtual HRESULT PrepForFirstSignin() = 0;
  virtual HRESULT CheckRebootRequiredAsync(IAsyncOperation<boolean>** p0) = 0;
  virtual HRESULT RebuildSchedulesAndSyncWithServerAsync(IAsyncAction** p0) = 0;
  virtual HRESULT RecreateEnrollmentTasksAsync(IAsyncAction** p0) = 0;
  virtual HRESULT ForceRunDeviceRegistrationScheduledTaskAsync(IAsyncAction** p0) = 0;
  virtual HRESULT CollectLogsEx(HSTRING p0, HSTRING p1, IAsyncAction** p2) = 0;
  virtual HRESULT AADUnregisterAsync(IAsyncAction** p0) = 0;
  virtual HRESULT CollectOneTrace(HSTRING p0, IAsyncAction** p1) = 0;
  virtual HRESULT MmpcGetManagementUrlsAsync(HSTRING p0, HSTRING p1, void** p2) = 0;
  virtual HRESULT GetSyncFailureTimeout(IAsyncOperation<INT32>** p0) = 0;
};



HRESULT EnrollAADUsingReflectedEnroller(const std::wstring& upn, const std::wstring& discovery_service_full_url)
{
  if (discovery_service_full_url.empty()) {
    return E_INVALIDARG;
  }

  // Initializing WinRT stack
  RoInitializeWrapper init(RO_INIT_MULTITHREADED);
  if (FAILED((HRESULT)init)) return PrintError((HRESULT)init);

  // Setting input hstrings
  HSTRING upn_hstr = NULL;
  HRESULT hr = WindowsCreateString(upn.c_str(), (UINT32)upn.length(), &upn_hstr);
  if (FAILED(hr)) return PrintError(hr);

  HSTRING discovery_url_hstr = NULL;
  hr = WindowsCreateString(discovery_service_full_url.c_str(), (UINT32)discovery_service_full_url.length(), &discovery_url_hstr);
  if (FAILED(hr)) return PrintError(hr);

  std::wstring dummy = L"value";
  HSTRING dummy_hstr = NULL;
  hr = WindowsCreateString(dummy.c_str(), (UINT32)dummy.length(), &dummy_hstr);
  if (FAILED(hr)) return PrintError(hr);

  // Activating ReflectedEnroller WinRT service, we are particulary interested in the ReflectedEnrollment interface
  const HStringReference reflected_enroller_name = HString::MakeReference(L"EnterpriseDeviceManagement.Enrollment.ReflectedEnroller");
  ComPtr<ReflectedEnrollment> reflected_enrollment_ptr = NULL;
  hr = Windows::Foundation::ActivateInstance(reflected_enroller_name.Get(), &reflected_enrollment_ptr);
  if (FAILED(hr) || !reflected_enrollment_ptr) return PrintError(hr);

  // Calling AADEnrollAsync to perform the unauthenticated AAD MDM Enrollment
  IAsyncOperation<ReflectedEnrollmentResult>* reflected_enrollment_result_ptr = NULL;
  hr = reflected_enrollment_ptr->AADEnrollAsync(
    upn_hstr,
    discovery_url_hstr,
    dummy_hstr,
    dummy_hstr,
    0,
    dummy_hstr,
    dummy_hstr,
    dummy_hstr,
    &reflected_enrollment_result_ptr);
  if (FAILED(hr) || !reflected_enrollment_result_ptr) return PrintError(hr);

  // wprintf(L"AADEnrollAsync: AAD Enrollment was successfully called for UPN %s\n", upn.c_str());

  return hr;
}