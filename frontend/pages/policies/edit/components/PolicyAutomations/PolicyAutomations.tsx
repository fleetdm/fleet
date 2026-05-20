import React from "react";
import { Link } from "react-router";

import { IPolicy, OtherAutomationType } from "interfaces/policy";
import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

import Button from "components/buttons/Button";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Graphic from "components/Graphic";
import { GraphicNames } from "components/graphics";
import Icon from "components/Icon";
import SoftwareIcon from "pages/SoftwarePage/components/icons/SoftwareIcon";

const baseClass = "policy-automations";

interface IPolicyAutomationsProps {
  storedPolicy: IPolicy;
  currentAutomatedPolicies: number[];
  /** Some users only have access to read-only view */
  canEditPolicy: boolean;
  onAddAutomation: () => void;
  isAddingAutomation?: boolean;
  otherAutomationType?: OtherAutomationType;
}

const OTHER_AUTOMATION_NAMES: Record<OtherAutomationType, string> = {
  webhook: "Webhook",
  ticket: "Ticket",
};

interface IAutomationRow {
  name: string;
  type: string;
  graphicName?: GraphicNames;
  isSoftware?: boolean;
  link?: string;
  sortOrder: number;
  sortName: string;
}

const PolicyAutomations = ({
  storedPolicy,
  currentAutomatedPolicies,
  canEditPolicy,
  onAddAutomation,
  isAddingAutomation,
  otherAutomationType,
}: IPolicyAutomationsProps): JSX.Element => {
  const isPatchPolicy = storedPolicy.type === "patch";
  const hasPatchSoftware = !!storedPolicy.patch_software;
  const hasSoftwareAutomation = !!storedPolicy.install_software;
  const showCtaCard =
    isPatchPolicy &&
    hasPatchSoftware &&
    !hasSoftwareAutomation &&
    canEditPolicy;

  const automationRows: IAutomationRow[] = [];

  if (storedPolicy.install_software) {
    automationRows.push({
      name: storedPolicy.install_software.name,
      type: "Software",
      isSoftware: true,
      link: getPathWithQueryParams(
        PATHS.SOFTWARE_TITLE_DETAILS(
          storedPolicy.install_software.software_title_id.toString()
        ),
        { fleet_id: storedPolicy.team_id }
      ),
      sortOrder: 0,
      sortName: storedPolicy.install_software.name.toLowerCase(),
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

  const patchSoftwareName =
    storedPolicy.patch_software?.display_name ||
    storedPolicy.patch_software?.name ||
    "";

  return (
    <div className={baseClass}>
      {showCtaCard && (
        <div className={`${baseClass}__cta-card`}>
          <span className={`${baseClass}__cta-label`}>
            Automatically patch {patchSoftwareName}
          </span>
          <GitOpsModeTooltipWrapper
            position="top"
            renderChildren={(disableChildren) => (
              <Button
                onClick={onAddAutomation}
                variant="text-icon"
                disabled={disableChildren || isAddingAutomation}
              >
                {isAddingAutomation ? (
                  "Adding..."
                ) : (
                  <>
                    <Icon name="plus" /> Add automation
                  </>
                )}
              </Button>
            )}
          />
        </div>
      )}
      <div className={`${baseClass}__list-label`}>Automations</div>
      {automationRows.length > 0 ? (
        <div className={`${baseClass}__list`}>
          {automationRows.map((row) => (
            <div
              key={`${row.type}-${row.name}`}
              className={`${baseClass}__row`}
            >
              <div className={`${baseClass}__row-name`}>
                {row.isSoftware ? (
                  <SoftwareIcon name={row.name} size="small" />
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

export default PolicyAutomations;
