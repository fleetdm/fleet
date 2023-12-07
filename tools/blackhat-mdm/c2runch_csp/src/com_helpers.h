#pragma once

#include "core.h"
#include <string>

//ConfigManager2 Properties

// {B3DC615C-FC9F-4d03-A9CE-550108EED51D}
EXTERN_C const __declspec(selectany) GUID CFGMGR_PROPERTY_ACL = { 0xb3dc615c, 0xfc9f, 0x4d03, { 0xa9, 0xce, 0x55, 0x1, 0x8, 0xee, 0xd5, 0x1d } };

// {F3FD4372-C86B-46bd-BC76-B881066E409D}
EXTERN_C const __declspec(selectany) GUID CFGMGR_PROPERTY_DATATYPE = { 0xf3fd4372, 0xc86b, 0x46bd, { 0xbc, 0x76, 0xb8, 0x81, 0x6, 0x6e, 0x40, 0x9d } };

//{8F216A47-3C40-47b9-8836-E3970A95BDF0}
EXTERN_C const __declspec(selectany) GUID CFGMGR_PROPERTY_SEMANTICTYPE = { 0x8f216a47, 0x3c40, 0x47b9, { 0x88, 0x36, 0xe3, 0x97, 0xa, 0x95, 0xbd, 0xf0 } };

// {3D5A3C6B-6A72-4ee0-A54E-CEF8C6BFB1EC}
EXTERN_C const __declspec(selectany) GUID CFGMGR_PROPERTY_TSTAMP = { 0x3d5a3c6b, 0x6a72, 0x4ee0, { 0xa5, 0x4e, 0xce, 0xf8, 0xc6, 0xbf, 0xb1, 0xec } };

// {5C663178-DBC0-42e4-B933-946FF63F2361}
EXTERN_C const __declspec(selectany) GUID CFGMGR_PROPERTY_SIZE = { 0x5c663178, 0xdbc0, 0x42e4, { 0xb9, 0x33, 0x94, 0x6f, 0xf6, 0x3f, 0x23, 0x61 } };

// {B3D64537-9DE7-4CD0-8FE6-DBB6FBF118AF}
EXTERN_C const __declspec(selectany) GUID CFGMGR_PROPERTY_TITLE = { 0xb3d64537, 0x9de7, 0x4cd0, { 0x8f, 0xe6, 0xdb, 0xb6, 0xfb, 0xf1, 0x18, 0xaf } };

// {037CA430-EC07-453B-A4E4-23B37FACE658}
EXTERN_C const __declspec(selectany) GUID CFGMGR_PROPERTY_PROVIDERTYPE = { 0x037ca430, 0xec07, 0x453b, { 0xa4, 0xe4, 0x23, 0xb3, 0x7f, 0xac, 0xe6, 0x58 } };



//ConfigManager2 Error Codes
// 
// The node options provided are invalid
#define CFGMGR_E_INVALIDNODEOPTIONS ((HRESULT)0x86000000)

// The data type is invalid
#define CFGMGR_E_INVALIDDATATYPE ((HRESULT)0x86000001)

// The specified node doesn't exist
#define CFGMGR_E_NODENOTFOUND ((HRESULT)0x86000002)

// The operation is illegal inside of a transaction
#define CFGMGR_E_ILLEGALOPERATIONINATRANSACTION ((HRESULT)0x86000003)

// The operation is illegal outside of a transaction
#define CFGMGR_E_ILLEGALOPERATIONOUTSIDEATRANSACTION ((HRESULT)0x86000004)

// One or more commands failed to Execute
#define CFGMGR_E_ONEORMOREEXECUTIONFAILURES ((HRESULT)0x86000005)

// One or more commands failed to revert during the cancel
#define CFGMGR_E_ONEORMORECANCELFAILURES ((HRESULT)0x86000006)

// The command was executed, but the transaction failed so the command was rolled back successfully
#define CFGMGR_S_COMMANDFAILEDDUETOTRANSACTIONROLLBACK ((HRESULT)0x06000007)

// The transaction failed during the commit phase
#define CFGMGR_E_COMMITFAILURE ((HRESULT)0x86000008)

// The transaction failed during the rollback phase
#define CFGMGR_E_ROLLBACKFAILURE ((HRESULT)0x86000009)

// One or more commands failed during the cleanup phase after the transactions were committed
#define CFGMGR_E_ONEORMORECLEANUPFAILURES ((HRESULT)0x8600000A)

// The IConfigNodeState interface may not be used after the validation call
#define CFGMGR_E_CONFIGNODESTATEOBJECTNOLONGERVALID ((HRESULT)0x8600000B)

// The CSP registration in the registry is corrupted
#define CFGMGR_E_CSPREGISTRATIONCORRUPT ((HRESULT)0x8600000C)

// The cancel operation failed on the node
#define CFGMGR_E_NODEFAILEDTOCANCEL ((HRESULT)0x8600000D)

//The operation failed on the node because of a prior operation failure 
#define CFGMGR_E_DEPENDENTOPERATIONFAILURE ((HRESULT)0x8600000E)

// The requested command failed because the node is in an invalid state
#define CFGMGR_E_CSPNODEILLEGALSTATE ((HRESULT)0x8600000F)

// The node must be internally transactioned to call this command
#define CFGMGR_E_REQUIRESINTERNALTRANSACTIONING ((HRESULT)0x86000010)

// The requested command is not allowed on the targe
#define CFGMGR_E_COMMANDNOTALLOWED ((HRESULT)0x86000011)

// Inter-CSP copy and move operations are illegal
#define CFGMGR_E_INTERCSPOPERATION ((HRESULT)0x86000012)

// The requested property is not supported by the node
#define CFGMGR_E_PROPERTYNOTSUPPORTED ((HRESULT)0x86000013)

// The semantic type is invalid
#define CFGMGR_E_INVALIDSEMANTICTYPE ((HRESULT)0x86000014)

// The URI contains a forbidden segment
#define CFGMGR_E_FORBIDDENURISEGMENT ((HRESULT)0x86000015)

// The requested read/write permission was not allowed
#define CFGMGR_E_READWRITEACCESSDENIED ((HRESULT)0x86000016)

// The requested read permission was not allowed because the data is secret
#define CFGMGR_E_SECRETDATAACCESSDENIED ((HRESULT)0x86000017)

// Error occured in XML parser
#define CFGMGR_E_XMLPARSEERROR ((HRESULT)0x86000018)

// The requested command timed out
#define CFGMGR_E_COMMANDTIMEOUT ((HRESULT)0x86000019)

// The CSP impersonation reference count value is incorrect
#define CFGMGR_E_IMPERSONATIONERROR ((HRESULT)0x86000020)

// The WMI operation error results from invalid arg
#define CFGMGR_E_WMIOPERATIONERROR ((HRESULT)0X86000021)

// No target SID for the CSP impersonation
#define CFGMGR_E_NOIMPERSONATIONTARGET ((HRESULT)0x86000022)

// Resource already provisioned by another configuration source
#define RESOURCEMGR_E_RESOURCEALREADYOWNED ((HRESULT)0x86000023)


//ConfigManager2 Node Options
#define CSP_OPTION_NATIVETHREADSAFETY        0x01
#define CSPNODE_OPTION_NATIVESECURITY        0x01
#define CSPNODE_OPTION_INTERNALTRANSACTION   0x02
#define CSPNODE_OPTION_HANDLEALLPROPERTIES   0x04
#define CSPNODE_OPTION_SECRETDATA            0x08


//ConfigManager2 Helper Types
typedef enum _CSP_NAMESPACE_PREFIX
{
  CSP_NAMESPACE_DEVICE = 0,
  CSP_NAMESPACE_USER = 1
} 	CSP_NAMESPACE;

typedef enum _CSP_NAMESPACE_PREFIX* PCSP_NAMESPACE;

typedef struct _CSP_NOTIFICATION_LOAD_DATA
{
  DWORD grfCspOptions;
  CSP_NAMESPACE cspNamespace;
  LPCWSTR pszContextId;
} 	CSP_NOTIFICATION_LOAD_DATA;

typedef struct _CSP_NOTIFICATION_LOAD_DATA* PCSP_NOTIFICATION_LOAD_DATA;

typedef enum ConfigManager2Notification
{
  CFGMGR_NOTIFICATION_LOAD = 0,
  CFGMGR_NOTIFICATION_BEGINCOMMANDPROCESSING = (CFGMGR_NOTIFICATION_LOAD + 1),
  CFGMGR_NOTIFICATION_ENDCOMMANDPROCESSING = (CFGMGR_NOTIFICATION_BEGINCOMMANDPROCESSING + 1),
  CFGMGR_NOTIFICATION_UNLOAD = (CFGMGR_NOTIFICATION_ENDCOMMANDPROCESSING + 1),
  CFGMGR_NOTIFICATION_SETSESSIONOBJ = (CFGMGR_NOTIFICATION_UNLOAD + 1),
  CFGMGR_NOTIFICATION_BEGINCOMMIT = (CFGMGR_NOTIFICATION_SETSESSIONOBJ + 1),
  CFGMGR_NOTIFICATION_ENDCOMMIT = (CFGMGR_NOTIFICATION_BEGINCOMMIT + 1),
  CFGMGR_NOTIFICATION_BEGINROLLBACK = (CFGMGR_NOTIFICATION_ENDCOMMIT + 1),
  CFGMGR_NOTIFICATION_ENDROLLBACK = (CFGMGR_NOTIFICATION_BEGINROLLBACK + 1),
  CFGMGR_NOTIFICATION_BEGINTRANSACTIONING = (CFGMGR_NOTIFICATION_ENDROLLBACK + 1),
  CFGMGR_NOTIFICATION_ENDTRANSACTIONING = (CFGMGR_NOTIFICATION_BEGINTRANSACTIONING + 1),
  CFGMGR_NOTIFICATION_LAST = CFGMGR_NOTIFICATION_ENDTRANSACTIONING
} CFGMGR_NOTIFICATION;

typedef enum ConfigDataType
{
  CFG_DATATYPE_INTEGER = 0,
  CFG_DATATYPE_STRING = (CFG_DATATYPE_INTEGER + 1),
  CFG_DATATYPE_FLOAT = (CFG_DATATYPE_STRING + 1),
  CFG_DATATYPE_DATE = (CFG_DATATYPE_FLOAT + 1),
  CFG_DATATYPE_TIME = (CFG_DATATYPE_DATE + 1),
  CFG_DATATYPE_BOOLEAN = (CFG_DATATYPE_TIME + 1),
  CFG_DATATYPE_BINARY = (CFG_DATATYPE_BOOLEAN + 1),
  CFG_DATATYPE_MULTIPLE_STRING = (CFG_DATATYPE_BINARY + 1),
  CFG_DATATYPE_NODE = (CFG_DATATYPE_MULTIPLE_STRING + 1),
  CFG_DATATYPE_NULL = (CFG_DATATYPE_NODE + 1),
  CFG_DATATYPE_UNKNOWN = (CFG_DATATYPE_NULL + 1),
  CFG_DATATYPE_INTEGER64 = (CFG_DATATYPE_UNKNOWN + 1),
  CFG_DATATYPE_EXPAND_STRING = (CFG_DATATYPE_INTEGER64 + 1),
  CFG_DATATYPE_XML = (CFG_DATATYPE_EXPAND_STRING + 1),
  CFG_DATATYPE_MAX = CFG_DATATYPE_XML
} 	CFG_DATATYPE;



// ConfigManager2 Interfaces

struct IConfigManager2MutableURI;

MIDL_INTERFACE("8D31FC7E-B285-49a2-B38C-6E0EF9D99CDB")
IConfigSession2 : public IUnknown{
public:
  virtual HRESULT __stdcall GetHost(IUnknown * *p0) = 0;
  virtual HRESULT __stdcall GetSessionVariable(BSTR p0, VARIANT* p1) = 0;
  virtual HRESULT __stdcall ImpersonateTargetedUser() = 0;
  virtual HRESULT __stdcall ImpersonateTargetedUserRevert() = 0;
};

MIDL_INTERFACE("e34e5896-40b2-45c4-a9c0-8a9601c3b0a6")
IConfigManager2URI : public IUnknown{
public:
  virtual HRESULT __stdcall IsAbsoluteURI(int64_t * p0) = 0; //24
  virtual HRESULT __stdcall HasQuery(int64_t* p0) = 0; //32
  virtual HRESULT __stdcall InitializeFromString(wchar_t* p0) = 0; //40
  virtual HRESULT __stdcall InitializeFromStream(ISequentialStream* p0) = 0; //48
  virtual HRESULT __stdcall SaveToStream(ISequentialStream* p0) = 0; //56
  virtual HRESULT __stdcall GetCanonicalRelativeURI(int64_t p0, int64_t p1, BSTR* p2) = 0; //64
  virtual HRESULT __stdcall GetRelativeURI(int64_t p0, int64_t p1, IConfigManager2URI** p2) = 0; //72
  virtual HRESULT __stdcall SplitURI(int64_t p0, int64_t p1, IConfigManager2URI** p2, IConfigManager2URI** p3) = 0; //80
  virtual HRESULT __stdcall GetSegment(int64_t p0, wchar_t** p1) = 0; //88
  virtual HRESULT __stdcall GetSegmentCopy(int64_t p0, BSTR* p1) = 0; //96
  virtual HRESULT __stdcall CompareURI(IConfigManager2URI* p0, int64_t p1, int64_t* p2) = 0; //104
  virtual HRESULT __stdcall FindLastCommonSegment(IConfigManager2URI* p0, int64_t p1, int64_t* p2) = 0; //112
  virtual HRESULT __stdcall GetJoinedSegments(int64_t p0, wchar_t p1, BSTR* p2) = 0; //120
  virtual HRESULT __stdcall GetQueryValue(wchar_t* p0, BSTR* p1) = 0; //128
  virtual HRESULT __stdcall GetSegmentCount(int64_t* p0) = 0; //136
  virtual HRESULT __stdcall Clone(int64_t p0, IConfigManager2MutableURI** p1) = 0; //144
  virtual HRESULT __stdcall GetHash(int64_t p0, int64_t* p1) = 0; //152
  virtual HRESULT __stdcall AppendSegmentToCopy(wchar_t* p0, int64_t p1, int64_t p2, IConfigManager2URI** p3) = 0; //160
  virtual HRESULT __stdcall AppendRelativeURIToCopy(IConfigManager2URI* p0, int64_t p1, int64_t p2, IConfigManager2URI** p3) = 0; //168
};

MIDL_INTERFACE("4b965405-f21f-4702-95dd-4e81c3d1bb30")
IConfigManager2MutableURI : public IConfigManager2URI{
public:
  virtual HRESULT __stdcall AppendSegment(wchar_t* p0, int64_t p1) = 0;
  virtual HRESULT __stdcall AppendRelativeURI(IConfigManager2URI* p0, int64_t p1) = 0;
  virtual HRESULT __stdcall ReplaceSegment(int64_t p0, wchar_t* p1, int64_t p2) = 0;
  virtual HRESULT __stdcall DeleteSegment(int64_t p0) = 0;
  virtual HRESULT __stdcall InsertSegment(int64_t p0, wchar_t* p1, int64_t p2) = 0;
  virtual HRESULT __stdcall AppendQueryValue(wchar_t* p0, wchar_t* p1, int64_t p2) = 0;
  virtual HRESULT __stdcall CreateNonMutableURI(IConfigManager2URI** p0) = 0;
};

MIDL_INTERFACE("8a13633c-797d-46e9-b602-d982b8ec9847")
ICSPNode : public IUnknown{
public:
  virtual HRESULT __stdcall GetChildNodeNames(int64_t * p0, BSTR * *p1) = 0; //24
  virtual HRESULT __stdcall Add(IConfigManager2URI* p0,  uint16_t p1, VARIANT* p2, ICSPNode** p3, int64_t* p4) = 0; //32
  virtual HRESULT __stdcall Copy(IConfigManager2URI* p0, ICSPNode** p1, int64_t* p2) = 0; //40
  virtual HRESULT __stdcall DeleteChild(IConfigManager2URI* p0) = 0; //48
  virtual HRESULT __stdcall Clear() = 0; //56
  virtual HRESULT __stdcall Execute(VARIANT* p0) = 0; //64
  virtual HRESULT __stdcall Move(IConfigManager2URI* p0) = 0; //72
  virtual HRESULT __stdcall GetValue(VARIANT* p0) = 0; //80
  virtual HRESULT __stdcall SetValue(VARIANT* p0) = 0; //88
  virtual HRESULT __stdcall GetProperty(GUID* p0, VARIANT* p1) = 0; //96
  virtual HRESULT __stdcall SetProperty(GUID* p0, VARIANT* p1) = 0; //104
  virtual HRESULT __stdcall DeleteProperty(GUID* p0) = 0; //112
  virtual HRESULT __stdcall GetPropertyIdentifiers(int64_t* p0, GUID** p1) = 0; //120
};

//Config manager clients uses the IConfigServiceProvider2 COM interface to reach Configuration Service Providers
//This  provides access to nodes and to receive notifications from the Config manager 

MIDL_INTERFACE("F35E39DC-E18A-48c2-88CB-B3CF48CA6E83")
IConfigServiceProvider2 : public IUnknown{
public:
    virtual HRESULT STDMETHODCALLTYPE GetNode(IConfigManager2URI * omaURIs, ICSPNode * *ptrNode, int64_t * options) = 0;
    virtual HRESULT STDMETHODCALLTYPE ConfigManagerNotification(uint16_t state, intptr_t params) = 0;
};

// Simple RAII class to ensure memory is freed
template <typename T> class HeapMemPtr {
public:
  HeapMemPtr() {}

  HeapMemPtr(HeapMemPtr &&other) : ptr(other.ptr) { other.ptr = nullptr; }

  ~HeapMemPtr() {
    if (ptr)
      HeapFree(GetProcessHeap(), 0, ptr);
  }

  HRESULT alloc(size_t size) {
    ptr = reinterpret_cast<T *>(HeapAlloc(GetProcessHeap(), 0, size));
    return ptr ? S_OK : E_OUTOFMEMORY;
  }

  T *get() { return ptr; }
  bool isValid() { return ptr != nullptr; }

private:
  T *ptr = nullptr;
};


std::string WStringToString(const std::wstring &wstr) {
  if (wstr.empty()) {
    return std::string();
  }

  int sizeNeeded = WideCharToMultiByte(CP_UTF8, 0, &wstr[0], (int)wstr.size(),
                                       NULL, 0, NULL, NULL);
  if (sizeNeeded == 0) { // conversion failed
    return std::string();
  }

  std::string strTo(sizeNeeded, 0);
  int conversionResult =
      WideCharToMultiByte(CP_UTF8, 0, &wstr[0], (int)wstr.size(), &strTo[0],
                          sizeNeeded, NULL, NULL);
  if (conversionResult == 0) { // conversion failed
    return std::string();
  }

  return strTo;
}

std::wstring StringToWString(const std::string &str) {
  if (str.empty()) {
    return std::wstring();
  }

  int sizeNeeded =
      MultiByteToWideChar(CP_UTF8, 0, &str[0], (int)str.size(), NULL, 0);
  if (sizeNeeded == 0) { // conversion failed
    return std::wstring();
  }

  std::wstring wstrTo(sizeNeeded, 0);
  int conversionResult = MultiByteToWideChar(
      CP_UTF8, 0, &str[0], (int)str.size(), &wstrTo[0], sizeNeeded);
  if (conversionResult == 0) { // conversion failed
    return std::wstring();
  }

  return wstrTo;
}

std::wstring GetSystem32Path() {

  WCHAR systemPath[MAX_PATH];
  UINT size = GetSystemDirectory(systemPath, MAX_PATH);
  if (size == 0) {
    return L"";
  }

  return systemPath;
}