import React from "react";
import { Link } from "react-router";

import { IPolicy, OtherAutomationType } from "interfaces/policy";
import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

import Graphic from "components/Graphic";
import { GraphicNames } from "components/graphics";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";
import { getDisplayedSoftwareName } from "pages/SoftwarePage/helpers";

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
  sortOrder: number;
  sortName: string;
}

interface IPolicyAutomationsListProps {
  storedPolicy: IPolicy;
  currentAutomatedPolicies: number[];
  otherAutomationType?: OtherAutomationType;
}

export const mapAutomationRows = (
  storedPolicy: IPolicy,
  currentAutomatedPolicies: number[],
  otherAutomationType?: OtherAutomationType
): IAutomationDisplayRow[] => {
  const rows: IAutomationDisplayRow[] = [];

  if (storedPolicy.install_software) {
    const displayedName = getDisplayedSoftwareName(
      storedPolicy.install_software.name,
      storedPolicy.install_software.display_name
    );
    rows.push({
      name: displayedName,
      iconName: storedPolicy.install_software.name,
      type: "Software",
      isSoftware: true,
      iconUrl: storedPolicy.install_software.icon_url,
      link: getPathWithQueryParams(
        PATHS.SOFTWARE_TITLE_DETAILS(
          storedPolicy.install_software.software_title_id.toString()
        ),
        { fleet_id: storedPolicy.team_id }
      ),
      sortOrder: 0,
      sortName: displayedName.toLowerCase(),
    });
  }

  if (storedPolicy.run_script) {
    rows.push({
      name: storedPolicy.run_script.name,
      type: "Script",
      graphicName: storedPolicy.run_script.name.endsWith(".sh")
        ? "file-sh"
        : "file-ps1",
      sortOrder: 1,
      sortName: storedPolicy.run_script.name.toLowerCase(),
    });
  }

  if (storedPolicy.calendar_events_enabled) {
    rows.push({
      name: "Maintenance window",
      type: "Calendar",
      graphicName: "calendar",
      sortOrder: 2,
      sortName: "",
    });
  }

  if (storedPolicy.conditional_access_enabled) {
    rows.push({
      name: "Block single sign-on",
      type: "Conditional access",
      graphicName: "lock",
      sortOrder: 3,
      sortName: "",
    });
  }

  if (currentAutomatedPolicies.includes(storedPolicy.id)) {
    const otherName = otherAutomationType
      ? OTHER_AUTOMATION_NAMES[otherAutomationType]
      : "Webhook or ticket";
    rows.push({
      name: otherName,
      type: "Other",
      graphicName: "settings",
      sortOrder: 4,
      sortName: "",
    });
  }

  rows.sort((a, b) => {
    if (a.sortOrder !== b.sortOrder) return a.sortOrder - b.sortOrder;
    return a.sortName.localeCompare(b.sortName);
  });

  return rows;
};

/** Read-only list of the automations currently configured on a policy: one row
 *  per active automation, or an empty state when there are none. */
const PolicyAutomationsList = ({
  storedPolicy,
  currentAutomatedPolicies,
  otherAutomationType,
}: IPolicyAutomationsListProps): JSX.Element => {
  const automationRows = mapAutomationRows(
    storedPolicy,
    currentAutomatedPolicies,
    otherAutomationType
  );

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
                    row.graphicName === "file-ps1"
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
