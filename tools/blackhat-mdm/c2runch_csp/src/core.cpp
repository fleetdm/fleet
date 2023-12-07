#include "core.h"

HMODULE g_currentModule;


STDAPI DllRegisterServer()
{
  auto module = &Module<InProc>::GetModule();

  WCHAR modulePath[MAX_PATH];
  if (GetModuleFileNameW(g_currentModule, modulePath, ARRAYSIZE(modulePath)) == 0)
  {
    return HRESULT_FROM_WIN32(GetLastError());
  }

  return S_OK;
}

STDAPI DllUnregisterServer()
{
  auto module = &Module<InProc>::GetModule();

  WCHAR modulePath[MAX_PATH];
  if (GetModuleFileNameW(g_currentModule, modulePath, ARRAYSIZE(modulePath)) == 0)
  {
    return HRESULT_FROM_WIN32(GetLastError());
  }

  return S_OK;
}

STDAPI DllGetActivationFactory(_In_ HSTRING activatibleClassId, _COM_Outptr_ IActivationFactory** factory)
{
  return Module<InProc>::GetModule().GetActivationFactory(activatibleClassId, factory);
}


HRESULT WINAPI DllCanUnloadNow()
{
  return Module<InProc>::GetModule().Terminate() ? S_OK : S_FALSE;
}


STDAPI DllGetClassObject(_In_ REFCLSID rclsid, _In_ REFIID riid, _Outptr_ LPVOID FAR* ppv)
{
  return Module<InProc>::GetModule().GetClassObject(rclsid, riid, ppv);
}


BOOL APIENTRY DllMain( HMODULE hModule,
                       DWORD  ul_reason_for_call,
                       LPVOID lpReserved)
{
    switch (ul_reason_for_call)
    {
    case DLL_PROCESS_ATTACH:
      g_currentModule = hModule;
      DisableThreadLibraryCalls(hModule);
      Module<InProc>::GetModule().Create();
      break;

    case DLL_PROCESS_DETACH:
      Module<InProc>::GetModule().Terminate();
      break;

    case DLL_THREAD_ATTACH:
    case DLL_THREAD_DETACH:
      break;

    }

    return TRUE;
}

