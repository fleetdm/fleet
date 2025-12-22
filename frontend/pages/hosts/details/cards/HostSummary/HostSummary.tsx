import React from "react";
import classnames from "classnames";
import { formatInTimeZone } from "date-fns-tz";
import {
  IHostMdmProfile,
  BootstrapPackageStatus,
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

import getHostStatusTooltipText from "pages/hosts/helpers";

import TooltipWrapper from "components/TooltipWrapper";
import Card from "components/Card";
import DataSet from "components/DataSet";
import StatusIndicator from "components/StatusIndicator";
import IssuesIndicator from "pages/hosts/components/IssuesIndicator";

import {
  DATE_FNS_FORMAT_STRINGS,
} from "utilities/constants";

import OSSettingsIndicator from "./OSSettingsIndicator";
import BootstrapPackageIndicator from "./BootstrapPackageIndicator/BootstrapPackageIndicator";

import {
  generateLinuxDiskEncryptionSetting,
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

  const {
    status,
    platform,
    os_version,
  } = summaryData;

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
      title="Team"
      value={
        summaryData.team_name !== "---" ? (
          `${summaryData.team_name}`
        ) : (
          <span className="no-team">No team</span>
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
    hostSettings = hostSettings
      ? [...hostSettings, winDiskEncryptionSetting]
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
    hostSettings = hostSettings
      ? [...hostSettings, linuxDiskEncryptionSetting]
      : [linuxDiskEncryptionSetting];
  }

  return (
    <Card
      borderRadiusSize="xxlarge"
      paddingSize="xlarge"
      className={classNames}
    >
      {!isIosOrIpadosHost && !isAndroidHost && (
        <DataSet
          title="Status"
          value={
            <StatusIndicator
              value={status || ""} // temporary work around of integration test bug
              tooltip={{
                tooltipText: getHostStatusTooltipText(status),
                position: "bottom",
              }}
            />
          }
        />
      )}
      {isPremiumTier && renderHostTeam()}
      {isOsSettingsDisplayPlatform(platform, os_version) &&
        hostSettings &&
        hostSettings.length > 0 && (
          <DataSet
            title="OS settings"
            value={
              <OSSettingsIndicator
                profiles={hostSettings}
                onClick={toggleOSSettingsModal}
              />
            }
          />
        )}
      {summaryData.issues?.total_issues_count > 0 &&
        !isIosOrIpadosHost &&
        !isAndroidHost &&
        renderIssues()}
      {bootstrapPackageData?.status && !isIosOrIpadosHost && !isAndroidHost && (
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
      {isPremiumTier &&
        // TODO - refactor normalizeEmptyValues pattern
        !!summaryData.maintenance_window &&
        summaryData.maintenance_window !== "---" &&
        renderMaintenanceWindow(summaryData.maintenance_window)}
    </Card>
  );
};

export default HostSummary;
