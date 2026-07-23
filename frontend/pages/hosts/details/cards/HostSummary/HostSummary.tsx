import React from "react";
import classnames from "classnames";
import { formatInTimeZone } from "date-fns-tz";
import {
  IHostMdmProfile,
  BootstrapPackageStatus,
  isEnrolledInMdm,
  isWindowsDiskEncryptionStatus,
  isLinuxDiskEncryptionStatus,
} from "interfaces/mdm";
import { IOSSettings, IHostMaintenanceWindow } from "interfaces/host";
import {
  isAndroid,
  isIPadOrIPhone,
  isDiskEncryptionSupportedLinuxPlatform,
  isOsSettingsDisplayPlatform,
} from "interfaces/platform";

import { getHostStatus, getHostStatusTooltipText } from "pages/hosts/helpers";

import TooltipWrapper from "components/TooltipWrapper";
import Card from "components/Card";
import DataSet from "components/DataSet";
import StatusIndicator from "components/StatusIndicator";
import IssuesIndicator from "pages/hosts/components/IssuesIndicator";

import { DATE_FNS_FORMAT_STRINGS } from "utilities/constants";

import OSSettingsIndicator from "./OSSettingsIndicator";
import BootstrapPackageIndicator from "./BootstrapPackageIndicator/BootstrapPackageIndicator";

import {
  generateHostNameSettingIfEligible,
  generateLinuxDiskEncryptionSetting,
  generateRecoveryLockPasswordSetting,
  generateWinDiskEncryptionSetting,
} from "../../helpers";

const baseClass = "host-summary-card";

interface IBootstrapPackageData {
  status?: BootstrapPackageStatus | "";
  details?: string;
}

interface IHostSummaryProps {
  summaryData: any; // TODO: create interfaces for this and use consistently across host pages and related helpers
  bootstrapPackageData?: IBootstrapPackageData;
  isPremiumTier?: boolean;
  toggleOSSettingsModal?: () => void;
  toggleBootstrapPackageModal?: () => void;
  hostSettings?: IHostMdmProfile[];
  osSettings?: IOSSettings;
  className?: string;
}

const HostSummary = ({
  summaryData,
  bootstrapPackageData,
  isPremiumTier,
  toggleOSSettingsModal,
  toggleBootstrapPackageModal,
  hostSettings,
  osSettings,
  className,
}: IHostSummaryProps): JSX.Element => {
  const classNames = classnames(baseClass, className);

  const { status, platform, os_version, mdm } = summaryData;

  // Derive a local copy so we can append the synthetic disk-encryption,
  // recovery-lock, and host-name rows without mutating the hostSettings prop.
  let derivedHostSettings = hostSettings;

  const isAndroidHost = isAndroid(platform);
  const isIosOrIpadosHost = isIPadOrIPhone(platform);

  const renderIssues = () => (
    <DataSet
      title="Issues"
      value={
        <IssuesIndicator
          totalIssuesCount={summaryData.issues.total_issues_count}
          criticalVulnerabilitiesCount={
            summaryData.issues.critical_vulnerabilities_count
          }
          failingPoliciesCount={summaryData.issues.failing_policies_count}
          tooltipPosition="bottom"
        />
      }
    />
  );

  const renderHostTeam = () => (
    <DataSet
      title="Fleet"
      value={
        summaryData.team_name !== "---" ? (
          `${summaryData.team_name}`
        ) : (
          <span className="no-team">Unassigned</span>
        )
      }
    />
  );

  const renderMaintenanceWindow = ({
    starts_at,
    timezone,
  }: IHostMaintenanceWindow) => {
    const formattedStartsAt = formatInTimeZone(
      starts_at,
      // since startsAt is already localized and contains offset information, this 2nd parameter is
      // logically redundant. It's included here to allow use of date-fns-tz.formatInTimeZone instead of date-fns.format, which
      // allows us to format a UTC datetime without converting to the user-agent local time.
      timezone || "UTC",
      DATE_FNS_FORMAT_STRINGS.dateAtTime
    );

    const tip =
      timezone && timezone !== "UTC" ? (
        <>
          End user&apos;s time zone:
          <br />
          (GMT{starts_at.slice(-6)}) {timezone.replace("_", " ")}
        </>
      ) : (
        <>
          End user&apos;s timezone unavailable.
          <br />
          Displaying in UTC.
        </>
      );

    return (
      <DataSet
        title="Scheduled maintenance"
        value={
          <TooltipWrapper tipContent={tip}>{formattedStartsAt}</TooltipWrapper>
        }
      />
    );
  };

  // for windows and linux hosts we have to manually add a profile for disk encryption
  // as this is not currently included in the `profiles` value from the API
  // response for windows and linux hosts.
  if (
    platform === "windows" &&
    osSettings?.disk_encryption?.status &&
    isWindowsDiskEncryptionStatus(osSettings.disk_encryption.status)
  ) {
    const winDiskEncryptionSetting: IHostMdmProfile = generateWinDiskEncryptionSetting(
      osSettings.disk_encryption.status,
      osSettings.disk_encryption.detail
    );
    derivedHostSettings = derivedHostSettings
      ? [...derivedHostSettings, winDiskEncryptionSetting]
      : [winDiskEncryptionSetting];
  }

  if (
    isDiskEncryptionSupportedLinuxPlatform(platform, os_version) &&
    osSettings?.disk_encryption?.status &&
    isLinuxDiskEncryptionStatus(osSettings.disk_encryption.status)
  ) {
    const linuxDiskEncryptionSetting: IHostMdmProfile = generateLinuxDiskEncryptionSetting(
      osSettings.disk_encryption.status,
      osSettings.disk_encryption.detail
    );
    derivedHostSettings = derivedHostSettings
      ? [...derivedHostSettings, linuxDiskEncryptionSetting]
      : [linuxDiskEncryptionSetting];
  }

  if (
    platform === "darwin" &&
    isEnrolledInMdm(mdm?.enrollment_status ?? null) &&
    osSettings?.recovery_lock_password?.status
  ) {
    const recoveryLockSetting = generateRecoveryLockPasswordSetting(
      osSettings.recovery_lock_password.status,
      osSettings.recovery_lock_password.detail
    );
    derivedHostSettings = derivedHostSettings
      ? [...derivedHostSettings, recoveryLockSetting]
      : [recoveryLockSetting];
  }

  // The host name template row (macOS/iOS/iPadOS) is synthetic like the rows
  // above, so it must be added here too — otherwise a host whose only OS setting
  // is the host name wouldn't surface the "OS settings" indicator that opens the
  // modal.
  const hostNameSetting = generateHostNameSettingIfEligible(
    platform,
    mdm?.enrollment_status ?? null,
    osSettings
  );
  if (hostNameSetting) {
    derivedHostSettings = derivedHostSettings
      ? [...derivedHostSettings, hostNameSetting]
      : [hostNameSetting];
  }

  const showStatus = !isIosOrIpadosHost && !isAndroidHost;
  const showTeam = !!isPremiumTier;
  const showOsSettings =
    isOsSettingsDisplayPlatform(platform, os_version) &&
    !!derivedHostSettings &&
    derivedHostSettings.length > 0;
  const showIssues =
    summaryData.issues?.total_issues_count > 0 &&
    !isIosOrIpadosHost &&
    !isAndroidHost;
  const showBootstrapPackage =
    !!bootstrapPackageData?.status && !isIosOrIpadosHost && !isAndroidHost;
  const showMaintenanceWindow =
    !!isPremiumTier &&
    // TODO - refactor normalizeEmptyValues pattern
    !!summaryData.maintenance_window &&
    summaryData.maintenance_window !== "---";

  // Hide the card entirely when nothing inside it would render (e.g. Free tier
  // Android host with no OS settings) — otherwise an empty card sits above the
  // Vitals section (#49441).
  if (
    !showStatus &&
    !showTeam &&
    !showOsSettings &&
    !showIssues &&
    !showBootstrapPackage &&
    !showMaintenanceWindow
  ) {
    return <></>;
  }

  return (
    <Card
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
      className={classNames}
    >
      {showStatus && (
        <DataSet
          title="Status"
          value={
            <StatusIndicator
              value={getHostStatus(status, mdm?.enrollment_status)}
              tooltip={{
                tooltipText: getHostStatusTooltipText(
                  getHostStatus(status, mdm?.enrollment_status)
                ),
                position: "bottom",
              }}
            />
          }
        />
      )}
      {showTeam && renderHostTeam()}
      {showOsSettings && derivedHostSettings && (
        <DataSet
          className={`${baseClass}__os-settings`}
          title="OS settings"
          value={
            <OSSettingsIndicator
              profiles={derivedHostSettings}
              onClick={toggleOSSettingsModal}
            />
          }
        />
      )}
      {showIssues && renderIssues()}
      {showBootstrapPackage && bootstrapPackageData?.status && (
        <DataSet
          title="Bootstrap package"
          value={
            <BootstrapPackageIndicator
              status={bootstrapPackageData.status}
              onClick={toggleBootstrapPackageModal}
            />
          }
        />
      )}
      {showMaintenanceWindow &&
        renderMaintenanceWindow(summaryData.maintenance_window)}
    </Card>
  );
};

export default HostSummary;
