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

/** Read-only summary of the automations currently configured on a policy:
 *  the "Automations" header, a row per active automation (or an empty state),
 *  and the footer text explaining when they run. */
const PolicyAutomationsList = ({
  storedPolicy,
  currentAutomatedPolicies,
  otherAutomationType,
}: IPolicyAutomationsListProps): JSX.Element => {
  const automationRows: IAutomationDisplayRow[] = [];

  if (storedPolicy.install_software) {
    const displayedName = getDisplayedSoftwareName(
      storedPolicy.install_software.name,
      storedPolicy.install_software.display_name
    );
    automationRows.push({
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
    automationRows.push({
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
    automationRows.push({
      name: "Maintenance window",
      type: "Calendar",
      graphicName: "calendar",
      sortOrder: 2,
      sortName: "",
    });
  }

  if (storedPolicy.conditional_access_enabled) {
    automationRows.push({
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
    automationRows.push({
      name: otherName,
      type: "Other",
      graphicName: "settings",
      sortOrder: 4,
      sortName: "",
    });
  }

  automationRows.sort((a, b) => {
    if (a.sortOrder !== b.sortOrder) return a.sortOrder - b.sortOrder;
    return a.sortName.localeCompare(b.sortName);
  });

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__header`}>Automations</div>
      {automationRows.length > 0 ? (
        <div className={`${baseClass}__list`}>
          {automationRows.map((row) => (
            <div
              key={`${row.type}-${row.name}`}
              className={`${baseClass}__row`}
            >
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
      ) : (
        <div className={`${baseClass}__empty-state`}>No automations</div>
      )}
      <p className={`${baseClass}__footer-text`}>
        {storedPolicy.continuous_automations_enabled ? (
          <>
            Software and script automations run <b>every time</b> Fleet receives
            a failing response.
            <br />
            All other automations run on a host&apos;s first failure, or when a
            host&apos;s response changes from pass to fail.
          </>
        ) : (
          "Automations run on a host's first failure, or when a host's response changes from pass to fail."
        )}
      </p>
    </div>
  );
};

export default PolicyAutomationsList;
