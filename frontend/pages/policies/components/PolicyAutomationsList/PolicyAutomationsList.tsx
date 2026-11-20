import React from "react";
import { Link } from "react-router";

import Graphic from "components/Graphic";
import { GraphicNames } from "components/graphics";
import { IPolicy, OtherAutomationType } from "interfaces/policy";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";
import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

const baseClass = "policy-automations-list";

const OTHER_AUTOMATION_NAMES: Record<OtherAutomationType, string> = {
  webhook: "Webhook",
  ticket: "Ticket",
};

interface IAutomationDisplayRow {
  name: string;
  /** Raw software name forwarded to SoftwareIcon for name-based fallback
   * matching; display-name overrides break the known-icon lookup. */
  iconName?: string;
  type: string;
  graphicName?: GraphicNames;
  isSoftware?: boolean;
  iconUrl?: string | null;
  link?: string;
}

interface IPolicyAutomationsListProps {
  storedPolicy: IPolicy;
  otherAutomationType?: OtherAutomationType;
}

/** The row for the policy's automation, or no rows when it has none. */
export const mapAutomationRows = (
  storedPolicy: IPolicy,
  otherAutomationType?: OtherAutomationType
): IAutomationDisplayRow[] => {
  const {
    install_software: software,
    run_script: script,
    resend_configuration_profile: profile,
  } = storedPolicy;

  switch (storedPolicy.automation_type) {
    case "software": {
      if (!software) return [];
      const displayedName = getDisplayedSoftwareName(
        software.name,
        software.display_name
      );
      return [
        {
          name: displayedName,
          iconName: software.name,
          type: "Software",
          isSoftware: true,
          iconUrl: software.icon_url,
          link: getPathWithQueryParams(
            PATHS.SOFTWARE_TITLE_DETAILS(software.software_title_id.toString()),
            { fleet_id: storedPolicy.fleet_id }
          ),
        },
      ];
    }
    case "script":
      return script
        ? [
            {
              name: script.name,
              type: "Script",
              graphicName: script.name.endsWith(".sh") ? "file-sh" : "file-ps1",
            },
          ]
        : [];
    case "configuration_profile":
      return profile
        ? [
            {
              name: profile.name,
              type: "Profile",
              graphicName: "file-configuration-profile",
            },
          ]
        : [];
    case "calendar":
      return [
        {
          name: "Maintenance window",
          type: "Calendar",
          graphicName: "calendar",
        },
      ];
    case "conditional_access":
      return [
        {
          name: "Block single sign-on",
          type: "Conditional access",
          graphicName: "lock",
        },
      ];
    case "webhook_or_ticket":
      return [
        {
          name: otherAutomationType
            ? OTHER_AUTOMATION_NAMES[otherAutomationType]
            : "Webhook or ticket",
          type: "Other",
          graphicName: "settings",
        },
      ];
    default:
      return [];
  }
};

/** Read-only list of the automations currently configured on a policy: one row
 *  per active automation, or an empty state when there are none. */
const PolicyAutomationsList = ({
  storedPolicy,
  otherAutomationType,
}: IPolicyAutomationsListProps): JSX.Element => {
  const automationRows = mapAutomationRows(storedPolicy, otherAutomationType);

  if (automationRows.length === 0) {
    return <div className={`${baseClass}__empty-state`}>No automations</div>;
  }

  return (
    <div className={baseClass}>
      {automationRows.map((row) => (
        <div key={`${row.type}-${row.name}`} className={`${baseClass}__row`}>
          <div className={`${baseClass}__row-name`}>
            {row.isSoftware ? (
              <SoftwareIcon
                name={row.iconName ?? row.name}
                url={row.iconUrl}
                size="small"
              />
            ) : (
              row.graphicName && (
                <Graphic
                  name={row.graphicName}
                  key={`${row.graphicName}-graphic`}
                  className={`${baseClass}__row-graphic ${
                    row.graphicName === "file-sh" ||
                    row.graphicName === "file-ps1" ||
                    row.graphicName === "file-configuration-profile"
                      ? "scale-40-24"
                      : ""
                  }`}
                />
              )
            )}
            {row.link ? <Link to={row.link}>{row.name}</Link> : row.name}
          </div>
        </div>
      ))}
    </div>
  );
};

export default PolicyAutomationsList;
