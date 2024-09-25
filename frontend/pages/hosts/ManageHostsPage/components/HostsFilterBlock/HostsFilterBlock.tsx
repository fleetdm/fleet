import React from "react";
import { invert } from "lodash";

import { ILabel } from "interfaces/label";
import {
  formatOperatingSystemDisplayName,
  IOperatingSystemVersion,
} from "interfaces/operating_system";
import {
  DiskEncryptionStatus,
  BootstrapPackageStatus,
  IMdmSolution,
  MDM_ENROLLMENT_STATUS,
  MdmProfileStatus,
} from "interfaces/mdm";
import { IMunkiIssuesAggregate } from "interfaces/macadmins";
import { IPolicy } from "interfaces/policy";
import { SoftwareAggregateStatus } from "interfaces/software";

import {
  HOSTS_QUERY_PARAMS,
  MacSettingsStatusQueryParam,
} from "services/entities/hosts";

import {
  PLATFORM_LABEL_DISPLAY_NAMES,
  PLATFORM_TYPE_ICONS,
  isPlatformLabelNameFromAPI,
  PolicyResponse,
} from "utilities/constants";

// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Button from "components/buttons/Button";
import Icon from "components/Icon/Icon";

import FilterPill from "../FilterPill";
import PoliciesFilter from "../PoliciesFilter";
import { OS_SETTINGS_FILTER_OPTIONS } from "../../HostsPageConfig";
import DiskEncryptionStatusFilter from "../DiskEncryptionStatusFilter";
import BootstrapPackageStatusFilter from "../BootstrapPackageStatusFilter/BootstrapPackageStatusFilter";

const baseClass = "hosts-filter-block";

type PlatformLabelNameFromAPI = keyof typeof PLATFORM_TYPE_ICONS;

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
    softwareId?: number;
    softwareTitleId?: number;
    softwareVersionId?: number;
    mdmId?: number;
    mdmEnrollmentStatus?: any;
    lowDiskSpaceHosts?: number;
    osVersionId?: string;
    osName?: string;
    osVersion?: string;
    vulnerability?: string;
    munkiIssueId?: number;
    osVersions?: IOperatingSystemVersion[];
    softwareDetails: { name: string; version?: string } | null;
    mdmSolutionDetails: IMdmSolution | null;
    osSettingsStatus?: MdmProfileStatus;
    diskEncryptionStatus?: DiskEncryptionStatus;
    bootstrapPackageStatus?: BootstrapPackageStatus;
    softwareStatus?: SoftwareAggregateStatus;
  };
  selectedLabel?: ILabel;
  isOnlyObserver?: boolean;
  handleClearRouteParam: () => void;
  handleClearFilter: (omitParams: string[]) => void;
  onChangePoliciesFilter: (response: PolicyResponse) => void;
  onChangeOsSettingsFilter: (newStatus: MdmProfileStatus) => void;
  onChangeDiskEncryptionStatusFilter: (response: DiskEncryptionStatus) => void;
  onChangeBootstrapPackageStatusFilter: (
    response: BootstrapPackageStatus
  ) => void;
  onChangeMacSettingsFilter: (
    newMacSettingsStatus: MacSettingsStatusQueryParam
  ) => void;
  onChangeSoftwareInstallStatusFilter: (
    newStatus: SoftwareAggregateStatus
  ) => void;
  onClickEditLabel: (evt: React.MouseEvent<HTMLButtonElement>) => void;
  onClickDeleteLabel: () => void;
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
    softwareTitleId,
    softwareVersionId,
    mdmId,
    mdmEnrollmentStatus,
    lowDiskSpaceHosts,
    osVersionId,
    osName,
    osVersion,
    vulnerability,
    munkiIssueId,
    munkiIssueDetails,
    policyResponse,
    osVersions,
    softwareDetails,
    policy,
    mdmSolutionDetails,
    osSettingsStatus,
    diskEncryptionStatus,
    bootstrapPackageStatus,
    softwareStatus,
  },
  selectedLabel,
  isOnlyObserver,
  handleClearRouteParam,
  handleClearFilter,
  onChangePoliciesFilter,
  onChangeOsSettingsFilter,
  onChangeDiskEncryptionStatusFilter,
  onChangeBootstrapPackageStatusFilter,
  onChangeMacSettingsFilter,
  onChangeSoftwareInstallStatusFilter,
  onClickEditLabel,
  onClickDeleteLabel,
}: IHostsFilterBlockProps) => {
  const renderLabelFilterPill = () => {
    if (selectedLabel) {
      const { description, display_text, label_type } = selectedLabel;
      const pillLabel =
        (isPlatformLabelNameFromAPI(display_text) &&
          PLATFORM_LABEL_DISPLAY_NAMES[display_text]) ||
        display_text;

      // Hide built-in labels supported in label dropdown
      if (
        label_type === "builtin" &&
        Object.keys(PLATFORM_TYPE_ICONS).includes(
          display_text as PlatformLabelNameFromAPI
        )
      ) {
        return <></>;
      }

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
    let os: IOperatingSystemVersion | undefined;
    if (osVersionId) {
      os = osVersions?.find(
        (v) => v.os_version_id === parseInt(osVersionId, 10)
      );
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
        onClear={() =>
          handleClearFilter(["os_version_id", "os_name", "os_version"])
        }
      />
    );
  };

  const renderVulnerabilityFilterBlock = () => {
    if (!vulnerability) return null;

    return (
      <FilterPill
        label={vulnerability}
        tooltipDescription={<span>Hosts affected by the specified CVE.</span>}
        onClear={() => handleClearFilter(["vulnerability"])}
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
        icon="policy"
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
          options={OS_SETTINGS_FILTER_OPTIONS}
          onChange={onChangeMacSettingsFilter}
          searchable={false}
          iconName="filter-alt"
        />
        <FilterPill
          label={label}
          onClear={() => handleClearFilter(["macos_settings"])}
        />
      </>
    );
  };

  const renderSoftwareFilterBlock = (additionalClearParams?: string[]) => {
    if (!softwareDetails) return null;

    const { name, version } = softwareDetails;
    let label = name;
    if (version) {
      label += ` ${version}`;
    }
    label = label.trim() || "Unknown software";

    const clearParams = [
      "software_id",
      "software_version_id",
      "software_title_id",
    ];

    if (additionalClearParams?.length) {
      clearParams.push(...additionalClearParams);
    }

    // const TooltipDescription = (
    //   <span>
    //     Hosts with {name || "Unknown software"},
    //     <br />
    //     {version || "version unknown"} installed
    //   </span>
    // );

    return (
      <FilterPill
        label={label}
        onClear={() => handleClearFilter(clearParams)}
        // tooltipDescription={TooltipDescription}
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
    const MDM_STATUS_PILL_TOOLTIP: Record<string, React.ReactNode> = {
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
      unenrolled: undefined, // no tooltip specified
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
        onClear={() => handleClearFilter(["low_disk_space"])}
      />
    );
  };

  const renderOsSettingsBlock = () => {
    const label = "OS settings";
    return (
      <>
        <Dropdown
          value={osSettingsStatus}
          className={`${baseClass}__os_settings-dropdown`}
          options={OS_SETTINGS_FILTER_OPTIONS}
          onChange={onChangeOsSettingsFilter}
          searchable={false}
          iconName="filter-alt"
        />
        <FilterPill
          label={label}
          onClear={() => handleClearFilter([HOSTS_QUERY_PARAMS.OS_SETTINGS])}
        />
      </>
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
          label="OS settings: Disk encryption"
          onClear={() =>
            handleClearFilter([HOSTS_QUERY_PARAMS.DISK_ENCRYPTION])
          }
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

  const renderSoftwareInstallStatusBlock = () => {
    const OPTIONS = [
      { value: "installed", label: "Installed" },
      { value: "failed", label: "Failed" },
      { value: "pending", label: "Pending" },
    ];

    return (
      <>
        <Dropdown
          value={softwareStatus}
          className={`${baseClass}__sw-install-status-dropdown`}
          options={OPTIONS}
          searchable={false}
          onChange={onChangeSoftwareInstallStatusFilter}
          iconName="filter-alt"
        />
        {renderSoftwareFilterBlock([HOSTS_QUERY_PARAMS.SOFTWARE_STATUS])}
      </>
    );
  };

  const showSelectedLabel = selectedLabel && selectedLabel.type !== "all";

  if (
    showSelectedLabel ||
    policyId ||
    macSettingsStatus ||
    softwareId ||
    softwareTitleId ||
    softwareVersionId ||
    softwareStatus ||
    mdmId ||
    mdmEnrollmentStatus ||
    lowDiskSpaceHosts ||
    osVersionId ||
    (osName && osVersion) ||
    munkiIssueId ||
    osSettingsStatus ||
    diskEncryptionStatus ||
    bootstrapPackageStatus ||
    vulnerability
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
        case !!softwareStatus:
          return renderSoftwareInstallStatusBlock();
        case !!softwareId || !!softwareVersionId || !!softwareTitleId:
          return renderSoftwareFilterBlock();
        case !!mdmId:
          return renderMDMSolutionFilterBlock();
        case !!mdmEnrollmentStatus:
          return renderMDMEnrollmentFilterBlock();
        case !!osVersionId || (!!osName && !!osVersion):
          return renderOSFilterBlock();
        case !!vulnerability:
          return renderVulnerabilityFilterBlock();
        case !!munkiIssueId:
          return renderMunkiIssueFilterBlock();
        case !!lowDiskSpaceHosts:
          return renderLowDiskSpaceFilterBlock();
        case !!osSettingsStatus:
          return renderOsSettingsBlock();
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
