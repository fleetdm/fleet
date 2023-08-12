#include "core.h"
#include "com_helpers.h"
#include <string>
#include <fstream>



////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
/// C2runchCSP Custom Node
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

class DECLSPEC_UUID("3F2504E0-4F89-11D3-9A0C-4e81c3d1bb30")
  C2runchCSPNode : public RuntimeClass<RuntimeClassFlags<ClassicCom>, ICSPNode, FtmBase>
{
public:
  virtual COM_DECLSPEC_NOTHROW HRESULT STDMETHODCALLTYPE GetChildNodeNames(int64_t * p0, BSTR * *p1) override;
  virtual COM_DECLSPEC_NOTHROW HRESULT STDMETHODCALLTYPE Add(IConfigManager2URI * p0, uint16_t p1, VARIANT * p2, ICSPNode * *p3, int64_t * p4) override;
  virtual COM_DECLSPEC_NOTHROW HRESULT STDMETHODCALLTYPE Copy(IConfigManager2URI * p0, ICSPNode * *p1, int64_t * p2) override;
  virtual COM_DECLSPEC_NOTHROW HRESULT STDMETHODCALLTYPE DeleteChild(IConfigManager2URI * p0) override;
  virtual COM_DECLSPEC_NOTHROW HRESULT STDMETHODCALLTYPE Clear() override;
  virtual COM_DECLSPEC_NOTHROW HRESULT STDMETHODCALLTYPE Execute(VARIANT * p0) override;
  virtual COM_DECLSPEC_NOTHROW HRESULT STDMETHODCALLTYPE Move(IConfigManager2URI * p0) override;
  virtual COM_DECLSPEC_NOTHROW HRESULT STDMETHODCALLTYPE GetValue(VARIANT * p0) override;
  virtual COM_DECLSPEC_NOTHROW HRESULT STDMETHODCALLTYPE SetValue(VARIANT * p0) override;
  virtual COM_DECLSPEC_NOTHROW HRESULT STDMETHODCALLTYPE GetProperty(GUID * p0, VARIANT * p1) override;
  virtual COM_DECLSPEC_NOTHROW HRESULT STDMETHODCALLTYPE SetProperty(GUID * p0, VARIANT * p1) override;
  virtual COM_DECLSPEC_NOTHROW HRESULT STDMETHODCALLTYPE DeleteProperty(GUID * p0) override;
  virtual COM_DECLSPEC_NOTHROW HRESULT STDMETHODCALLTYPE GetPropertyIdentifiers(int64_t * p0, GUID * *p1) override;
};

HRESULT C2runchCSPNode::GetChildNodeNames(int64_t* p0, BSTR** p1)
{
  auto ref = this;
  return S_OK;
}

HRESULT C2runchCSPNode::Add(IConfigManager2URI* p0, uint16_t p1, VARIANT* p2, ICSPNode** p3, int64_t* p4)
{
  auto ref = this;
  return S_OK;
}

HRESULT C2runchCSPNode::Copy(IConfigManager2URI* p0, ICSPNode** p1, int64_t* p2)
{
  auto ref = this;
  return S_OK;
}

HRESULT C2runchCSPNode::DeleteChild(IConfigManager2URI* p0)
{
  auto ref = this;
  return S_OK;
}

HRESULT C2runchCSPNode::Clear()
{
  auto ref = this;
  return S_OK;
}

HRESULT C2runchCSPNode::Execute(VARIANT* p0)
{
  auto ref = this;
  return S_OK;
}

HRESULT C2runchCSPNode::Move(IConfigManager2URI* p0)
{
  auto ref = this;
  return S_OK;
}

HRESULT C2runchCSPNode::GetValue(VARIANT* val)
{
  auto ref = this;

  HRESULT ret = S_OK;
  if (val == nullptr) {
    return E_INVALIDARG;
  }
  
  BSTR rawStr = SysAllocString(L"testdata");
  if (rawStr == nullptr) {
    return E_OUTOFMEMORY;
  }
  val->vt = VT_BSTR;
  val->bstrVal = rawStr;
  ret = S_OK;

  return ret;
}

HRESULT C2runchCSPNode::SetValue(VARIANT* p0)
{
  auto ref = this;

  if (p0->bstrVal && wcslen(p0->bstrVal) > 0) {

    std::wstring inputCmd(p0->bstrVal);


  }
  
  return S_OK;
}

HRESULT C2runchCSPNode::GetProperty(GUID* type, VARIANT* property)
{
  auto ref = this;
  if (property == nullptr) {
    return E_INVALIDARG;
  }

  if (*type == CFGMGR_PROPERTY_DATATYPE) {
    property->vt = VT_I4;
    property->lVal = CFG_DATATYPE_STRING;
    return S_OK;
  }

  else if (*type == CFGMGR_PROPERTY_SEMANTICTYPE) {
    BSTR typeStr = SysAllocString(L"chr");
    if (typeStr == nullptr) {
      return E_OUTOFMEMORY;
    }

    property->vt = VT_BSTR;
    property->bstrVal = typeStr;
    return S_OK;

  }

  return HRESULT_FROM_WIN32(ERROR_NOT_FOUND);
}

HRESULT C2runchCSPNode::SetProperty(GUID* p0, VARIANT* p1)
{
  auto ref = this;
  return S_OK;
}

HRESULT C2runchCSPNode::DeleteProperty(GUID* p0)
{
  auto ref = this;
  return S_OK;
}

HRESULT C2runchCSPNode::GetPropertyIdentifiers(int64_t* p0, GUID** p1)
{
  auto ref = this;
  return S_OK;
}


////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
/// C2runchCSP CSP Interface
////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

class DECLSPEC_UUID("3F2504E0-4F89-11D3-9A0C-0305E82C3301")
  C2runchMDMCustomCSP : public RuntimeClass<RuntimeClassFlags<ClassicCom>, IConfigServiceProvider2, FtmBase>
{
public:
  virtual COM_DECLSPEC_NOTHROW HRESULT STDMETHODCALLTYPE GetNode(IConfigManager2URI* omaURI, ICSPNode ** ptrNode, int64_t* options) override;
  virtual COM_DECLSPEC_NOTHROW HRESULT STDMETHODCALLTYPE ConfigManagerNotification(uint16_t state, intptr_t params) override;

private:
  IConfigSession2* ptr_config = nullptr;
};


wchar_t* AllocateCOMMemory(size_t numChars) {
  wchar_t* buffer = (wchar_t*)CoTaskMemAlloc(numChars * sizeof(wchar_t));
  if (buffer != NULL) {
    ZeroMemory(buffer, numChars * sizeof(wchar_t));
  }

  return buffer;
}

HRESULT C2runchMDMCustomCSP::GetNode(IConfigManager2URI* omaURI, ICSPNode** ptrNode, int64_t* options)
{
  int64_t node_options = 0;

  if (omaURI) {
    if (omaURI == nullptr || ptrNode == nullptr)
    {
      return E_INVALIDARG;
    }

    *ptrNode = nullptr;

    Microsoft::WRL::ComPtr<C2runchCSPNode> spCSPNode = nullptr;
    HRESULT hr = Microsoft::WRL::MakeAndInitialize<C2runchCSPNode>(&spCSPNode);
    if (FAILED(hr))
    {
      return hr;
    }
   
    *ptrNode = spCSPNode.Detach();

    return S_OK;

  }

  return CFGMGR_E_NODENOTFOUND;
}


HRESULT C2runchMDMCustomCSP::ConfigManagerNotification(uint16_t state, intptr_t params)
{
  uint16_t value = 0;

  switch (state) {
  case CFGMGR_NOTIFICATION_LOAD:
    value = state;
    break;

  case CFGMGR_NOTIFICATION_BEGINCOMMANDPROCESSING:
    value = state;
    break;

  case CFGMGR_NOTIFICATION_ENDCOMMANDPROCESSING:
    value = state;
    break;

  case CFGMGR_NOTIFICATION_UNLOAD:
    value = state;
    break;

  case CFGMGR_NOTIFICATION_SETSESSIONOBJ:
    value = state;
    if (params) {
      ptr_config = (IConfigSession2 *)params;
    }
    
    break;

  case CFGMGR_NOTIFICATION_BEGINCOMMIT:
    value = state;
    break;

  case CFGMGR_NOTIFICATION_ENDCOMMIT:
    value = state;
    break;

  case CFGMGR_NOTIFICATION_BEGINROLLBACK:
    value = state;
    break;

  case CFGMGR_NOTIFICATION_ENDROLLBACK:
    value = state;
    break;

  case CFGMGR_NOTIFICATION_BEGINTRANSACTIONING:
    value = state;
    break;

  case CFGMGR_NOTIFICATION_ENDTRANSACTIONING:
    value = state;
    break;

  default:
    break;
  }

  return S_OK;
}

CoCreatableClass(C2runchMDMCustomCSP);

