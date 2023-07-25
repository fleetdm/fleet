import React from "react";
import { invert } from "lodash";

import { ILabel } from "interfaces/label";
import {
  formatOperatingSystemDisplayName,
  IOperatingSystemVersion,
} from "interfaces/operating_system";
import {
  FileVaultProfileStatus,
  BootstrapPackageStatus,
  IMdmSolution,
  MDM_ENROLLMENT_STATUS,
} from "interfaces/mdm";
import { IMunkiIssuesAggregate } from "interfaces/macadmins";
import { ISoftware } from "interfaces/software";
import { IPolicy } from "interfaces/policy";
import { MacSettingsStatusQueryParam } from "services/entities/hosts";

import {
  PLATFORM_LABEL_DISPLAY_NAMES,
  PolicyResponse,
} from "utilities/constants";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";

import FilterPill from "../FilterPill";
import PoliciesFilter from "../PoliciesFilter";
import { MAC_SETTINGS_FILTER_OPTIONS } from "../../HostsPageConfig";
import DiskEncryptionStatusFilter from "../DiskEncryptionStatusFilter";
import BootstrapPackageStatusFilter from "../BootstrapPackageStatusFilter/BootstrapPackageStatusFilter";

import PolicyIcon from "../../../../../../assets/images/icon-policy-fleet-black-12x12@2x.png";

const baseClass = "hosts-filter-block";

interface IHostsFilterBlockProps {
  /**
   * An object of params the the HostFilterBlock uses to render the correct
   * filter pills and dropdowns.
   *
   * TODO: improve as some of the request for this data can happen here or lower
   * in component tree.
   */
  params: {
    munkiIssueDetails: IMunkiIssuesAggregate | null;
    policyResponse: PolicyResponse;
    policyId?: any;
    policy?: IPolicy;
    macSettingsStatus?: any;
    softwareId?: any;
    mdmId?: number;
    mdmEnrollmentStatus?: any;
    lowDiskSpaceHosts?: number;
    osId?: any;
    osName?: any;
    osVersion?: any;
    munkiIssueId?: number;
    osVersions?: IOperatingSystemVersion[];
    softwareDetails: ISoftware | null;
    mdmSolutionDetails: IMdmSolution | null;
    diskEncryptionStatus?: FileVaultProfileStatus;
    bootstrapPackageStatus?: BootstrapPackageStatus;
  };
  selectedLabel?: ILabel;
  isOnlyObserver?: boolean;
  handleClearRouteParam: () => void;
  handleClearFilter: (omitParams: string[]) => void;
  onChangePoliciesFilter: (response: PolicyResponse) => void;
  onChangeDiskEncryptionStatusFilter: (
    response: FileVaultProfileStatus
  ) => void;
  onChangeBootstrapPackageStatusFilter: (
    response: BootstrapPackageStatus
  ) => void;
  onChangeMacSettingsFilter: (
    newMacSettingsStatus: MacSettingsStatusQueryParam
  ) => void;
  onClickEditLabel: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  onClickDeleteLabel: () => void;
  isSandboxMode?: boolean;
}

/**
 * Renders the filtering section of the Manage Hosts Page. This will handle rendering
 * the correct filter pills and any filter dropdowns associated with those pills.
 */
const HostsFilterBlock = ({
  params: {
    policyId,
    macSettingsStatus,
    softwareId,
    mdmId,
    mdmEnrollmentStatus,
    lowDiskSpaceHosts,
    osId,
    osName,
    osVersion,
    munkiIssueId,
    munkiIssueDetails,
    policyResponse,
    osVersions,
    softwareDetails,
    policy,
    mdmSolutionDetails,
    diskEncryptionStatus,
    bootstrapPackageStatus,
  },
  selectedLabel,
  isOnlyObserver,
  handleClearRouteParam,
  handleClearFilter,
  onChangePoliciesFilter,
  onChangeDiskEncryptionStatusFilter,
  onChangeBootstrapPackageStatusFilter,
  onChangeMacSettingsFilter,
  onClickEditLabel,
  onClickDeleteLabel,
  isSandboxMode = false,
}: IHostsFilterBlockProps) => {
  const renderLabelFilterPill = () => {
    if (selectedLabel) {
      const { description, display_text, label_type } = selectedLabel;
      const pillLabel =
        PLATFORM_LABEL_DISPLAY_NAMES[display_text] ?? display_text;

      return (
        <>
          <FilterPill
            label={pillLabel}
            tooltipDescription={description}
            onClear={handleClearRouteParam}
          />
          {label_type !== "builtin" && !isOnlyObserver && (
            <>
              <Button onClick={onClickEditLabel} variant="small-icon">
                <Icon name="pencil" size="small" />
              </Button>
              <Button onClick={onClickDeleteLabel} variant="small-icon">
                <Icon name="trash" size="small" />
              </Button>
            </>
          )}
        </>
      );
    }

    return null;
  };

  const renderOSFilterBlock = () => {
    if (!osId && !(osName && osVersion)) return null;

    let os: IOperatingSystemVersion | undefined;
    if (osId) {
      os = osVersions?.find((v) => v.os_id === osId);
    } else if (osName && osVersion) {
      const name: string = osName;
      const vers: string = osVersion;

      os = osVersions?.find(
        ({ name_only, version }) =>
          name_only.toLowerCase() === name.toLowerCase() &&
          version.toLowerCase() === vers.toLowerCase()
      );
    }
    if (!os) return null;

    const { name, name_only, version } = os;
    // TODO: Move formatOperatingSystemDisplayName into utils file
    const label = formatOperatingSystemDisplayName(
      name_only || version
        ? `${name_only || ""} ${version || ""}`
        : `${name || ""}`
    );
    const TooltipDescription = (
      <span>
        Hosts with {formatOperatingSystemDisplayName(name_only || name)},
        <br />
        {version && `${version} installed`}
      </span>
    );

    return (
      <FilterPill
        label={label}
        tooltipDescription={TooltipDescription}
        onClear={() => handleClearFilter(["os_id", "os_name", "os_version"])}
      />
    );
  };

  // NOTE: good example of filter dropdown with pill
  const renderPoliciesFilterBlock = () => (
    <>
      <PoliciesFilter
        policyResponse={policyResponse}
        onChange={onChangePoliciesFilter}
      />
      <FilterPill
        icon={PolicyIcon}
        label={policy?.name ?? "..."}
        onClear={() => handleClearFilter(["policy_id", "policy_response"])}
        className={`${baseClass}__policies-filter-pill`}
      />
    </>
  );

  const renderMacSettingsStatusFilterBlock = () => {
    const label = "macOS settings";
    return (
      <>
        <Dropdown
          value={macSettingsStatus}
          className={`${baseClass}__macsettings-dropdown`}
          options={MAC_SETTINGS_FILTER_OPTIONS}
          onChange={onChangeMacSettingsFilter}
        />
        <FilterPill
          label={label}
          onClear={() => handleClearFilter(["macos_settings"])}
        />
      </>
    );
  };

  const renderSoftwareFilterBlock = () => {
    if (!softwareDetails) return null;

    const { name, version } = softwareDetails;
    const label = `${name || "Unknown software"} ${version || ""}`;

    const TooltipDescription = (
      <span>
        Hosts with {name || "Unknown software"},
        <br />
        {version || "version unknown"} installed
      </span>
    );

    return (
      <FilterPill
        label={label}
        onClear={() => handleClearFilter(["software_id"])}
        tooltipDescription={TooltipDescription}
      />
    );
  };

  const renderMDMSolutionFilterBlock = () => {
    if (!mdmSolutionDetails) return null;

    const { name, server_url } = mdmSolutionDetails;
    const label = name ? `${name} ${server_url}` : `${server_url}`;

    const TooltipDescription = (
      <span>
        Host enrolled
        {name !== "Unknown" && ` to ${name}`}
        <br /> at {server_url}
      </span>
    );

    return (
      <FilterPill
        label={label}
        tooltipDescription={TooltipDescription}
        onClear={() => handleClearFilter(["mdm_id"])}
      />
    );
  };

  const renderMDMEnrollmentFilterBlock = () => {
    if (!mdmEnrollmentStatus) return null;

    const label = `MDM status: ${
      invert(MDM_ENROLLMENT_STATUS)[mdmEnrollmentStatus]
    }`;

    // More narrow tooltip than other MDM tooltip
    const MDM_STATUS_PILL_TOOLTIP: Record<string, JSX.Element> = {
      automatic: (
        <span>
          MDM was turned on <br />
          automatically using Apple <br />
          Automated Device <br />
          Enrollment (DEP), <br />
          Windows Autopilot, or <br />
          Windows Azure AD Join. <br />
          Administrators can block <br />
          device users from turning
          <br /> MDM off.
        </span>
      ),
      manual: (
        <span>
          MDM was turned on <br />
          manually. Device users <br />
          can turn MDM off.
        </span>
      ),
      unenrolled: (
        <span>
          Hosts with MDM off <br />
          don&apos;t receive macOS <br />
          settings and macOS <br />
          update encouragement.
        </span>
      ),
      pending: (
        <span>
          Hosts ordered using Apple <br />
          Business Manager (ABM). <br />
          They will automatically enroll <br />
          to Fleet and turn on MDM <br />
          when they&apos;re unboxed.
        </span>
      ),
    };

    return (
      <FilterPill
        label={label}
        tooltipDescription={MDM_STATUS_PILL_TOOLTIP[mdmEnrollmentStatus]}
        onClear={() => handleClearFilter(["mdm_enrollment_status"])}
      />
    );
  };

  const renderMunkiIssueFilterBlock = () => {
    if (munkiIssueDetails) {
      return (
        <FilterPill
          label={munkiIssueDetails.name}
          tooltipDescription={
            <span>
              Hosts that reported this Munki issue <br />
              the last time Munki ran on each host.
            </span>
          }
          onClear={() => handleClearFilter(["munki_issue_id"])}
        />
      );
    }
    return null;
  };

  const renderLowDiskSpaceFilterBlock = () => {
    const TooltipDescription = (
      <span>
        Hosts that have {lowDiskSpaceHosts} GB or less <br />
        disk space available.
      </span>
    );

    return (
      <FilterPill
        label="Low disk space"
        tooltipDescription={TooltipDescription}
        premiumFeatureTooltipDelayHide={1000}
        onClear={() => handleClearFilter(["low_disk_space"])}
        isSandboxMode={isSandboxMode}
        sandboxPremiumOnlyIcon
      />
    );
  };

  const renderDiskEncryptionStatusBlock = () => {
    if (!diskEncryptionStatus) return null;

    return (
      <>
        <DiskEncryptionStatusFilter
          diskEncryptionStatus={diskEncryptionStatus}
          onChange={onChangeDiskEncryptionStatusFilter}
        />
        <FilterPill
          label="macOS settings: Disk encryption"
          onClear={() => handleClearFilter(["macos_settings_disk_encryption"])}
        />
      </>
    );
  };

  const renderBootstrapPackageStatusBlock = () => {
    if (!bootstrapPackageStatus) return null;

    return (
      <>
        <BootstrapPackageStatusFilter
          bootstrapPackageStatus={bootstrapPackageStatus}
          onChange={onChangeBootstrapPackageStatusFilter}
        />
        <FilterPill
          label="macOS settings: bootstrap package"
          onClear={() => handleClearFilter(["bootstrap_package"])}
        />
      </>
    );
  };

  const showSelectedLabel = selectedLabel && selectedLabel.type !== "all";

  if (
    showSelectedLabel ||
    policyId ||
    macSettingsStatus ||
    softwareId ||
    mdmId ||
    mdmEnrollmentStatus ||
    lowDiskSpaceHosts ||
    osId ||
    (osName && osVersion) ||
    munkiIssueId ||
    diskEncryptionStatus ||
    bootstrapPackageStatus
  ) {
    const renderFilterPill = () => {
      switch (true) {
        // backend allows for pill combos (label + low disk space) OR
        // (label + mdm solution) OR (label + mdm enrollment status)
        case showSelectedLabel && !!lowDiskSpaceHosts:
          return (
            <>
              {renderLabelFilterPill()} {renderLowDiskSpaceFilterBlock()}
            </>
          );
        case showSelectedLabel && !!mdmId:
          return (
            <>
              {renderLabelFilterPill()} {renderMDMSolutionFilterBlock()}
            </>
          );

        case showSelectedLabel && !!mdmEnrollmentStatus:
          return (
            <>
              {renderLabelFilterPill()} {renderMDMEnrollmentFilterBlock()}
            </>
          );
        case showSelectedLabel:
          return renderLabelFilterPill();
        case !!policyId:
          return renderPoliciesFilterBlock();
        case !!macSettingsStatus:
          return renderMacSettingsStatusFilterBlock();
        case !!softwareId:
          return renderSoftwareFilterBlock();
        case !!mdmId:
          return renderMDMSolutionFilterBlock();
        case !!mdmEnrollmentStatus:
          return renderMDMEnrollmentFilterBlock();
        case !!osId || (!!osName && !!osVersion):
          return renderOSFilterBlock();
        case !!munkiIssueId:
          return renderMunkiIssueFilterBlock();
        case !!lowDiskSpaceHosts:
          return renderLowDiskSpaceFilterBlock();
        case !!diskEncryptionStatus:
          return renderDiskEncryptionStatusBlock();
        case !!bootstrapPackageStatus:
          return renderBootstrapPackageStatusBlock();
        default:
          return null;
      }
    };

    return (
      <div className={`${baseClass}__labels-active-filter-wrap`}>
        {renderFilterPill()}
      </div>
    );
  }

  return null;
};

export default HostsFilterBlock;
