import React from "react";
import { Meta, StoryObj } from "@storybook/react";
import { NotificationContext } from "context/notification";
import { IHostMdmProfileWithAddedStatus } from "./OSSettingsTableConfig";
import OSSettingsTable from "./OSSettingsTable";

const noop = async () => {};

// Provide a minimal NotificationContext so the error cell's renderFlash works
const NotificationWrapper = ({ children }: { children: React.ReactNode }) => (
  <NotificationContext.Provider
    value={{
      notification: { alertType: null, isVisible: false, message: null, persistOnPageChange: false },
      renderFlash: () => {},
      renderMultiFlash: () => {},
      hideFlash: () => {},
    }}
  >
    {children}
  </NotificationContext.Provider>
);

const meta: Meta<typeof OSSettingsTable> = {
  component: OSSettingsTable,
  title: "Pages/Hosts/OSSettings/AndroidONCWithCertDependency",
  decorators: [
    (Story) => (
      <NotificationWrapper>
        <div style={{ width: 800, padding: 24 }}>
          <Story />
        </div>
      </NotificationWrapper>
    ),
  ],
  args: {
    canResendProfiles: true,
    resendRequest: noop,
    onProfileResent: () => {},
  },
};

export default meta;

type Story = StoryObj<typeof OSSettingsTable>;

const androidONCWithheld: IHostMdmProfileWithAddedStatus = {
  profile_uuid: "g-onc-wifi-001",
  name: "Corporate Wi-Fi (EAP-TLS)",
  operation_type: "install",
  platform: "android",
  status: "pending",
  detail:
    'Waiting for certificate "wifi-cert" to be installed on the host before applying this profile.',
  scope: null,
  managed_local_account: null,
};

const androidCameraPolicy: IHostMdmProfileWithAddedStatus = {
  profile_uuid: "g-camera-002",
  name: "Camera disabled",
  operation_type: "install",
  platform: "android",
  status: "pending",
  detail: "",
  scope: null,
  managed_local_account: null,
};

const androidCameraVerified: IHostMdmProfileWithAddedStatus = {
  profile_uuid: "g-camera-002",
  name: "Camera disabled",
  operation_type: "install",
  platform: "android",
  status: "verified",
  detail: "",
  scope: null,
  managed_local_account: null,
};

const androidONCReleased: IHostMdmProfileWithAddedStatus = {
  profile_uuid: "g-onc-wifi-001",
  name: "Corporate Wi-Fi (EAP-TLS)",
  operation_type: "install",
  platform: "android",
  status: "verified",
  detail: "",
  scope: null,
  managed_local_account: null,
};

const androidCertPending: IHostMdmProfileWithAddedStatus = {
  profile_uuid: "cert-wifi-cert-001",
  name: "wifi-cert",
  operation_type: "install",
  platform: "android",
  status: "pending",
  detail: "",
  certificate_template_id: 42,
  scope: null,
  managed_local_account: null,
};

const androidCertVerified: IHostMdmProfileWithAddedStatus = {
  profile_uuid: "cert-wifi-cert-001",
  name: "wifi-cert",
  operation_type: "install",
  platform: "android",
  status: "verified",
  detail: "",
  certificate_template_id: 42,
  scope: null,
  managed_local_account: null,
};

const androidCertFailed: IHostMdmProfileWithAddedStatus = {
  profile_uuid: "cert-wifi-cert-001",
  name: "wifi-cert",
  operation_type: "install",
  platform: "android",
  status: "failed",
  detail: "SCEP enrollment failed: CA server unreachable after 3 attempts",
  certificate_template_id: 42,
  scope: null,
  managed_local_account: null,
};

const androidONCKeyPairNotFound: IHostMdmProfileWithAddedStatus = {
  profile_uuid: "g-onc-wifi-001",
  name: "Corporate Wi-Fi (EAP-TLS)",
  operation_type: "install",
  platform: "android",
  status: "failed",
  detail:
    'nonComplianceDetail: ONC_WIFI_KEY_PAIR_ALIAS_NOT_CORRESPONDING_TO_EXISTING_KEY — the alias "wifi-cert" does not correspond to an existing key pair installed on the device.',
  scope: null,
  managed_local_account: null,
};

/**
 * **Phase 1: Certificate pending** - The "wifi-cert" certificate is still being
 * installed on the device. The ONC Wi-Fi profile is withheld until the cert is
 * ready. The camera policy profile is applied normally.
 */
export const CertificatePending: Story = {
  args: {
    tableData: [androidCertPending, androidONCWithheld, androidCameraPolicy],
  },
};

/**
 * **Phase 2: Certificate verified** - The "wifi-cert" certificate was
 * successfully installed. The ONC Wi-Fi profile has been released and both
 * profiles are now verified.
 */
export const CertificateVerified: Story = {
  args: {
    tableData: [androidCertVerified, androidONCReleased, androidCameraVerified],
  },
};

/**
 * **Certificate failed, Wi-Fi alias not found** - The "wifi-cert" certificate
 * failed to install, so the ONC Wi-Fi profile was sent as-is. Android reported
 * ONC_WIFI_KEY_PAIR_ALIAS_NOT_CORRESPONDING_TO_EXISTING_KEY because the key
 * pair referenced by ClientCertKeyPairAlias does not exist on the device.
 */
export const CertificateFailedKeyPairNotFound: Story = {
  args: {
    tableData: [androidCertFailed, androidONCKeyPairNotFound, androidCameraVerified],
  },
};
