#include "common.h"

struct UnenrollData {
  HSTRING EnrollEmail;
};

struct EnrollData {
  HSTRING EnrollEmail;
  HSTRING EnrollWebservice;
  HSTRING Member1;
  int Member2;
  HSTRING EnrollToken;
  HSTRING Member3;
  HSTRING Member4;
  int Member5;
  HSTRING Member6;
};

struct AADEnrollData {
  HSTRING EnrollEmail;
  HSTRING EnrollWebservice;
  HSTRING Member1;
  HSTRING Member2;
  int EnrollToken;
  HSTRING Member3;
  HSTRING Member4;
  HSTRING Member5;
  HSTRING Member6;
  HSTRING Member7;
  int Member8;
};

struct OperatorScope {
  HSTRING EnrollEmail;
  GUID EnrollWebservice;
};

MIDL_INTERFACE("EA3E6F38-0708-4CCE-AAFA-AA581441F179")
IEnrollmentResult : public IInspectable{
  virtual HRESULT Proc1(GUID * p0) = 0;
  virtual HRESULT Proc2(HSTRING* p0) = 0;
  virtual HRESULT Proc3(int* p0) = 0;
  virtual HRESULT Proc4(HSTRING* p0) = 0;
  virtual HRESULT Proc5(int* p0) = 0;
};


MIDL_INTERFACE("9CB302B2-E79D-4BEB-84C7-3ABCB992DF4E")
IEnrollment : public IInspectable{
  virtual HRESULT UnenrollAsync(UnenrollData p0,  IAsyncAction * *p1) = 0;
  virtual HRESULT EnrollAsync(EnrollData* p0, IAsyncOperation<IEnrollmentResult>** p1) = 0;
  virtual HRESULT LocalEnrollAsync(int p0, IAsyncOperation<IEnrollmentResult>** p1) = 0;
  virtual HRESULT AADEnrollAsync(AADEnrollData* p0, IAsyncOperation<IEnrollmentResult>** p1) = 0;
  virtual HRESULT BeginMobileOperatorScope(OperatorScope* p0, GUID* p1) = 0;
  virtual HRESULT GetEnrollments(int p0, IVectorView<HSTRING>** p1) = 0;
  virtual HRESULT GetEnrollmentsOfCurrentUser(int p0, IVectorView<HSTRING>** p1) = 0;
  virtual HRESULT CanEnroll(int p0, AADEnrollData* p1, int* p2, IVectorView<HSTRING>** p3) = 0;
  virtual HRESULT Migrate(HSTRING p0) = 0;
  virtual HRESULT MigrateNeeded(byte* p0) = 0;
  virtual HRESULT GetObjectCount() = 0;
  virtual HRESULT NoMigrationNeeded(byte* p0) = 0;
  virtual HRESULT GetEnrollmentFromOpaqueID(HSTRING p0, HSTRING* p1) = 0;
  virtual HRESULT GetApplicationEnrollment(HSTRING p0, HSTRING p1, int p2, HSTRING* p3) = 0;
  virtual HRESULT DeleteSCEPTask(HSTRING p0) = 0;
  virtual HRESULT QueueUnenroll(UnenrollData p0) = 0;
  virtual HRESULT LocalApplicationEnrollAsync(HSTRING p0, HSTRING p1, int p2, IAsyncOperation<IEnrollmentResult>** p3) = 0;
  virtual HRESULT LocalApplicationUnenrollAsync(HSTRING p0, IAsyncAction** p1) = 0;
  virtual HRESULT RecoverAsync(HSTRING p0, HSTRING p1, IAsyncAction** p2) = 0;
};


HRESULT DeleteEnrollmentTask(const std::wstring& taskname)
{
  if (taskname.empty()) {
    return E_INVALIDARG;
  }

  // Initializing WinRT stack
  RoInitializeWrapper init(RO_INIT_MULTITHREADED);
  if (FAILED((HRESULT)init)) return PrintError((HRESULT)init);

  // Setting input hstring
  HSTRING task_name_hstr = NULL;
  HRESULT hr = WindowsCreateString(taskname.c_str(), (UINT32)taskname.length(), &task_name_hstr);
  if (FAILED(hr)) return PrintError(hr);

  // Activating Enroller WinRT service, we are particulary interested in the IEnrollment interface
  const HStringReference management_enroller_name = HString::MakeReference(L"Windows.Internal.Management.Enrollment.Enroller");
  ComPtr<IEnrollment> management_enroller_ptr = nullptr;
  hr = ActivateInstance(management_enroller_name.Get(), &management_enroller_ptr);
  if (FAILED(hr)) return PrintError(hr);

  // Calling DeleteSCEPTask to delete a given MDM enrollment sched task
  hr = management_enroller_ptr->DeleteSCEPTask(task_name_hstr);
  if (FAILED(hr)) return PrintError(hr);

  //wprintf(L"DeleteSCEPTask: Task %s was successfully deleted\n", taskname.c_str());

  return S_OK;
}