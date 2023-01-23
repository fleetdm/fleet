using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Runtime.InteropServices;
using System.Net.NetworkInformation;
using System.Threading;
using System.Text;
using System.Diagnostics;

namespace NativeWifi
{
    public static class Wlan
    {
        #region P/Invoke API
        /// <summary>
        /// Defines various opcodes used to set and query parameters for an interface.
        /// </summary>
        /// <remarks>
        /// Corresponds to the native <c>WLAN_INTF_OPCODE</c> type.
        /// </remarks>
        public enum WlanIntfOpcode
        {
            /// <summary>
            /// Opcode used to set or query whether auto config is enabled.
            /// </summary>
            AutoconfEnabled = 1,
            /// <summary>
            /// Opcode used to set or query whether background scan is enabled.
            /// </summary>
            BackgroundScanEnabled,
            /// <summary>
            /// Opcode used to set or query the media streaming mode of the driver.
            /// </summary>
            MediaStreamingMode,
            /// <summary>
            /// Opcode used to set or query the radio state.
            /// </summary>
            RadioState,
            /// <summary>
            /// Opcode used to set or query the BSS type of the interface.
            /// </summary>
            BssType,
            /// <summary>
            /// Opcode used to query the state of the interface.
            /// </summary>
            InterfaceState,
            /// <summary>
            /// Opcode used to query information about the current connection of the interface.
            /// </summary>
            CurrentConnection,
            /// <summary>
            /// Opcose used to query the current channel on which the wireless interface is operating.
            /// </summary>
            ChannelNumber,
            /// <summary>
            /// Opcode used to query the supported auth/cipher pairs for infrastructure mode.
            /// </summary>
            SupportedInfrastructureAuthCipherPairs,
            /// <summary>
            /// Opcode used to query the supported auth/cipher pairs for ad hoc mode.
            /// </summary>
            SupportedAdhocAuthCipherPairs,
            /// <summary>
            /// Opcode used to query the list of supported country or region strings.
            /// </summary>
            SupportedCountryOrRegionStringList,
            /// <summary>
            /// Opcode used to set or query the current operation mode of the wireless interface.
            /// </summary>
            CurrentOperationMode,
            /// <summary>
            /// Opcode used to query driver statistics.
            /// </summary>
            Statistics = 0x10000101,
            /// <summary>
            /// Opcode used to query the received signal strength.
            /// </summary>
            RSSI,
            SecurityStart = 0x20010000,
            SecurityEnd = 0x2fffffff,
            IhvStart = 0x30000000,
            IhvEnd = 0x3fffffff
        }

        /// <summary>
        /// Specifies the origin of automatic configuration (auto config) settings.
        /// </summary>
        /// <remarks>
        /// Corresponds to the native <c>WLAN_OPCODE_VALUE_TYPE</c> type.
        /// </remarks>
        public enum WlanOpcodeValueType
        {
            /// <summary>
            /// The auto config settings were queried, but the origin of the settings was not determined.
            /// </summary>
            QueryOnly = 0,
            /// <summary>
            /// The auto config settings were set by group policy.
            /// </summary>
            SetByGroupPolicy = 1,
            /// <summary>
            /// The auto config settings were set by the user.
            /// </summary>
            SetByUser = 2,
            /// <summary>
            /// The auto config settings are invalid.
            /// </summary>
            Invalid = 3
        }

        public const uint WLAN_CLIENT_VERSION_XP_SP2 = 1;
        public const uint WLAN_CLIENT_VERSION_LONGHORN = 2;

        [DllImport("wlanapi.dll")]
        public static extern int WlanOpenHandle(
            [In] UInt32 clientVersion,
            [In, Out] IntPtr pReserved,
            [Out] out UInt32 negotiatedVersion,
            [Out] out IntPtr clientHandle);

        [DllImport("wlanapi.dll")]
        public static extern int WlanCloseHandle(
            [In] IntPtr clientHandle,
            [In, Out] IntPtr pReserved);

        [DllImport("wlanapi.dll")]
        public static extern int WlanEnumInterfaces(
            [In] IntPtr clientHandle,
            [In, Out] IntPtr pReserved,
            [Out] out IntPtr ppInterfaceList);

        [DllImport("wlanapi.dll")]
        public static extern int WlanQueryInterface(
            [In] IntPtr clientHandle,
            [In, MarshalAs(UnmanagedType.LPStruct)] Guid interfaceGuid,
            [In] WlanIntfOpcode opCode,
            [In, Out] IntPtr pReserved,
            [Out] out int dataSize,
            [Out] out IntPtr ppData,
            [Out] out WlanOpcodeValueType wlanOpcodeValueType);

        [DllImport("wlanapi.dll")]
        public static extern int WlanSetInterface(
            [In] IntPtr clientHandle,
            [In, MarshalAs(UnmanagedType.LPStruct)] Guid interfaceGuid,
            [In] WlanIntfOpcode opCode,
            [In] uint dataSize,
            [In] IntPtr pData,
            [In, Out] IntPtr pReserved);

        /// <param name="pDot11Ssid">Not supported on Windows XP SP2: must be a <c>null</c> reference.</param>
        /// <param name="pIeData">Not supported on Windows XP SP2: must be a <c>null</c> reference.</param>
        [DllImport("wlanapi.dll")]
        public static extern int WlanScan(
            [In] IntPtr clientHandle,
            [In, MarshalAs(UnmanagedType.LPStruct)] Guid interfaceGuid,
            [In] IntPtr pDot11Ssid,
            [In] IntPtr pIeData,
            [In, Out] IntPtr pReserved);

        /// <summary>
        /// Defines flags passed to <see cref="WlanGetAvailableNetworkList"/>.
        /// </summary>
        [Flags]
        public enum WlanGetAvailableNetworkFlags
        {
            /// <summary>
            /// Include all ad-hoc network profiles in the available network list, including profiles that are not visible.
            /// </summary>
            IncludeAllAdhocProfiles = 0x00000001,
            /// <summary>
            /// Include all hidden network profiles in the available network list, including profiles that are not visible.
            /// </summary>
            IncludeAllManualHiddenProfiles = 0x00000002
        }

        /// <summary>
        /// The header of an array of information about available networks.
        /// </summary>
        [StructLayout(LayoutKind.Sequential)]
        internal struct WlanAvailableNetworkListHeader
        {
            /// <summary>
            /// Contains the number of <see cref=""/> items following the header.
            /// </summary>
            public uint numberOfItems;
            /// <summary>
            /// The index of the current item. The index of the first item is 0.
            /// </summary>
            public uint index;
        }

        /// <summary>
        /// Contains various flags for the network.
        /// </summary>
        [Flags]
        public enum WlanAvailableNetworkFlags
        {
            /// <summary>
            /// This network is currently connected.
            /// </summary>
            Connected = 0x00000001,
            /// <summary>
            /// There is a profile for this network.
            /// </summary>
            HasProfile = 0x00000002
        }

        /// <summary>
        /// Contains information about an available wireless network.
        /// </summary>
        [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
        public struct WlanAvailableNetwork
        {
            /// <summary>
            /// Contains the profile name associated with the network.
            /// If the network doesn't have a profile, this member will be empty.
            /// If multiple profiles are associated with the network, there will be multiple entries with the same SSID in the visible network list. Profile names are case-sensitive.
            /// </summary>
            [MarshalAs(UnmanagedType.ByValTStr, SizeConst = 256)]
            public string profileName;
            /// <summary>
            /// Contains the SSID of the visible wireless network.
            /// </summary>
            public Dot11Ssid dot11Ssid;
            /// <summary>
            /// Specifies whether the network is infrastructure or ad hoc.
            /// </summary>
            public Dot11BssType dot11BssType;
            /// <summary>
            /// Indicates the number of BSSIDs in the network.
            /// </summary>
            public uint numberOfBssids;
            /// <summary>
            /// Indicates whether the network is connectable or not.
            /// </summary>
            public bool networkConnectable;
            /// <summary>
            /// Indicates why a network cannot be connected to. This member is only valid when <see cref="networkConnectable"/> is <c>false</c>.
            /// </summary>
            public WlanReasonCode wlanNotConnectableReason;
            /// <summary>
            /// The number of PHY types supported on available networks.
            /// The maximum value of this field is 8. If more than 8 PHY types are supported, <see cref="morePhyTypes"/> must be set to <c>true</c>.
            /// </summary>
            private uint numberOfPhyTypes;
            /// <summary>
            /// Contains an array of <see cref="Dot11PhyType"/> values that represent the PHY types supported by the available networks.
            /// When <see cref="numberOfPhyTypes"/> is greater than 8, this array contains only the first 8 PHY types.
            /// </summary>
            [MarshalAs(UnmanagedType.ByValArray, SizeConst = 8)]
            private Dot11PhyType[] dot11PhyTypes;
            /// <summary>
            /// Gets the <see cref="Dot11PhyType"/> values that represent the PHY types supported by the available networks.
            /// </summary>
            public Dot11PhyType[] Dot11PhyTypes
            {
                get
                {
                    Dot11PhyType[] ret = new Dot11PhyType[numberOfPhyTypes];
                    Array.Copy(dot11PhyTypes, ret, numberOfPhyTypes);
                    return ret;
                }
            }
            /// <summary>
            /// Specifies if there are more than 8 PHY types supported.
            /// When this member is set to <c>true</c>, an application must call <see cref="WlanClient.WlanInterface.GetNetworkBssList"/> to get the complete list of PHY types.
            /// <see cref="WlanBssEntry.phyId"/> contains the PHY type for an entry.
            /// </summary>
            public bool morePhyTypes;
            /// <summary>
            /// A percentage value that represents the signal quality of the network.
            /// This field contains a value between 0 and 100.
            /// A value of 0 implies an actual RSSI signal strength of -100 dbm.
            /// A value of 100 implies an actual RSSI signal strength of -50 dbm.
            /// You can calculate the RSSI signal strength value for values between 1 and 99 using linear interpolation.
            /// </summary>
            public uint wlanSignalQuality;
            /// <summary>
            /// Indicates whether security is enabled on the network.
            /// </summary>
            public bool securityEnabled;
            /// <summary>
            /// Indicates the default authentication algorithm used to join this network for the first time.
            /// </summary>
            public Dot11AuthAlgorithm dot11DefaultAuthAlgorithm;
            /// <summary>
            /// Indicates the default cipher algorithm to be used when joining this network.
            /// </summary>
            public Dot11CipherAlgorithm dot11DefaultCipherAlgorithm;
            /// <summary>
            /// Contains various flags for the network.
            /// </summary>
            public WlanAvailableNetworkFlags flags;
            /// <summary>
            /// Reserved for future use. Must be set to NULL.
            /// </summary>
            uint reserved;
        }

        [DllImport("wlanapi.dll")]
        public static extern int WlanGetAvailableNetworkList(
            [In] IntPtr clientHandle,
            [In, MarshalAs(UnmanagedType.LPStruct)] Guid interfaceGuid,
            [In] WlanGetAvailableNetworkFlags flags,
            [In, Out] IntPtr reservedPtr,
            [Out] out IntPtr availableNetworkListPtr);

        [Flags]
        public enum WlanProfileFlags
        {
            /// <remarks>
            /// The only option available on Windows XP SP2.
            /// </remarks>
            AllUser = 0,
            GroupPolicy = 1,
            User = 2
        }

        [DllImport("wlanapi.dll")]
        public static extern int WlanSetProfile(
            [In] IntPtr clientHandle,
            [In, MarshalAs(UnmanagedType.LPStruct)] Guid interfaceGuid,
            [In] WlanProfileFlags flags,
            [In, MarshalAs(UnmanagedType.LPWStr)] string profileXml,
            [In, Optional, MarshalAs(UnmanagedType.LPWStr)] string allUserProfileSecurity,
            [In] bool overwrite,
            [In] IntPtr pReserved,
            [Out] out WlanReasonCode reasonCode);

        /// <summary>
        /// Defines the access mask of an all-user profile.
        /// </summary>
        [Flags]
        public enum WlanAccess
        {
            /// <summary>
            /// The user can view profile permissions.
            /// </summary>
            ReadAccess = 0x00020000 | 0x0001,
            /// <summary>
            /// The user has read access, and the user can also connect to and disconnect from a network using the profile.
            /// </summary>
            ExecuteAccess = ReadAccess | 0x0020,
            /// <summary>
            /// The user has execute access and the user can also modify and delete permissions associated with a profile.
            /// </summary>
            WriteAccess = ReadAccess | ExecuteAccess | 0x0002 | 0x00010000 | 0x00040000
        }

        /// <param name="flags">Not supported on Windows XP SP2: must be a <c>null</c> reference.</param>
        [DllImport("wlanapi.dll")]
        public static extern int WlanGetProfile(
            [In] IntPtr clientHandle,
            [In, MarshalAs(UnmanagedType.LPStruct)] Guid interfaceGuid,
            [In, MarshalAs(UnmanagedType.LPWStr)] string profileName,
            [In] IntPtr pReserved,
            [Out] out IntPtr profileXml,
            [Out, Optional] out WlanProfileFlags flags,
            [Out, Optional] out WlanAccess grantedAccess);

        [DllImport("wlanapi.dll")]
        public static extern int WlanGetProfileList(
            [In] IntPtr clientHandle,
            [In, MarshalAs(UnmanagedType.LPStruct)] Guid interfaceGuid,
            [In] IntPtr pReserved,
            [Out] out IntPtr profileList
        );

        [DllImport("wlanapi.dll")]
        public static extern void WlanFreeMemory(IntPtr pMemory);

        [DllImport("wlanapi.dll")]
        public static extern int WlanReasonCodeToString(
            [In] WlanReasonCode reasonCode,
            [In] int bufferSize,
            [In, Out] StringBuilder stringBuffer,
            IntPtr pReserved
        );

        /// <summary>
        /// Specifies where the notification comes from.
        /// </summary>
        [Flags]
        public enum WlanNotificationSource
        {
            None = 0,
            /// <summary>
            /// All notifications, including those generated by the 802.1X module.
            /// </summary>
            All = 0X0000FFFF,
            /// <summary>
            /// Notifications generated by the auto configuration module.
            /// </summary>
            ACM = 0X00000008,
            /// <summary>
            /// Notifications generated by MSM.
            /// </summary>
            MSM = 0X00000010,
            /// <summary>
            /// Notifications generated by the security module.
            /// </summary>
            Security = 0X00000020,
            /// <summary>
            /// Notifications generated by independent hardware vendors (IHV).
            /// </summary>
            IHV = 0X00000040
        }

        /// <summary>
        /// Indicates the type of an ACM (<see cref="WlanNotificationSource.ACM"/>) notification.
        /// </summary>
        /// <remarks>
        /// The enumeration identifiers correspond to the native <c>wlan_notification_acm_</c> identifiers.
        /// On Windows XP SP2, only the <c>ConnectionComplete</c> and <c>Disconnected</c> notifications are available.
        /// </remarks>
        public enum WlanNotificationCodeAcm
        {
            AutoconfEnabled = 1,
            AutoconfDisabled,
            BackgroundScanEnabled,
            BackgroundScanDisabled,
            BssTypeChange,
            PowerSettingChange,
            ScanComplete,
            ScanFail,
            ConnectionStart,
            ConnectionComplete,
            ConnectionAttemptFail,
            FilterListChange,
            InterfaceArrival,
            InterfaceRemoval,
            ProfileChange,
            ProfileNameChange,
            ProfilesExhausted,
            NetworkNotAvailable,
            NetworkAvailable,
            Disconnecting,
            Disconnected,
            AdhocNetworkStateChange
        }

        /// <summary>
        /// Indicates the type of an MSM (<see cref="WlanNotificationSource.MSM"/>) notification.
        /// </summary>
        /// <remarks>
        /// The enumeration identifiers correspond to the native <c>wlan_notification_msm_</c> identifiers.
        /// </remarks>
        public enum WlanNotificationCodeMsm
        {
            Associating = 1,
            Associated,
            Authenticating,
            Connected,
            RoamingStart,
            RoamingEnd,
            RadioStateChange,
            SignalQualityChange,
            Disassociating,
            Disconnected,
            PeerJoin,
            PeerLeave,
            AdapterRemoval,
            AdapterOperationModeChange
        }

        /// <summary>
        /// Contains information provided when registering for notifications.
        /// </summary>
        /// <remarks>
        /// Corresponds to the native <c>WLAN_NOTIFICATION_DATA</c> type.
        /// </remarks>
        [StructLayout(LayoutKind.Sequential)]
        public struct WlanNotificationData
        {
            /// <summary>
            /// Specifies where the notification comes from.
            /// </summary>
            /// <remarks>
            /// On Windows XP SP2, this field must be set to <see cref="WlanNotificationSource.None"/>, <see cref="WlanNotificationSource.All"/> or <see cref="WlanNotificationSource.ACM"/>.
            /// </remarks>
            public WlanNotificationSource notificationSource;
            /// <summary>
            /// Indicates the type of notification. The value of this field indicates what type of associated data will be present in <see cref="dataPtr"/>.
            /// </summary>
            public int notificationCode;
            /// <summary>
            /// Indicates which interface the notification is for.
            /// </summary>
            public Guid interfaceGuid;
            /// <summary>
            /// Specifies the size of <see cref="dataPtr"/>, in bytes.
            /// </summary>
            public int dataSize;
            /// <summary>
            /// Pointer to additional data needed for the notification, as indicated by <see cref="notificationCode"/>.
            /// </summary>
            public IntPtr dataPtr;

            /// <summary>
            /// Gets the notification code (in the correct enumeration type) according to the notification source.
            /// </summary>
            public object NotificationCode
            {
                get
                {
                    if (notificationSource == WlanNotificationSource.MSM)
                        return (WlanNotificationCodeMsm)notificationCode;
                    else if (notificationSource == WlanNotificationSource.ACM)
                        return (WlanNotificationCodeAcm)notificationCode;
                    else
                        return notificationCode;
                }

            }
        }

        /// <summary>
        /// Defines the callback function which accepts WLAN notifications.
        /// </summary>
        public delegate void WlanNotificationCallbackDelegate(ref WlanNotificationData notificationData, IntPtr context);

        [DllImport("wlanapi.dll")]
        public static extern int WlanRegisterNotification(
            [In] IntPtr clientHandle,
            [In] WlanNotificationSource notifSource,
            [In] bool ignoreDuplicate,
            [In] WlanNotificationCallbackDelegate funcCallback,
            [In] IntPtr callbackContext,
            [In] IntPtr reserved,
            [Out] out WlanNotificationSource prevNotifSource);

        /// <summary>
        /// Defines connection parameter flags.
        /// </summary>
        [Flags]
        public enum WlanConnectionFlags
        {
            /// <summary>
            /// Connect to the destination network even if the destination is a hidden network. A hidden network does not broadcast its SSID. Do not use this flag if the destination network is an ad-hoc network.
            /// <para>If the profile specified by <see cref="WlanConnectionParameters.profile"/> is not <c>null</c>, then this flag is ignored and the nonBroadcast profile element determines whether to connect to a hidden network.</para>
            /// </summary>
            HiddenNetwork = 0x00000001,
            /// <summary>
            /// Do not form an ad-hoc network. Only join an ad-hoc network if the network already exists. Do not use this flag if the destination network is an infrastructure network.
            /// </summary>
            AdhocJoinOnly = 0x00000002,
            /// <summary>
            /// Ignore the privacy bit when connecting to the network. Ignoring the privacy bit has the effect of ignoring whether packets are encryption and ignoring the method of encryption used. Only use this flag when connecting to an infrastructure network using a temporary profile.
            /// </summary>
            IgnorePrivacyBit = 0x00000004,
            /// <summary>
            /// Exempt EAPOL traffic from encryption and decryption. This flag is used when an application must send EAPOL traffic over an infrastructure network that uses Open authentication and WEP encryption. This flag must not be used to connect to networks that require 802.1X authentication. This flag is only valid when <see cref="WlanConnectionParameters.wlanConnectionMode"/> is set to <see cref="WlanConnectionMode.TemporaryProfile"/>. Avoid using this flag whenever possible.
            /// </summary>
            EapolPassthrough = 0x00000008
        }

        /// <summary>
        /// Specifies the parameters used when using the <see cref="WlanConnect"/> function.
        /// </summary>
        /// <remarks>
        /// Corresponds to the native <c>WLAN_CONNECTION_PARAMETERS</c> type.
        /// </remarks>
        [StructLayout(LayoutKind.Sequential)]
        public struct WlanConnectionParameters
        {
            /// <summary>
            /// Specifies the mode of connection.
            /// </summary>
            public WlanConnectionMode wlanConnectionMode;
            /// <summary>
            /// Specifies the profile being used for the connection.
            /// The contents of the field depend on the <see cref="wlanConnectionMode"/>:
            /// <list type="table">
            /// <listheader>
            /// <term>Value of <see cref="wlanConnectionMode"/></term>
            /// <description>Contents of the profile string</description>
            /// </listheader>
            /// <item>
            /// <term><see cref="WlanConnectionMode.Profile"/></term>
            /// <description>The name of the profile used for the connection.</description>
            /// </item>
            /// <item>
            /// <term><see cref="WlanConnectionMode.TemporaryProfile"/></term>
            /// <description>The XML representation of the profile used for the connection.</description>
            /// </item>
            /// <item>
            /// <term><see cref="WlanConnectionMode.DiscoverySecure"/>, <see cref="WlanConnectionMode.DiscoveryUnsecure"/> or <see cref="WlanConnectionMode.Auto"/></term>
            /// <description><c>null</c></description>
            /// </item>
            /// </list>
            /// </summary>
            [MarshalAs(UnmanagedType.LPWStr)]
            public string profile;
            /// <summary>
            /// Pointer to a <see cref="Dot11Ssid"/> structure that specifies the SSID of the network to connect to.
            /// This field is optional. When set to <c>null</c>, all SSIDs in the profile will be tried.
            /// This field must not be <c>null</c> if <see cref="wlanConnectionMode"/> is set to <see cref="WlanConnectionMode.DiscoverySecure"/> or <see cref="WlanConnectionMode.DiscoveryUnsecure"/>.
            /// </summary>
            public IntPtr dot11SsidPtr;
            /// <summary>
            /// Pointer to a <see cref="Dot11BssidList"/> structure that contains the list of basic service set (BSS) identifiers desired for the connection.
            /// </summary>
            /// <remarks>
            /// On Windows XP SP2, must be set to <c>null</c>.
            /// </remarks>
            public IntPtr desiredBssidListPtr;
            /// <summary>
            /// A <see cref="Dot11BssType"/> value that indicates the BSS type of the network. If a profile is provided, this BSS type must be the same as the one in the profile.
            /// </summary>
            public Dot11BssType dot11BssType;
            /// <summary>
            /// Specifies ocnnection parameters.
            /// </summary>
            /// <remarks>
            /// On Windows XP SP2, must be set to 0.
            /// </remarks>
            public WlanConnectionFlags flags;
        }

        /// <summary>
        /// The connection state of an ad hoc network.
        /// </summary>
        public enum WlanAdhocNetworkState
        {
            /// <summary>
            /// The ad hoc network has been formed, but no client or host is connected to the network.
            /// </summary>
            Formed = 0,
            /// <summary>
            /// A client or host is connected to the ad hoc network.
            /// </summary>
            Connected = 1
        }

        [DllImport("wlanapi.dll")]
        public static extern int WlanConnect(
            [In] IntPtr clientHandle,
            [In, MarshalAs(UnmanagedType.LPStruct)] Guid interfaceGuid,
            [In] ref WlanConnectionParameters connectionParameters,
            IntPtr pReserved);

        [DllImport("wlanapi.dll")]
        public static extern int WlanDeleteProfile(
            [In] IntPtr clientHandle,
            [In, MarshalAs(UnmanagedType.LPStruct)] Guid interfaceGuid,
            [In, MarshalAs(UnmanagedType.LPWStr)] string profileName,
            IntPtr reservedPtr
        );

        [DllImport("wlanapi.dll")]
        public static extern int WlanGetNetworkBssList(
            [In] IntPtr clientHandle,
            [In, MarshalAs(UnmanagedType.LPStruct)] Guid interfaceGuid,
            [In] IntPtr dot11SsidInt,
            [In] Dot11BssType dot11BssType,
            [In] bool securityEnabled,
            IntPtr reservedPtr,
            [Out] out IntPtr wlanBssList
        );

        [StructLayout(LayoutKind.Sequential)]
        internal struct WlanBssListHeader
        {
            internal uint totalSize;
            internal uint numberOfItems;
        }

        /// <summary>
        /// Contains information about a basic service set (BSS).
        /// </summary>
        [StructLayout(LayoutKind.Sequential)]
        public struct WlanBssEntry
        {
            /// <summary>
            /// Contains the SSID of the access point (AP) associated with the BSS.
            /// </summary>
            public Dot11Ssid dot11Ssid;
            /// <summary>
            /// The identifier of the PHY on which the AP is operating.
            /// </summary>
            public uint phyId;
            /// <summary>
            /// Contains the BSS identifier.
            /// </summary>
            [MarshalAs(UnmanagedType.ByValArray, SizeConst = 6)]
            public byte[] dot11Bssid;
            /// <summary>
            /// Specifies whether the network is infrastructure or ad hoc.
            /// </summary>
            public Dot11BssType dot11BssType;
            public Dot11PhyType dot11BssPhyType;
            /// <summary>
            /// The received signal strength in dBm.
            /// </summary>
            public int rssi;
            /// <summary>
            /// The link quality reported by the driver. Ranges from 0-100.
            /// </summary>
            public uint linkQuality;
            /// <summary>
            /// If 802.11d is not implemented, the network interface card (NIC) must set this field to TRUE. If 802.11d is implemented (but not necessarily enabled), the NIC must set this field to TRUE if the BSS operation complies with the configured regulatory domain.
            /// </summary>
            public bool inRegDomain;
            /// <summary>
            /// Contains the beacon interval value from the beacon packet or probe response.
            /// </summary>
            public ushort beaconPeriod;
            /// <summary>
            /// The timestamp from the beacon packet or probe response.
            /// </summary>
            public ulong timestamp;
            /// <summary>
            /// The host timestamp value when the beacon or probe response is received.
            /// </summary>
            public ulong hostTimestamp;
            /// <summary>
            /// The capability value from the beacon packet or probe response.
            /// </summary>
            public ushort capabilityInformation;
            /// <summary>
            /// The frequency of the center channel, in kHz.
            /// </summary>
            public uint chCenterFrequency;
            /// <summary>
            /// Contains the set of data transfer rates supported by the BSS.
            /// </summary>
            public WlanRateSet wlanRateSet;
            /// <summary>
            /// Offset of the information element (IE) data blob.
            /// </summary>
            public uint ieOffset;
            /// <summary>
            /// Size of the IE data blob, in bytes.
            /// </summary>
            public uint ieSize;
        }

        /// <summary>
        /// Contains the set of supported data rates.
        /// </summary>
        [StructLayout(LayoutKind.Sequential)]
        public struct WlanRateSet
        {
            /// <summary>
            /// The length, in bytes, of <see cref="rateSet"/>.
            /// </summary>
            private uint rateSetLength;
            /// <summary>
            /// An array of supported data transfer rates.
            /// If the rate is a basic rate, the first bit of the rate value is set to 1.
            /// A basic rate is the data transfer rate that all stations in a basic service set (BSS) can use to receive frames from the wireless medium.
            /// </summary>
            [MarshalAs(UnmanagedType.ByValArray, SizeConst = 126)]
            private ushort[] rateSet;

            public ushort[] Rates
            {
                get
                {
                    ushort[] rates = new ushort[rateSetLength / sizeof(ushort)];
                    Array.Copy(rateSet, rates, rates.Length);
                    return rates;
                }
            }

            /// <summary>
            /// CalculateS the data transfer rate in Mbps for an arbitrary supported rate.
            /// </summary>
            /// <param name="rate"></param>
            /// <returns></returns>
            public double GetRateInMbps(int rate)
            {
                return (rateSet[rate] & 0x7FFF) * 0.5;
            }
        }

        /// <summary>
        /// Represents an error occurring during WLAN operations which indicate their failure via a <see cref="WlanReasonCode"/>.
        /// </summary>
        public class WlanException : Exception
        {
            private WlanReasonCode reasonCode;

            WlanException(WlanReasonCode reasonCode)
            {
                this.reasonCode = reasonCode;
            }

            /// <summary>
            /// Gets the WLAN reason code.
            /// </summary>
            /// <value>The WLAN reason code.</value>
            public WlanReasonCode ReasonCode
            {
                get { return reasonCode; }
            }

            /// <summary>
            /// Gets a message that describes the reason code.
            /// </summary>
            /// <value></value>
            /// <returns>The error message that explains the reason for the exception, or an empty string("").</returns>
            public override string Message
            {
                get
                {
                    StringBuilder sb = new StringBuilder(1024);
                    if (WlanReasonCodeToString(reasonCode, sb.Capacity, sb, IntPtr.Zero) == 0)
                        return sb.ToString();
                    else
                        return string.Empty;
                }
            }
        }

        // TODO: .NET-ify the WlanReasonCode enum (naming convention + docs).

        /// <summary>
        /// Specifies reasons for a failure of a WLAN operation.
        /// </summary>
        /// <remarks>
        /// To get the WLAN API native reason code identifiers, prefix the identifiers with <c>WLAN_REASON_CODE_</c>.
        /// </remarks>
        public enum WlanReasonCode
        {
            Success = 0,
            // general codes
            UNKNOWN = 0x10000 + 1,

            RANGE_SIZE = 0x10000,
            BASE = 0x10000 + RANGE_SIZE,

            // range for Auto Config
            //
            AC_BASE = 0x10000 + RANGE_SIZE,
            AC_CONNECT_BASE = (AC_BASE + RANGE_SIZE / 2),
            AC_END = (AC_BASE + RANGE_SIZE - 1),

            // range for profile manager
            // it has profile adding failure reason codes, but may not have
            // connection reason codes
            //
            PROFILE_BASE = 0x10000 + (7 * RANGE_SIZE),
            PROFILE_CONNECT_BASE = (PROFILE_BASE + RANGE_SIZE / 2),
            PROFILE_END = (PROFILE_BASE + RANGE_SIZE - 1),

            // range for MSM
            //
            MSM_BASE = 0x10000 + (2 * RANGE_SIZE),
            MSM_CONNECT_BASE = (MSM_BASE + RANGE_SIZE / 2),
            MSM_END = (MSM_BASE + RANGE_SIZE - 1),

            // range for MSMSEC
            //
            MSMSEC_BASE = 0x10000 + (3 * RANGE_SIZE),
            MSMSEC_CONNECT_BASE = (MSMSEC_BASE + RANGE_SIZE / 2),
            MSMSEC_END = (MSMSEC_BASE + RANGE_SIZE - 1),

            // AC network incompatible reason codes
            //
            NETWORK_NOT_COMPATIBLE = (AC_BASE + 1),
            PROFILE_NOT_COMPATIBLE = (AC_BASE + 2),

            // AC connect reason code
            //
            NO_AUTO_CONNECTION = (AC_CONNECT_BASE + 1),
            NOT_VISIBLE = (AC_CONNECT_BASE + 2),
            GP_DENIED = (AC_CONNECT_BASE + 3),
            USER_DENIED = (AC_CONNECT_BASE + 4),
            BSS_TYPE_NOT_ALLOWED = (AC_CONNECT_BASE + 5),
            IN_FAILED_LIST = (AC_CONNECT_BASE + 6),
            IN_BLOCKED_LIST = (AC_CONNECT_BASE + 7),
            SSID_LIST_TOO_LONG = (AC_CONNECT_BASE + 8),
            CONNECT_CALL_FAIL = (AC_CONNECT_BASE + 9),
            SCAN_CALL_FAIL = (AC_CONNECT_BASE + 10),
            NETWORK_NOT_AVAILABLE = (AC_CONNECT_BASE + 11),
            PROFILE_CHANGED_OR_DELETED = (AC_CONNECT_BASE + 12),
            KEY_MISMATCH = (AC_CONNECT_BASE + 13),
            USER_NOT_RESPOND = (AC_CONNECT_BASE + 14),

            // Profile validation errors
            //
            INVALID_PROFILE_SCHEMA = (PROFILE_BASE + 1),
            PROFILE_MISSING = (PROFILE_BASE + 2),
            INVALID_PROFILE_NAME = (PROFILE_BASE + 3),
            INVALID_PROFILE_TYPE = (PROFILE_BASE + 4),
            INVALID_PHY_TYPE = (PROFILE_BASE + 5),
            MSM_SECURITY_MISSING = (PROFILE_BASE + 6),
            IHV_SECURITY_NOT_SUPPORTED = (PROFILE_BASE + 7),
            IHV_OUI_MISMATCH = (PROFILE_BASE + 8),
            // IHV OUI not present but there is IHV settings in profile
            IHV_OUI_MISSING = (PROFILE_BASE + 9),
            // IHV OUI is present but there is no IHV settings in profile
            IHV_SETTINGS_MISSING = (PROFILE_BASE + 10),
            // both/conflict MSMSec and IHV security settings exist in profile
            CONFLICT_SECURITY = (PROFILE_BASE + 11),
            // no IHV or MSMSec security settings in profile
            SECURITY_MISSING = (PROFILE_BASE + 12),
            INVALID_BSS_TYPE = (PROFILE_BASE + 13),
            INVALID_ADHOC_CONNECTION_MODE = (PROFILE_BASE + 14),
            NON_BROADCAST_SET_FOR_ADHOC = (PROFILE_BASE + 15),
            AUTO_SWITCH_SET_FOR_ADHOC = (PROFILE_BASE + 16),
            AUTO_SWITCH_SET_FOR_MANUAL_CONNECTION = (PROFILE_BASE + 17),
            IHV_SECURITY_ONEX_MISSING = (PROFILE_BASE + 18),
            PROFILE_SSID_INVALID = (PROFILE_BASE + 19),
            TOO_MANY_SSID = (PROFILE_BASE + 20),

            // MSM network incompatible reasons
            //
            UNSUPPORTED_SECURITY_SET_BY_OS = (MSM_BASE + 1),
            UNSUPPORTED_SECURITY_SET = (MSM_BASE + 2),
            BSS_TYPE_UNMATCH = (MSM_BASE + 3),
            PHY_TYPE_UNMATCH = (MSM_BASE + 4),
            DATARATE_UNMATCH = (MSM_BASE + 5),

            // MSM connection failure reasons, to be defined
            // failure reason codes
            //
            // user called to disconnect
            USER_CANCELLED = (MSM_CONNECT_BASE + 1),
            // got disconnect while associating
            ASSOCIATION_FAILURE = (MSM_CONNECT_BASE + 2),
            // timeout for association
            ASSOCIATION_TIMEOUT = (MSM_CONNECT_BASE + 3),
            // pre-association security completed with failure
            PRE_SECURITY_FAILURE = (MSM_CONNECT_BASE + 4),
            // fail to start post-association security
            START_SECURITY_FAILURE = (MSM_CONNECT_BASE + 5),
            // post-association security completed with failure
            SECURITY_FAILURE = (MSM_CONNECT_BASE + 6),
            // security watchdog timeout
            SECURITY_TIMEOUT = (MSM_CONNECT_BASE + 7),
            // got disconnect from driver when roaming
            ROAMING_FAILURE = (MSM_CONNECT_BASE + 8),
            // failed to start security for roaming
            ROAMING_SECURITY_FAILURE = (MSM_CONNECT_BASE + 9),
            // failed to start security for adhoc-join
            ADHOC_SECURITY_FAILURE = (MSM_CONNECT_BASE + 10),
            // got disconnection from driver
            DRIVER_DISCONNECTED = (MSM_CONNECT_BASE + 11),
            // driver operation failed
            DRIVER_OPERATION_FAILURE = (MSM_CONNECT_BASE + 12),
            // Ihv service is not available
            IHV_NOT_AVAILABLE = (MSM_CONNECT_BASE + 13),
            // Response from ihv timed out
            IHV_NOT_RESPONDING = (MSM_CONNECT_BASE + 14),
            // Timed out waiting for driver to disconnect
            DISCONNECT_TIMEOUT = (MSM_CONNECT_BASE + 15),
            // An internal error prevented the operation from being completed.
            INTERNAL_FAILURE = (MSM_CONNECT_BASE + 16),
            // UI Request timed out.
            UI_REQUEST_TIMEOUT = (MSM_CONNECT_BASE + 17),
            // Roaming too often, post security is not completed after 5 times.
            TOO_MANY_SECURITY_ATTEMPTS = (MSM_CONNECT_BASE + 18),

            // MSMSEC reason codes
            //

            MSMSEC_MIN = MSMSEC_BASE,

            // Key index specified is not valid
            MSMSEC_PROFILE_INVALID_KEY_INDEX = (MSMSEC_BASE + 1),
            // Key required, PSK present
            MSMSEC_PROFILE_PSK_PRESENT = (MSMSEC_BASE + 2),
            // Invalid key length
            MSMSEC_PROFILE_KEY_LENGTH = (MSMSEC_BASE + 3),
            // Invalid PSK length
            MSMSEC_PROFILE_PSK_LENGTH = (MSMSEC_BASE + 4),
            // No auth/cipher specified
            MSMSEC_PROFILE_NO_AUTH_CIPHER_SPECIFIED = (MSMSEC_BASE + 5),
            // Too many auth/cipher specified
            MSMSEC_PROFILE_TOO_MANY_AUTH_CIPHER_SPECIFIED = (MSMSEC_BASE + 6),
            // Profile contains duplicate auth/cipher
            MSMSEC_PROFILE_DUPLICATE_AUTH_CIPHER = (MSMSEC_BASE + 7),
            // Profile raw data is invalid (1x or key data)
            MSMSEC_PROFILE_RAWDATA_INVALID = (MSMSEC_BASE + 8),
            // Invalid auth/cipher combination
            MSMSEC_PROFILE_INVALID_AUTH_CIPHER = (MSMSEC_BASE + 9),
            // 802.1x disabled when it's required to be enabled
            MSMSEC_PROFILE_ONEX_DISABLED = (MSMSEC_BASE + 10),
            // 802.1x enabled when it's required to be disabled
            MSMSEC_PROFILE_ONEX_ENABLED = (MSMSEC_BASE + 11),
            MSMSEC_PROFILE_INVALID_PMKCACHE_MODE = (MSMSEC_BASE + 12),
            MSMSEC_PROFILE_INVALID_PMKCACHE_SIZE = (MSMSEC_BASE + 13),
            MSMSEC_PROFILE_INVALID_PMKCACHE_TTL = (MSMSEC_BASE + 14),
            MSMSEC_PROFILE_INVALID_PREAUTH_MODE = (MSMSEC_BASE + 15),
            MSMSEC_PROFILE_INVALID_PREAUTH_THROTTLE = (MSMSEC_BASE + 16),
            // PreAuth enabled when PMK cache is disabled
            MSMSEC_PROFILE_PREAUTH_ONLY_ENABLED = (MSMSEC_BASE + 17),
            // Capability matching failed at network
            MSMSEC_CAPABILITY_NETWORK = (MSMSEC_BASE + 18),
            // Capability matching failed at NIC
            MSMSEC_CAPABILITY_NIC = (MSMSEC_BASE + 19),
            // Capability matching failed at profile
            MSMSEC_CAPABILITY_PROFILE = (MSMSEC_BASE + 20),
            // Network does not support specified discovery type
            MSMSEC_CAPABILITY_DISCOVERY = (MSMSEC_BASE + 21),
            // Passphrase contains invalid character
            MSMSEC_PROFILE_PASSPHRASE_CHAR = (MSMSEC_BASE + 22),
            // Key material contains invalid character
            MSMSEC_PROFILE_KEYMATERIAL_CHAR = (MSMSEC_BASE + 23),
            // Wrong key type specified for the auth/cipher pair
            MSMSEC_PROFILE_WRONG_KEYTYPE = (MSMSEC_BASE + 24),
            // "Mixed cell" suspected (AP not beaconing privacy, we have privacy enabled profile)
            MSMSEC_MIXED_CELL = (MSMSEC_BASE + 25),
            // Auth timers or number of timeouts in profile is incorrect
            MSMSEC_PROFILE_AUTH_TIMERS_INVALID = (MSMSEC_BASE + 26),
            // Group key update interval in profile is incorrect
            MSMSEC_PROFILE_INVALID_GKEY_INTV = (MSMSEC_BASE + 27),
            // "Transition network" suspected, trying legacy 802.11 security
            MSMSEC_TRANSITION_NETWORK = (MSMSEC_BASE + 28),
            // Key contains characters which do not map to ASCII
            MSMSEC_PROFILE_KEY_UNMAPPED_CHAR = (MSMSEC_BASE + 29),
            // Capability matching failed at profile (auth not found)
            MSMSEC_CAPABILITY_PROFILE_AUTH = (MSMSEC_BASE + 30),
            // Capability matching failed at profile (cipher not found)
            MSMSEC_CAPABILITY_PROFILE_CIPHER = (MSMSEC_BASE + 31),

            // Failed to queue UI request
            MSMSEC_UI_REQUEST_FAILURE = (MSMSEC_CONNECT_BASE + 1),
            // 802.1x authentication did not start within configured time
            MSMSEC_AUTH_START_TIMEOUT = (MSMSEC_CONNECT_BASE + 2),
            // 802.1x authentication did not complete within configured time
            MSMSEC_AUTH_SUCCESS_TIMEOUT = (MSMSEC_CONNECT_BASE + 3),
            // Dynamic key exchange did not start within configured time
            MSMSEC_KEY_START_TIMEOUT = (MSMSEC_CONNECT_BASE + 4),
            // Dynamic key exchange did not succeed within configured time
            MSMSEC_KEY_SUCCESS_TIMEOUT = (MSMSEC_CONNECT_BASE + 5),
            // Message 3 of 4 way handshake has no key data (RSN/WPA)
            MSMSEC_M3_MISSING_KEY_DATA = (MSMSEC_CONNECT_BASE + 6),
            // Message 3 of 4 way handshake has no IE (RSN/WPA)
            MSMSEC_M3_MISSING_IE = (MSMSEC_CONNECT_BASE + 7),
            // Message 3 of 4 way handshake has no Group Key (RSN)
            MSMSEC_M3_MISSING_GRP_KEY = (MSMSEC_CONNECT_BASE + 8),
            // Matching security capabilities of IE in M3 failed (RSN/WPA)
            MSMSEC_PR_IE_MATCHING = (MSMSEC_CONNECT_BASE + 9),
            // Matching security capabilities of Secondary IE in M3 failed (RSN)
            MSMSEC_SEC_IE_MATCHING = (MSMSEC_CONNECT_BASE + 10),
            // Required a pairwise key but AP configured only group keys
            MSMSEC_NO_PAIRWISE_KEY = (MSMSEC_CONNECT_BASE + 11),
            // Message 1 of group key handshake has no key data (RSN/WPA)
            MSMSEC_G1_MISSING_KEY_DATA = (MSMSEC_CONNECT_BASE + 12),
            // Message 1 of group key handshake has no group key
            MSMSEC_G1_MISSING_GRP_KEY = (MSMSEC_CONNECT_BASE + 13),
            // AP reset secure bit after connection was secured
            MSMSEC_PEER_INDICATED_INSECURE = (MSMSEC_CONNECT_BASE + 14),
            // 802.1x indicated there is no authenticator but profile requires 802.1x
            MSMSEC_NO_AUTHENTICATOR = (MSMSEC_CONNECT_BASE + 15),
            // Plumbing settings to NIC failed
            MSMSEC_NIC_FAILURE = (MSMSEC_CONNECT_BASE + 16),
            // Operation was cancelled by caller
            MSMSEC_CANCELLED = (MSMSEC_CONNECT_BASE + 17),
            // Key was in incorrect format
            MSMSEC_KEY_FORMAT = (MSMSEC_CONNECT_BASE + 18),
            // Security downgrade detected
            MSMSEC_DOWNGRADE_DETECTED = (MSMSEC_CONNECT_BASE + 19),
            // PSK mismatch suspected
            MSMSEC_PSK_MISMATCH_SUSPECTED = (MSMSEC_CONNECT_BASE + 20),
            // Forced failure because connection method was not secure
            MSMSEC_FORCED_FAILURE = (MSMSEC_CONNECT_BASE + 21),
            // ui request couldn't be queued or user pressed cancel
            MSMSEC_SECURITY_UI_FAILURE = (MSMSEC_CONNECT_BASE + 22),

            MSMSEC_MAX = MSMSEC_END
        }

        /// <summary>
        /// Contains information about connection related notifications.
        /// </summary>
        /// <remarks>
        /// Corresponds to the native <c>WLAN_CONNECTION_NOTIFICATION_DATA</c> type.
        /// </remarks>
        [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
        public struct WlanConnectionNotificationData
        {
            /// <remarks>
            /// On Windows XP SP 2, only <see cref="WlanConnectionMode.Profile"/> is supported.
            /// </remarks>
            public WlanConnectionMode wlanConnectionMode;
            /// <summary>
            /// The name of the profile used for the connection. Profile names are case-sensitive.
            /// </summary>
            [MarshalAs(UnmanagedType.ByValTStr, SizeConst = 32)]
            public string profileName;
            /// <summary>
            /// The SSID of the association.
            /// </summary>
            public Dot11Ssid dot11Ssid;
            /// <summary>
            /// The BSS network type.
            /// </summary>
            public Dot11BssType dot11BssType;
            /// <summary>
            /// Indicates whether security is enabled for this connection.
            /// </summary>
            public bool securityEnabled;
            /// <summary>
            /// Indicates the reason for an operation failure.
            /// This field has a value of <see cref="WlanReasonCode.Success"/> for all connection-related notifications except <see cref="WlanNotificationCodeAcm.ConnectionComplete"/>.
            /// If the connection fails, this field indicates the reason for the failure.
            /// </summary>
            public WlanReasonCode wlanReasonCode;
            /// <summary>
            /// This field contains the XML presentation of the profile used for discovery, if the connection succeeds.
            /// </summary>
            [MarshalAs(UnmanagedType.ByValTStr, SizeConst = 1)]
            public string profileXml;
        }

        /// <summary>
        /// Indicates the state of an interface.
        /// </summary>
        /// <remarks>
        /// Corresponds to the native <c>WLAN_INTERFACE_STATE</c> type.
        /// </remarks>
        public enum WlanInterfaceState
        {
            /// <summary>
            /// The interface is not ready to operate.
            /// </summary>
            NotReady = 0,
            /// <summary>
            /// The interface is connected to a network.
            /// </summary>
            Connected = 1,
            /// <summary>
            /// The interface is the first node in an ad hoc network. No peer has connected.
            /// </summary>
            AdHocNetworkFormed = 2,
            /// <summary>
            /// The interface is disconnecting from the current network.
            /// </summary>
            Disconnecting = 3,
            /// <summary>
            /// The interface is not connected to any network.
            /// </summary>
            Disconnected = 4,
            /// <summary>
            /// The interface is attempting to associate with a network.
            /// </summary>
            Associating = 5,
            /// <summary>
            /// Auto configuration is discovering the settings for the network.
            /// </summary>
            Discovering = 6,
            /// <summary>
            /// The interface is in the process of authenticating.
            /// </summary>
            Authenticating = 7
        }

        /// <summary>
        /// Contains the SSID of an interface.
        /// </summary>
        public struct Dot11Ssid
        {
            /// <summary>
            /// The length, in bytes, of the <see cref="SSID"/> array.
            /// </summary>
            public uint SSIDLength;
            /// <summary>
            /// The SSID.
            /// </summary>
            [MarshalAs(UnmanagedType.ByValArray, SizeConst = 32)]
            public byte[] SSID;
        }

        /// <summary>
        /// Defines an 802.11 PHY and media type.
        /// </summary>
        /// <remarks>
        /// Corresponds to the native <c>DOT11_PHY_TYPE</c> type.
        /// </remarks>
        public enum Dot11PhyType : uint
        {
            /// <summary>
            /// Specifies an unknown or uninitialized PHY type.
            /// </summary>
            Unknown = 0,
            /// <summary>
            /// Specifies any PHY type.
            /// </summary>
            Any = Unknown,
            /// <summary>
            /// Specifies a frequency-hopping spread-spectrum (FHSS) PHY. Bluetooth devices can use FHSS or an adaptation of FHSS.
            /// </summary>
            FHSS = 1,
            /// <summary>
            /// Specifies a direct sequence spread spectrum (DSSS) PHY.
            /// </summary>
            DSSS = 2,
            /// <summary>
            /// Specifies an infrared (IR) baseband PHY.
            /// </summary>
            IrBaseband = 3,
            /// <summary>
            /// Specifies an orthogonal frequency division multiplexing (OFDM) PHY. 802.11a devices can use OFDM.
            /// </summary>
            OFDM = 4,
            /// <summary>
            /// Specifies a high-rate DSSS (HRDSSS) PHY.
            /// </summary>
            HRDSSS = 5,
            /// <summary>
            /// Specifies an extended rate PHY (ERP). 802.11g devices can use ERP.
            /// </summary>
            ERP = 6,
            /// <summary>
            /// Specifies the start of the range that is used to define PHY types that are developed by an independent hardware vendor (IHV).
            /// </summary>
            IHV_Start = 0x80000000,
            /// <summary>
            /// Specifies the end of the range that is used to define PHY types that are developed by an independent hardware vendor (IHV).
            /// </summary>
            IHV_End = 0xffffffff
        }

        /// <summary>
        /// Defines a basic service set (BSS) network type.
        /// </summary>
        /// <remarks>
        /// Corresponds to the native <c>DOT11_BSS_TYPE</c> type.
        /// </remarks>
        public enum Dot11BssType
        {
            /// <summary>
            /// Specifies an infrastructure BSS network.
            /// </summary>
            Infrastructure = 1,
            /// <summary>
            /// Specifies an independent BSS (IBSS) network.
            /// </summary>
            Independent = 2,
            /// <summary>
            /// Specifies either infrastructure or IBSS network.
            /// </summary>
            Any = 3
        }

        /// <summary>
        /// Contains association attributes for a connection
        /// </summary>
        /// <remarks>
        /// Corresponds to the native <c>WLAN_ASSOCIATION_ATTRIBUTES</c> type.
        /// </remarks>
        [StructLayout(LayoutKind.Sequential)]
        public struct WlanAssociationAttributes
        {
            /// <summary>
            /// The SSID of the association.
            /// </summary>
            public Dot11Ssid dot11Ssid;
            /// <summary>
            /// Specifies whether the network is infrastructure or ad hoc.
            /// </summary>
            public Dot11BssType dot11BssType;
            /// <summary>
            /// The BSSID of the association.
            /// </summary>
            [MarshalAs(UnmanagedType.ByValArray, SizeConst = 6)]
            public byte[] dot11Bssid;
            /// <summary>
            /// The physical type of the association.
            /// </summary>
            public Dot11PhyType dot11PhyType;
            /// <summary>
            /// The position of the <see cref="Dot11PhyType"/> value in the structure containing the list of PHY types.
            /// </summary>
            public uint dot11PhyIndex;
            /// <summary>
            /// A percentage value that represents the signal quality of the network.
            /// This field contains a value between 0 and 100.
            /// A value of 0 implies an actual RSSI signal strength of -100 dbm.
            /// A value of 100 implies an actual RSSI signal strength of -50 dbm.
            /// You can calculate the RSSI signal strength value for values between 1 and 99 using linear interpolation.
            /// </summary>
            public uint wlanSignalQuality;
            /// <summary>
            /// The receiving rate of the association.
            /// </summary>
            public uint rxRate;
            /// <summary>
            /// The transmission rate of the association.
            /// </summary>
            public uint txRate;

            /// <summary>
            /// Gets the BSSID of the associated access point.
            /// </summary>
            /// <value>The BSSID.</value>
            public PhysicalAddress Dot11Bssid
            {
                get { return new PhysicalAddress(dot11Bssid); }
            }
        }

        /// <summary>
        /// Defines the mode of connection.
        /// </summary>
        /// <remarks>
        /// Corresponds to the native <c>WLAN_CONNECTION_MODE</c> type.
        /// </remarks>
        public enum WlanConnectionMode
        {
            /// <summary>
            /// A profile will be used to make the connection.
            /// </summary>
            Profile = 0,
            /// <summary>
            /// A temporary profile will be used to make the connection.
            /// </summary>
            TemporaryProfile,
            /// <summary>
            /// Secure discovery will be used to make the connection.
            /// </summary>
            DiscoverySecure,
            /// <summary>
            /// Unsecure discovery will be used to make the connection.
            /// </summary>
            DiscoveryUnsecure,
            /// <summary>
            /// A connection will be made automatically, generally using a persistent profile.
            /// </summary>
            Auto,
            /// <summary>
            /// Not used.
            /// </summary>
            Invalid
        }

        /// <summary>
        /// Defines a wireless LAN authentication algorithm.
        /// </summary>
        /// <remarks>
        /// Corresponds to the native <c>DOT11_AUTH_ALGORITHM</c> type.
        /// </remarks>
        public enum Dot11AuthAlgorithm : uint
        {
            /// <summary>
            /// Specifies an IEEE 802.11 Open System authentication algorithm.
            /// </summary>
            IEEE80211_Open = 1,
            /// <summary>
            /// Specifies an 802.11 Shared Key authentication algorithm that requires the use of a pre-shared Wired Equivalent Privacy (WEP) key for the 802.11 authentication.
            /// </summary>
            IEEE80211_SharedKey = 2,
            /// <summary>
            /// Specifies a Wi-Fi Protected Access (WPA) algorithm. IEEE 802.1X port authentication is performed by the supplicant, authenticator, and authentication server. Cipher keys are dynamically derived through the authentication process.
            /// <para>This algorithm is valid only for BSS types of <see cref="Dot11BssType.Infrastructure"/>.</para>
            /// <para>When the WPA algorithm is enabled, the 802.11 station will associate only with an access point whose beacon or probe responses contain the authentication suite of type 1 (802.1X) within the WPA information element (IE).</para>
            /// </summary>
            WPA = 3,
            /// <summary>
            /// Specifies a WPA algorithm that uses preshared keys (PSK). IEEE 802.1X port authentication is performed by the supplicant and authenticator. Cipher keys are dynamically derived through a preshared key that is used on both the supplicant and authenticator.
            /// <para>This algorithm is valid only for BSS types of <see cref="Dot11BssType.Infrastructure"/>.</para>
            /// <para>When the WPA PSK algorithm is enabled, the 802.11 station will associate only with an access point whose beacon or probe responses contain the authentication suite of type 2 (preshared key) within the WPA IE.</para>
            /// </summary>
            WPA_PSK = 4,
            /// <summary>
            /// This value is not supported.
            /// </summary>
            WPA_None = 5,
            /// <summary>
            /// Specifies an 802.11i Robust Security Network Association (RSNA) algorithm. WPA2 is one such algorithm. IEEE 802.1X port authentication is performed by the supplicant, authenticator, and authentication server. Cipher keys are dynamically derived through the authentication process.
            /// <para>This algorithm is valid only for BSS types of <see cref="Dot11BssType.Infrastructure"/>.</para>
            /// <para>When the RSNA algorithm is enabled, the 802.11 station will associate only with an access point whose beacon or probe responses contain the authentication suite of type 1 (802.1X) within the RSN IE.</para>
            /// </summary>
            RSNA = 6,
            /// <summary>
            /// Specifies an 802.11i RSNA algorithm that uses PSK. IEEE 802.1X port authentication is performed by the supplicant and authenticator. Cipher keys are dynamically derived through a preshared key that is used on both the supplicant and authenticator.
            /// <para>This algorithm is valid only for BSS types of <see cref="Dot11BssType.Infrastructure"/>.</para>
            /// <para>When the RSNA PSK algorithm is enabled, the 802.11 station will associate only with an access point whose beacon or probe responses contain the authentication suite of type 2(preshared key) within the RSN IE.</para>
            /// </summary>
            RSNA_PSK = 7,
            /// <summary>
            /// Indicates the start of the range that specifies proprietary authentication algorithms that are developed by an IHV.
            /// </summary>
            /// <remarks>
            /// This enumerator is valid only when the miniport driver is operating in Extensible Station (ExtSTA) mode.
            /// </remarks>
            IHV_Start = 0x80000000,
            /// <summary>
            /// Indicates the end of the range that specifies proprietary authentication algorithms that are developed by an IHV.
            /// </summary>
            /// <remarks>
            /// This enumerator is valid only when the miniport driver is operating in Extensible Station (ExtSTA) mode.
            /// </remarks>
            IHV_End = 0xffffffff
        }

        /// <summary>
        /// Defines a cipher algorithm for data encryption and decryption.
        /// </summary>
        /// <remarks>
        /// Corresponds to the native <c>DOT11_CIPHER_ALGORITHM</c> type.
        /// </remarks>
        public enum Dot11CipherAlgorithm : uint
        {
            /// <summary>
            /// Specifies that no cipher algorithm is enabled or supported.
            /// </summary>
            None = 0x00,
            /// <summary>
            /// Specifies a Wired Equivalent Privacy (WEP) algorithm, which is the RC4-based algorithm that is specified in the 802.11-1999 standard. This enumerator specifies the WEP cipher algorithm with a 40-bit cipher key.
            /// </summary>
            WEP40 = 0x01,
            /// <summary>
            /// Specifies a Temporal Key Integrity Protocol (TKIP) algorithm, which is the RC4-based cipher suite that is based on the algorithms that are defined in the WPA specification and IEEE 802.11i-2004 standard. This cipher also uses the Michael Message Integrity Code (MIC) algorithm for forgery protection.
            /// </summary>
            TKIP = 0x02,
            /// <summary>
            /// Specifies an AES-CCMP algorithm, as specified in the IEEE 802.11i-2004 standard and RFC 3610. Advanced Encryption Standard (AES) is the encryption algorithm defined in FIPS PUB 197.
            /// </summary>
            CCMP = 0x04,
            /// <summary>
            /// Specifies a WEP cipher algorithm with a 104-bit cipher key.
            /// </summary>
            WEP104 = 0x05,
            /// <summary>
            /// Specifies a Robust Security Network (RSN) Use Group Key cipher suite. For more information about the Use Group Key cipher suite, refer to Clause 7.3.2.9.1 of the IEEE 802.11i-2004 standard.
            /// </summary>
            WPA_UseGroup = 0x100,
            /// <summary>
            /// Specifies a Wifi Protected Access (WPA) Use Group Key cipher suite. For more information about the Use Group Key cipher suite, refer to Clause 7.3.2.9.1 of the IEEE 802.11i-2004 standard.
            /// </summary>
            RSN_UseGroup = 0x100,
            /// <summary>
            /// Specifies a WEP cipher algorithm with a cipher key of any length.
            /// </summary>
            WEP = 0x101,
            /// <summary>
            /// Specifies the start of the range that is used to define proprietary cipher algorithms that are developed by an independent hardware vendor (IHV).
            /// </summary>
            IHV_Start = 0x80000000,
            /// <summary>
            /// Specifies the end of the range that is used to define proprietary cipher algorithms that are developed by an IHV.
            /// </summary>
            IHV_End = 0xffffffff
        }

        /// <summary>
        /// Defines the security attributes for a wireless connection.
        /// </summary>
        /// <remarks>
        /// Corresponds to the native <c>WLAN_SECURITY_ATTRIBUTES</c> type.
        /// </remarks>
        [StructLayout(LayoutKind.Sequential)]
        public struct WlanSecurityAttributes
        {
            /// <summary>
            /// Indicates whether security is enabled for this connection.
            /// </summary>
            [MarshalAs(UnmanagedType.Bool)]
            public bool securityEnabled;
            [MarshalAs(UnmanagedType.Bool)]
            public bool oneXEnabled;
            /// <summary>
            /// The authentication algorithm.
            /// </summary>
            public Dot11AuthAlgorithm dot11AuthAlgorithm;
            /// <summary>
            /// The cipher algorithm.
            /// </summary>
            public Dot11CipherAlgorithm dot11CipherAlgorithm;
        }

        /// <summary>
        /// Defines the attributes of a wireless connection.
        /// </summary>
        /// <remarks>
        /// Corresponds to the native <c>WLAN_CONNECTION_ATTRIBUTES</c> type.
        /// </remarks>
        [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
        public struct WlanConnectionAttributes
        {
            /// <summary>
            /// The state of the interface.
            /// </summary>
            public WlanInterfaceState isState;
            /// <summary>
            /// The mode of the connection.
            /// </summary>
            public WlanConnectionMode wlanConnectionMode;
            /// <summary>
            /// The name of the profile used for the connection. Profile names are case-sensitive.
            /// </summary>
            [MarshalAs(UnmanagedType.ByValTStr, SizeConst = 256)]
            public string profileName;
            /// <summary>
            /// The attributes of the association.
            /// </summary>
            public WlanAssociationAttributes wlanAssociationAttributes;
            /// <summary>
            /// The security attributes of the connection.
            /// </summary>
            public WlanSecurityAttributes wlanSecurityAttributes;
        }

        /// <summary>
        /// Contains information about a LAN interface.
        /// </summary>
        [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
        public struct WlanInterfaceInfo
        {
            /// <summary>
            /// The GUID of the interface.
            /// </summary>
            public Guid interfaceGuid;
            /// <summary>
            /// The description of the interface.
            /// </summary>
            [MarshalAs(UnmanagedType.ByValTStr, SizeConst = 256)]
            public string interfaceDescription;
            /// <summary>
            /// The current state of the interface.
            /// </summary>
            public WlanInterfaceState isState;
        }

        /// <summary>
        /// The header of the list returned by <see cref="WlanEnumInterfaces"/>.
        /// </summary>
        [StructLayout(LayoutKind.Sequential)]
        internal struct WlanInterfaceInfoListHeader
        {
            public uint numberOfItems;
            public uint index;
        }

        /// <summary>
        /// The header of the list returned by <see cref="WlanGetProfileList"/>.
        /// </summary>
        [StructLayout(LayoutKind.Sequential)]
        internal struct WlanProfileInfoListHeader
        {
            public uint numberOfItems;
            public uint index;
        }

        /// <summary>
        /// Contains basic information about a profile.
        /// </summary>
        [StructLayout(LayoutKind.Sequential, CharSet = CharSet.Unicode)]
        public struct WlanProfileInfo
        {
            /// <summary>
            /// The name of the profile. This value may be the name of a domain if the profile is for provisioning. Profile names are case-sensitive.
            /// </summary>
            [MarshalAs(UnmanagedType.ByValTStr, SizeConst = 256)]
            public string profileName;
            /// <summary>
            /// Profile flags.
            /// </summary>
            public WlanProfileFlags profileFlags;
        }

        /// <summary>
        /// Flags that specify the miniport driver's current operation mode.
        /// </summary>
        [Flags]
        public enum Dot11OperationMode : uint
        {
            Unknown = 0x00000000,
            Station = 0x00000001,
            AP = 0x00000002,
            /// <summary>
            /// Specifies that the miniport driver supports the Extensible Station (ExtSTA) operation mode.
            /// </summary>
            ExtensibleStation = 0x00000004,
            /// <summary>
            /// Specifies that the miniport driver supports the Network Monitor (NetMon) operation mode.
            /// </summary>
            NetworkMonitor = 0x80000000
        }
        #endregion

        /// <summary>
        /// Helper method to wrap calls to Native WiFi API methods.
        /// If the method falls, throws an exception containing the error code.
        /// </summary>
        /// <param name="win32ErrorCode">The error code.</param>
        [DebuggerStepThrough]
        internal static void ThrowIfError(int win32ErrorCode)
        {
            if (win32ErrorCode != 0)
                throw new Win32Exception(win32ErrorCode);
        }
    }
	/// <summary>
	/// Represents a client to the Zeroconf (Native Wifi) service.
	/// </summary>
	/// <remarks>
	/// This class is the entrypoint to Native Wifi management. To manage WiFi settings, create an instance
	/// of this class.
	/// </remarks>
	public class WlanClient
	{
		/// <summary>
		/// Represents a Wifi network interface.
		/// </summary>
		public class WlanInterface
		{
			private WlanClient client;
			private Wlan.WlanInterfaceInfo info;
            public bool scanInProgress = false;

			#region Events
			/// <summary>
			/// Represents a method that will handle <see cref="WlanNotification"/> events.
			/// </summary>
			/// <param name="notifyData">The notification data.</param>
			public delegate void WlanNotificationEventHandler(Wlan.WlanNotificationData notifyData);

			/// <summary>
			/// Represents a method that will handle <see cref="WlanConnectionNotification"/> events.
			/// </summary>
			/// <param name="notifyData">The notification data.</param>
			/// <param name="connNotifyData">The notification data.</param>
			public delegate void WlanConnectionNotificationEventHandler(Wlan.WlanNotificationData notifyData, Wlan.WlanConnectionNotificationData connNotifyData);

			/// <summary>
			/// Represents a method that will handle <see cref="WlanReasonNotification"/> events.
			/// </summary>
			/// <param name="notifyData">The notification data.</param>
			/// <param name="reasonCode">The reason code.</param>
			public delegate void WlanReasonNotificationEventHandler(Wlan.WlanNotificationData notifyData, Wlan.WlanReasonCode reasonCode);

			/// <summary>
			/// Occurs when an event of any kind occurs on a WLAN interface.
			/// </summary>
			public event WlanNotificationEventHandler WlanNotification;

			/// <summary>
			/// Occurs when a WLAN interface changes connection state.
			/// </summary>
			public event WlanConnectionNotificationEventHandler WlanConnectionNotification;

			/// <summary>
			/// Occurs when a WLAN operation fails due to some reason.
			/// </summary>
			public event WlanReasonNotificationEventHandler WlanReasonNotification;

			#endregion

			#region Event queue
			private bool queueEvents;
			private AutoResetEvent eventQueueFilled = new AutoResetEvent(false);
			private Queue<object> eventQueue = new Queue<object>();

			private struct WlanConnectionNotificationEventData
			{
				public Wlan.WlanNotificationData notifyData;
				public Wlan.WlanConnectionNotificationData connNotifyData;
			}
			private struct WlanReasonNotificationData
			{
				public Wlan.WlanNotificationData notifyData;
				public Wlan.WlanReasonCode reasonCode;
			}
			#endregion

			internal WlanInterface(WlanClient client, Wlan.WlanInterfaceInfo info)
			{
				this.client = client;
				this.info = info;
			}

			/// <summary>
			/// Sets a parameter of the interface whose data type is <see cref="int"/>.
			/// </summary>
			/// <param name="opCode">The opcode of the parameter.</param>
			/// <param name="value">The value to set.</param>
			private void SetInterfaceInt(Wlan.WlanIntfOpcode opCode, int value)
			{
				IntPtr valuePtr = Marshal.AllocHGlobal(sizeof(int));
				Marshal.WriteInt32(valuePtr, value);
				try
				{
					Wlan.ThrowIfError(
						Wlan.WlanSetInterface(client.clientHandle, info.interfaceGuid, opCode, sizeof(int), valuePtr, IntPtr.Zero));
				}
				finally
				{
					Marshal.FreeHGlobal(valuePtr);
				}
			}

			/// <summary>
			/// Gets a parameter of the interface whose data type is <see cref="int"/>.
			/// </summary>
			/// <param name="opCode">The opcode of the parameter.</param>
			/// <returns>The integer value.</returns>
			private int GetInterfaceInt(Wlan.WlanIntfOpcode opCode)
			{
				IntPtr valuePtr;
				int valueSize;
				Wlan.WlanOpcodeValueType opcodeValueType;
				Wlan.ThrowIfError(
					Wlan.WlanQueryInterface(client.clientHandle, info.interfaceGuid, opCode, IntPtr.Zero, out valueSize, out valuePtr, out opcodeValueType));
				try
				{
					return Marshal.ReadInt32(valuePtr);
				}
				finally
				{
					Wlan.WlanFreeMemory(valuePtr);
				}
			}

			/// <summary>
			/// Gets or sets a value indicating whether this <see cref="WlanInterface"/> is automatically configured.
			/// </summary>
			/// <value><c>true</c> if "autoconf" is enabled; otherwise, <c>false</c>.</value>
			public bool Autoconf
			{
				get
				{
					return GetInterfaceInt(Wlan.WlanIntfOpcode.AutoconfEnabled) != 0;
				}
				set
				{
					SetInterfaceInt(Wlan.WlanIntfOpcode.AutoconfEnabled, value ? 1 : 0);
				}
			}

			/// <summary>
			/// Gets or sets the BSS type for the indicated interface.
			/// </summary>
			/// <value>The type of the BSS.</value>
			public Wlan.Dot11BssType BssType
			{
				get
				{
					return (Wlan.Dot11BssType) GetInterfaceInt(Wlan.WlanIntfOpcode.BssType);
				}
				set
				{
					SetInterfaceInt(Wlan.WlanIntfOpcode.BssType, (int)value);
				}
			}

			/// <summary>
			/// Gets the state of the interface.
			/// </summary>
			/// <value>The state of the interface.</value>
			public Wlan.WlanInterfaceState InterfaceState
			{
				get
				{
					return (Wlan.WlanInterfaceState)GetInterfaceInt(Wlan.WlanIntfOpcode.InterfaceState);
				}
			}

			/// <summary>
			/// Gets the channel.
			/// </summary>
			/// <value>The channel.</value>
			/// <remarks>Not supported on Windows XP SP2.</remarks>
			public int Channel
			{
				get
				{
					return GetInterfaceInt(Wlan.WlanIntfOpcode.ChannelNumber);
				}
			}

			/// <summary>
			/// Gets the RSSI.
			/// </summary>
			/// <value>The RSSI.</value>
			/// <remarks>Not supported on Windows XP SP2.</remarks>
			public int RSSI
			{
				get
				{
					return GetInterfaceInt(Wlan.WlanIntfOpcode.RSSI);
				}
			}

			/// <summary>
			/// Gets the current operation mode.
			/// </summary>
			/// <value>The current operation mode.</value>
			/// <remarks>Not supported on Windows XP SP2.</remarks>
			public Wlan.Dot11OperationMode CurrentOperationMode
			{
				get
				{
					return (Wlan.Dot11OperationMode) GetInterfaceInt(Wlan.WlanIntfOpcode.CurrentOperationMode);
				}
			}

			/// <summary>
			/// Gets the attributes of the current connection.
			/// </summary>
			/// <value>The current connection attributes.</value>
			/// <exception cref="Win32Exception">An exception with code 0x0000139F (The group or resource is not in the correct state to perform the requested operation.) will be thrown if the interface is not connected to a network.</exception>
			public Wlan.WlanConnectionAttributes CurrentConnection
			{
				get
				{
					int valueSize;
					IntPtr valuePtr;
					Wlan.WlanOpcodeValueType opcodeValueType;
					Wlan.ThrowIfError(
						Wlan.WlanQueryInterface(client.clientHandle, info.interfaceGuid, Wlan.WlanIntfOpcode.CurrentConnection, IntPtr.Zero, out valueSize, out valuePtr, out opcodeValueType));
					try
					{
							return (Wlan.WlanConnectionAttributes)Marshal.PtrToStructure(valuePtr, typeof(Wlan.WlanConnectionAttributes));
					}
					finally
					{
						Wlan.WlanFreeMemory(valuePtr);
					}
				}
			}

			/// <summary>
			/// Requests a scan for available networks.
			/// </summary>
			/// <remarks>
			/// The method returns immediately. Progress is reported through the <see cref="WlanNotification"/> event.
			/// </remarks>
			public void Scan()
			{
                scanInProgress = true;
				Wlan.ThrowIfError(
					Wlan.WlanScan(client.clientHandle, info.interfaceGuid, IntPtr.Zero, IntPtr.Zero, IntPtr.Zero));
			}

			/// <summary>
			/// Converts a pointer to a available networks list (header + entries) to an array of available network entries.
			/// </summary>
			/// <param name="bssListPtr">A pointer to an available networks list's header.</param>
			/// <returns>An array of available network entries.</returns>
			private Wlan.WlanAvailableNetwork[] ConvertAvailableNetworkListPtr(IntPtr availNetListPtr)
			{
				Wlan.WlanAvailableNetworkListHeader availNetListHeader = (Wlan.WlanAvailableNetworkListHeader)Marshal.PtrToStructure(availNetListPtr, typeof(Wlan.WlanAvailableNetworkListHeader));
				long availNetListIt = availNetListPtr.ToInt64() + Marshal.SizeOf(typeof(Wlan.WlanAvailableNetworkListHeader));
				Wlan.WlanAvailableNetwork[] availNets = new Wlan.WlanAvailableNetwork[availNetListHeader.numberOfItems];
				for (int i = 0; i < availNetListHeader.numberOfItems; ++i)
				{
					availNets[i] = (Wlan.WlanAvailableNetwork)Marshal.PtrToStructure(new IntPtr(availNetListIt), typeof(Wlan.WlanAvailableNetwork));
					availNetListIt += Marshal.SizeOf(typeof(Wlan.WlanAvailableNetwork));
				}
				return availNets;
			}

			/// <summary>
			/// Retrieves the list of available networks.
			/// </summary>
			/// <param name="flags">Controls the type of networks returned.</param>
			/// <returns>A list of the available networks.</returns>
			public Wlan.WlanAvailableNetwork[] GetAvailableNetworkList(Wlan.WlanGetAvailableNetworkFlags flags)
			{
				IntPtr availNetListPtr;
				Wlan.ThrowIfError(
					Wlan.WlanGetAvailableNetworkList(client.clientHandle, info.interfaceGuid, flags, IntPtr.Zero, out availNetListPtr));
				try
				{
					return ConvertAvailableNetworkListPtr(availNetListPtr);
				}
				finally
				{
					Wlan.WlanFreeMemory(availNetListPtr);
				}
			}

			/// <summary>
			/// Converts a pointer to a BSS list (header + entries) to an array of BSS entries.
			/// </summary>
			/// <param name="bssListPtr">A pointer to a BSS list's header.</param>
			/// <returns>An array of BSS entries.</returns>
			private Wlan.WlanBssEntry[] ConvertBssListPtr(IntPtr bssListPtr)
			{
				Wlan.WlanBssListHeader bssListHeader = (Wlan.WlanBssListHeader)Marshal.PtrToStructure(bssListPtr, typeof(Wlan.WlanBssListHeader));
				long bssListIt = bssListPtr.ToInt64() + Marshal.SizeOf(typeof(Wlan.WlanBssListHeader));
				Wlan.WlanBssEntry[] bssEntries = new Wlan.WlanBssEntry[bssListHeader.numberOfItems];
				for (int i=0; i<bssListHeader.numberOfItems; ++i)
				{
					bssEntries[i] = (Wlan.WlanBssEntry)Marshal.PtrToStructure(new IntPtr(bssListIt), typeof(Wlan.WlanBssEntry));
					bssListIt += Marshal.SizeOf(typeof(Wlan.WlanBssEntry));
				}
				return bssEntries;
			}

			/// <summary>
			/// Retrieves the basic service sets (BSS) list of all available networks.
			/// </summary>
			public Wlan.WlanBssEntry[] GetNetworkBssList()
			{
				IntPtr bssListPtr;
				Wlan.ThrowIfError(
					Wlan.WlanGetNetworkBssList(client.clientHandle, info.interfaceGuid, IntPtr.Zero, Wlan.Dot11BssType.Any, false, IntPtr.Zero, out bssListPtr));
				try
				{
					return ConvertBssListPtr(bssListPtr);
				}
				finally
				{
					Wlan.WlanFreeMemory(bssListPtr);
				}
			}

			/// <summary>
			/// Retrieves the basic service sets (BSS) list of the specified network.
			/// </summary>
			/// <param name="ssid">Specifies the SSID of the network from which the BSS list is requested.</param>
			/// <param name="bssType">Indicates the BSS type of the network.</param>
			/// <param name="securityEnabled">Indicates whether security is enabled on the network.</param>
			public Wlan.WlanBssEntry[] GetNetworkBssList(Wlan.Dot11Ssid ssid, Wlan.Dot11BssType bssType, bool securityEnabled)
			{
				IntPtr ssidPtr = Marshal.AllocHGlobal(Marshal.SizeOf(ssid));
				Marshal.StructureToPtr(ssid, ssidPtr, false);
				try
				{
					IntPtr bssListPtr;
					Wlan.ThrowIfError(
						Wlan.WlanGetNetworkBssList(client.clientHandle, info.interfaceGuid, ssidPtr, bssType, securityEnabled, IntPtr.Zero, out bssListPtr));
					try
					{
						return ConvertBssListPtr(bssListPtr);
					}
					finally
					{
						Wlan.WlanFreeMemory(bssListPtr);
					}
				}
				finally
				{
					Marshal.FreeHGlobal(ssidPtr);
				}
			}

			/// <summary>
			/// Connects to a network defined by a connection parameters structure.
			/// </summary>
			/// <param name="connectionParams">The connection parameters.</param>
			protected void Connect(Wlan.WlanConnectionParameters connectionParams)
			{
				Wlan.ThrowIfError(
					Wlan.WlanConnect(client.clientHandle, info.interfaceGuid, ref connectionParams, IntPtr.Zero));
			}

			/// <summary>
			/// Requests a connection (association) to the specified wireless network.
			/// </summary>
			/// <remarks>
			/// The method returns immediately. Progress is reported through the <see cref="WlanNotification"/> event.
			/// </remarks>
			public void Connect(Wlan.WlanConnectionMode connectionMode, Wlan.Dot11BssType bssType, string profile)
			{
				Wlan.WlanConnectionParameters connectionParams = new Wlan.WlanConnectionParameters();
				connectionParams.wlanConnectionMode = connectionMode;
				connectionParams.profile = profile;
				connectionParams.dot11BssType = bssType;
				connectionParams.flags = 0;
				Connect(connectionParams);
			}

			/// <summary>
			/// Connects (associates) to the specified wireless network, returning either on a success to connect
			/// or a failure.
			/// </summary>
			/// <param name="connectionMode"></param>
			/// <param name="bssType"></param>
			/// <param name="profile"></param>
			/// <param name="connectTimeout"></param>
			/// <returns></returns>
			public bool ConnectSynchronously(Wlan.WlanConnectionMode connectionMode, Wlan.Dot11BssType bssType, string profile, int connectTimeout)
			{
				queueEvents = true;
				try
				{
					Connect(connectionMode, bssType, profile);
					while (queueEvents && eventQueueFilled.WaitOne(connectTimeout, true))
					{
						lock (eventQueue)
						{
							while (eventQueue.Count != 0)
							{
								object e = eventQueue.Dequeue();
								if (e is WlanConnectionNotificationEventData)
								{
									WlanConnectionNotificationEventData wlanConnectionData = (WlanConnectionNotificationEventData)e;
									// Check if the conditions are good to indicate either success or failure.
									if (wlanConnectionData.notifyData.notificationSource == Wlan.WlanNotificationSource.ACM)
									{
										switch ((Wlan.WlanNotificationCodeAcm)wlanConnectionData.notifyData.notificationCode)
										{
											case Wlan.WlanNotificationCodeAcm.ConnectionComplete:
												if (wlanConnectionData.connNotifyData.profileName == profile)
													return true;
												break;
										}
									}
									break;
								}
							}
						}
					}
				}
				finally
				{
					queueEvents = false;
					eventQueue.Clear();
				}
				return false; // timeout expired and no "connection complete"
			}

			/// <summary>
			/// Connects to the specified wireless network.
			/// </summary>
			/// <remarks>
			/// The method returns immediately. Progress is reported through the <see cref="WlanNotification"/> event.
			/// </remarks>
			public void Connect(Wlan.WlanConnectionMode connectionMode, Wlan.Dot11BssType bssType, Wlan.Dot11Ssid ssid, Wlan.WlanConnectionFlags flags)
			{
				Wlan.WlanConnectionParameters connectionParams = new Wlan.WlanConnectionParameters();
				connectionParams.wlanConnectionMode = connectionMode;
				connectionParams.dot11SsidPtr = Marshal.AllocHGlobal(Marshal.SizeOf(ssid));
				Marshal.StructureToPtr(ssid, connectionParams.dot11SsidPtr, false);
				connectionParams.dot11BssType = bssType;
				connectionParams.flags = flags;
				Connect(connectionParams);
				Marshal.DestroyStructure(connectionParams.dot11SsidPtr, ssid.GetType());
				Marshal.FreeHGlobal(connectionParams.dot11SsidPtr);
			}

			/// <summary>
			/// Deletes a profile.
			/// </summary>
			/// <param name="profileName">
			/// The name of the profile to be deleted. Profile names are case-sensitive.
			/// On Windows XP SP2, the supplied name must match the profile name derived automatically from the SSID of the network. For an infrastructure network profile, the SSID must be supplied for the profile name. For an ad hoc network profile, the supplied name must be the SSID of the ad hoc network followed by <c>-adhoc</c>.
			/// </param>
			public void DeleteProfile(string profileName)
			{
				Wlan.ThrowIfError(
					Wlan.WlanDeleteProfile(client.clientHandle, info.interfaceGuid, profileName, IntPtr.Zero));
			}

			/// <summary>
			/// Sets the profile.
			/// </summary>
			/// <param name="flags">The flags to set on the profile.</param>
			/// <param name="profileXml">The XML representation of the profile. On Windows XP SP 2, special care should be taken to adhere to its limitations.</param>
			/// <param name="overwrite">If a profile by the given name already exists, then specifies whether to overwrite it (if <c>true</c>) or return an error (if <c>false</c>).</param>
			/// <returns>The resulting code indicating a success or the reason why the profile wasn't valid.</returns>
			public Wlan.WlanReasonCode SetProfile(Wlan.WlanProfileFlags flags, string profileXml, bool overwrite)
			{
				Wlan.WlanReasonCode reasonCode;
				Wlan.ThrowIfError(
						Wlan.WlanSetProfile(client.clientHandle, info.interfaceGuid, flags, profileXml, null, overwrite, IntPtr.Zero, out reasonCode));
				return reasonCode;
			}

			/// <summary>
			/// Gets the profile's XML specification.
			/// </summary>
			/// <param name="profileName">The name of the profile.</param>
			/// <returns>The XML document.</returns>
			public string GetProfileXml(string profileName)
			{
				IntPtr profileXmlPtr;
				Wlan.WlanProfileFlags flags;
				Wlan.WlanAccess access;
				Wlan.ThrowIfError(
					Wlan.WlanGetProfile(client.clientHandle, info.interfaceGuid, profileName, IntPtr.Zero, out profileXmlPtr, out flags,
					               out access));
				try
				{
					return Marshal.PtrToStringUni(profileXmlPtr);
				}
				finally
				{
					Wlan.WlanFreeMemory(profileXmlPtr);
				}
			}

			/// <summary>
			/// Gets the information of all profiles on this interface.
			/// </summary>
			/// <returns>The profiles information.</returns>
			public Wlan.WlanProfileInfo[] GetProfiles()
			{
				IntPtr profileListPtr;
				Wlan.ThrowIfError(
					Wlan.WlanGetProfileList(client.clientHandle, info.interfaceGuid, IntPtr.Zero, out profileListPtr));
				try
				{
					Wlan.WlanProfileInfoListHeader header = (Wlan.WlanProfileInfoListHeader) Marshal.PtrToStructure(profileListPtr, typeof(Wlan.WlanProfileInfoListHeader));
					Wlan.WlanProfileInfo[] profileInfos = new Wlan.WlanProfileInfo[header.numberOfItems];
					long profileListIterator = profileListPtr.ToInt64() + Marshal.SizeOf(header);
					for (int i=0; i<header.numberOfItems; ++i)
					{
						Wlan.WlanProfileInfo profileInfo = (Wlan.WlanProfileInfo) Marshal.PtrToStructure(new IntPtr(profileListIterator), typeof(Wlan.WlanProfileInfo));
						profileInfos[i] = profileInfo;
						profileListIterator += Marshal.SizeOf(profileInfo);
					}
					return profileInfos;
				}
				finally
				{
					Wlan.WlanFreeMemory(profileListPtr);
				}
			}

			internal void OnWlanConnection(Wlan.WlanNotificationData notifyData, Wlan.WlanConnectionNotificationData connNotifyData)
			{
				if (WlanConnectionNotification != null)
					WlanConnectionNotification(notifyData, connNotifyData);

				if (queueEvents)
				{
					WlanConnectionNotificationEventData queuedEvent = new WlanConnectionNotificationEventData();
					queuedEvent.notifyData = notifyData;
					queuedEvent.connNotifyData = connNotifyData;
					EnqueueEvent(queuedEvent);
				}
			}

			internal void OnWlanReason(Wlan.WlanNotificationData notifyData, Wlan.WlanReasonCode reasonCode)
			{
				if (WlanReasonNotification != null)
					WlanReasonNotification(notifyData, reasonCode);
				if (queueEvents)
				{
					WlanReasonNotificationData queuedEvent = new WlanReasonNotificationData();
					queuedEvent.notifyData = notifyData;
					queuedEvent.reasonCode = reasonCode;
					EnqueueEvent(queuedEvent);
				}
			}

			internal void OnWlanNotification(Wlan.WlanNotificationData notifyData)
			{
				if (WlanNotification != null)
					WlanNotification(notifyData);
			}

			/// <summary>
			/// Enqueues a notification event to be processed serially.
			/// </summary>
			private void EnqueueEvent(object queuedEvent)
			{
				lock (eventQueue)
					eventQueue.Enqueue(queuedEvent);
				eventQueueFilled.Set();
			}

			/// <summary>
			/// Gets the network interface of this wireless interface.
			/// </summary>
			/// <remarks>
			/// The network interface allows querying of generic network properties such as the interface's IP address.
			/// </remarks>
			public NetworkInterface NetworkInterface
			{
				get
				{
                    // Do not cache the NetworkInterface; We need it fresh
                    // each time cause otherwise it caches the IP information.
					foreach (NetworkInterface netIface in NetworkInterface.GetAllNetworkInterfaces())
					{
						Guid netIfaceGuid = new Guid(netIface.Id);
						if (netIfaceGuid.Equals(info.interfaceGuid))
						{
							return netIface;
						}
					}
                    return null;
				}
			}

			/// <summary>
			/// The GUID of the interface (same content as the <see cref="System.Net.NetworkInformation.NetworkInterface.Id"/> value).
			/// </summary>
			public Guid InterfaceGuid
			{
				get { return info.interfaceGuid; }
			}

			/// <summary>
			/// The description of the interface.
			/// This is a user-immutable string containing the vendor and model name of the adapter.
			/// </summary>
			public string InterfaceDescription
			{
				get { return info.interfaceDescription; }
			}

			/// <summary>
			/// The friendly name given to the interface by the user (e.g. "Local Area Network Connection").
			/// </summary>
			public string InterfaceName
			{
				get { return NetworkInterface.Name; }
			}
		}

		private IntPtr clientHandle;
		private uint negotiatedVersion;
		private Wlan.WlanNotificationCallbackDelegate wlanNotificationCallback;

		private Dictionary<Guid,WlanInterface> ifaces = new Dictionary<Guid,WlanInterface>();

		/// <summary>
		/// Creates a new instance of a Native Wifi service client.
		/// </summary>
		public WlanClient()
		{
			Wlan.ThrowIfError(
				Wlan.WlanOpenHandle(Wlan.WLAN_CLIENT_VERSION_XP_SP2, IntPtr.Zero, out negotiatedVersion, out clientHandle));
			try
			{
				Wlan.WlanNotificationSource prevSrc;
				wlanNotificationCallback = new Wlan.WlanNotificationCallbackDelegate(OnWlanNotification);
				Wlan.ThrowIfError(
					Wlan.WlanRegisterNotification(clientHandle, Wlan.WlanNotificationSource.All, false, wlanNotificationCallback, IntPtr.Zero, IntPtr.Zero, out prevSrc));
			}
			catch
			{
				Wlan.WlanCloseHandle(clientHandle, IntPtr.Zero);
				throw;
			}
		}

		~WlanClient()
		{
			Wlan.WlanCloseHandle(clientHandle, IntPtr.Zero);
		}

		private Wlan.WlanConnectionNotificationData? ParseWlanConnectionNotification(ref Wlan.WlanNotificationData notifyData)
		{
			int expectedSize = Marshal.SizeOf(typeof(Wlan.WlanConnectionNotificationData));
			if (notifyData.dataSize < expectedSize)
				return null;

			Wlan.WlanConnectionNotificationData connNotifyData =
				(Wlan.WlanConnectionNotificationData)
				Marshal.PtrToStructure(notifyData.dataPtr, typeof(Wlan.WlanConnectionNotificationData));
			if (connNotifyData.wlanReasonCode == Wlan.WlanReasonCode.Success)
			{
				IntPtr profileXmlPtr = new IntPtr(
					notifyData.dataPtr.ToInt64() +
					Marshal.OffsetOf(typeof(Wlan.WlanConnectionNotificationData), "profileXml").ToInt64());
				connNotifyData.profileXml = Marshal.PtrToStringUni(profileXmlPtr);
			}
			return connNotifyData;
		}

		private void OnWlanNotification(ref Wlan.WlanNotificationData notifyData, IntPtr context)
		{
			WlanInterface wlanIface = ifaces.ContainsKey(notifyData.interfaceGuid) ? ifaces[notifyData.interfaceGuid] : null;

			switch(notifyData.notificationSource)
			{
				case Wlan.WlanNotificationSource.ACM:
					switch((Wlan.WlanNotificationCodeAcm)notifyData.notificationCode)
					{
						case Wlan.WlanNotificationCodeAcm.ConnectionStart:
						case Wlan.WlanNotificationCodeAcm.ConnectionComplete:
						case Wlan.WlanNotificationCodeAcm.ConnectionAttemptFail:
						case Wlan.WlanNotificationCodeAcm.Disconnecting:
						case Wlan.WlanNotificationCodeAcm.Disconnected:
							Wlan.WlanConnectionNotificationData? connNotifyData = ParseWlanConnectionNotification(ref notifyData);
							if (connNotifyData.HasValue)
								if (wlanIface != null)
									wlanIface.OnWlanConnection(notifyData, connNotifyData.Value);
							break;
                        case Wlan.WlanNotificationCodeAcm.ScanComplete:
                            if (wlanIface != null) {
                                wlanIface.scanInProgress = false;
                            }
                            break;
						case Wlan.WlanNotificationCodeAcm.ScanFail:
							{
								int expectedSize = Marshal.SizeOf(typeof (Wlan.WlanReasonCode));
								if (notifyData.dataSize >= expectedSize)
								{
									Wlan.WlanReasonCode reasonCode = (Wlan.WlanReasonCode) Marshal.ReadInt32(notifyData.dataPtr);
									if (wlanIface != null)
										wlanIface.OnWlanReason(notifyData, reasonCode);
								}
							}
							break;
					}
					break;
				case Wlan.WlanNotificationSource.MSM:
					switch((Wlan.WlanNotificationCodeMsm)notifyData.notificationCode)
					{
						case Wlan.WlanNotificationCodeMsm.Associating:
						case Wlan.WlanNotificationCodeMsm.Associated:
						case Wlan.WlanNotificationCodeMsm.Authenticating:
						case Wlan.WlanNotificationCodeMsm.Connected:
						case Wlan.WlanNotificationCodeMsm.RoamingStart:
						case Wlan.WlanNotificationCodeMsm.RoamingEnd:
						case Wlan.WlanNotificationCodeMsm.Disassociating:
						case Wlan.WlanNotificationCodeMsm.Disconnected:
						case Wlan.WlanNotificationCodeMsm.PeerJoin:
						case Wlan.WlanNotificationCodeMsm.PeerLeave:
						case Wlan.WlanNotificationCodeMsm.AdapterRemoval:
							Wlan.WlanConnectionNotificationData? connNotifyData = ParseWlanConnectionNotification(ref notifyData);
							if (connNotifyData.HasValue)
								if (wlanIface != null)
									wlanIface.OnWlanConnection(notifyData, connNotifyData.Value);
							break;
					}
					break;
			}

			if (wlanIface != null)
				wlanIface.OnWlanNotification(notifyData);
		}

		/// <summary>
		/// Gets the WLAN interfaces.
		/// </summary>
		/// <value>The WLAN interfaces.</value>
		public WlanInterface[] Interfaces
		{
			get
			{
				IntPtr ifaceList;
				Wlan.ThrowIfError(
					Wlan.WlanEnumInterfaces(clientHandle, IntPtr.Zero, out ifaceList));
				try
				{
					Wlan.WlanInterfaceInfoListHeader header =
						(Wlan.WlanInterfaceInfoListHeader) Marshal.PtrToStructure(ifaceList, typeof (Wlan.WlanInterfaceInfoListHeader));
					Int64 listIterator = ifaceList.ToInt64() + Marshal.SizeOf(header);
					WlanInterface[] interfaces = new WlanInterface[header.numberOfItems];
					List<Guid> currentIfaceGuids = new List<Guid>();
					for (int i = 0; i < header.numberOfItems; ++i)
					{
						Wlan.WlanInterfaceInfo info =
							(Wlan.WlanInterfaceInfo) Marshal.PtrToStructure(new IntPtr(listIterator), typeof (Wlan.WlanInterfaceInfo));
						listIterator += Marshal.SizeOf(info);
						WlanInterface wlanIface;
						currentIfaceGuids.Add(info.interfaceGuid);
						if (ifaces.ContainsKey(info.interfaceGuid))
							wlanIface = ifaces[info.interfaceGuid];
						else
							wlanIface = new WlanInterface(this, info);
						interfaces[i] = wlanIface;
						ifaces[info.interfaceGuid] = wlanIface;
					}

					// Remove stale interfaces
					Queue<Guid> deadIfacesGuids = new Queue<Guid>();
					foreach (Guid ifaceGuid in ifaces.Keys)
					{
						if (!currentIfaceGuids.Contains(ifaceGuid))
							deadIfacesGuids.Enqueue(ifaceGuid);
					}
					while(deadIfacesGuids.Count != 0)
					{
						Guid deadIfaceGuid = deadIfacesGuids.Dequeue();
						ifaces.Remove(deadIfaceGuid);
					}

					return interfaces;
				}
				finally
				{
					Wlan.WlanFreeMemory(ifaceList);
				}
			}
		}

		/// <summary>
		/// Gets a string that describes a specified reason code.
		/// </summary>
		/// <param name="reasonCode">The reason code.</param>
		/// <returns>The string.</returns>
		public string GetStringForReasonCode(Wlan.WlanReasonCode reasonCode)
		{
			StringBuilder sb = new StringBuilder(1024); // the 1024 size here is arbitrary; the WlanReasonCodeToString docs fail to specify a recommended size
			Wlan.ThrowIfError(
				Wlan.WlanReasonCodeToString(reasonCode, sb.Capacity, sb, IntPtr.Zero));
			return sb.ToString();
		}
	}
}
