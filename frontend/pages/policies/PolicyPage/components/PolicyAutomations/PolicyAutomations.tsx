import React from "react";
import { Link } from "react-router";

import { IPolicy } from "interfaces/policy";
import { IconNames } from "components/icons";
import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";
import Button from "components/buttons/Button";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Icon from "components/Icon/Icon";

const baseClass = "policy-automations";

interface IPolicyAutomationsProps {
  storedPolicy: IPolicy;
  currentAutomatedPolicies: number[];
  onAddAutomation: () => void;
  isAddingAutomation: boolean;
  gitOpsModeEnabled: boolean;
}

interface IAutomationRow {
  name: string;
  type: string;
  iconName: IconNames;
  link?: string;
  sortOrder: number;
  sortName: string;
}

const PolicyAutomations = ({
  storedPolicy,
  currentAutomatedPolicies,
  onAddAutomation,
  isAddingAutomation,
  gitOpsModeEnabled,
}: IPolicyAutomationsProps): JSX.Element => {
  const isPatchPolicy = storedPolicy.type === "patch";
  const hasPatchSoftware = !!storedPolicy.patch_software;
  const hasSoftwareAutomation = !!storedPolicy.install_software;
  const showCtaCard =
    isPatchPolicy && hasPatchSoftware && !hasSoftwareAutomation;

  const automationRows: IAutomationRow[] = [];

  if (storedPolicy.install_software) {
    automationRows.push({
      name: storedPolicy.install_software.name,
      type: "Software",
      iconName: "install",
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
      iconName: "text",
      sortOrder: 1,
      sortName: storedPolicy.run_script.name.toLowerCase(),
    });
  }

  if (storedPolicy.calendar_events_enabled) {
    automationRows.push({
      name: "Maintenance window",
      type: "Calendar",
      iconName: "calendar",
      sortOrder: 2,
      sortName: "",
    });
  }

  if (storedPolicy.conditional_access_enabled) {
    automationRows.push({
      name: "Block single sign-on",
      type: "Conditional access",
      iconName: "disable",
      sortOrder: 3,
      sortName: "",
    });
  }

  if (currentAutomatedPolicies.includes(storedPolicy.id)) {
    automationRows.push({
      name: "Create ticket or send webhook",
      type: "Other",
      iconName: "external-link",
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
      {automationRows.length > 0 && (
        <div className={`${baseClass}__list`}>
          <div className={`${baseClass}__list-header`}>Automations</div>
          {automationRows.map((row) => (
            <div
              key={`${row.type}-${row.name}`}
              className={`${baseClass}__row`}
            >
              <span className={`${baseClass}__row-name`}>
                <Icon
                  name={row.iconName}
                  size="small"
                  className={`${baseClass}__row-icon`}
                />
                {row.link ? <Link to={row.link}>{row.name}</Link> : row.name}
              </span>
              <span className={`${baseClass}__row-type`}>{row.type}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default PolicyAutomations;
