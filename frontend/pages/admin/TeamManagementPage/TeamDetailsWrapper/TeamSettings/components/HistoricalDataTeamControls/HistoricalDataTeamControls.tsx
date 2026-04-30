import React from "react";

import { IInputFieldParseTarget } from "interfaces/form_field";

import SectionHeader from "components/SectionHeader";
import Checkbox from "components/forms/fields/Checkbox";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";

interface IHistoricalDataTeamControlsProps {
  disableHostsActive: boolean;
  disableVulnerabilities: boolean;
  globalHostsActiveDisabled: boolean;
  globalVulnerabilitiesDisabled: boolean;
  onChange: (parsed: IInputFieldParseTarget) => void;
}

const HistoricalDataTeamControls = ({
  disableHostsActive,
  disableVulnerabilities,
  globalHostsActiveDisabled,
  globalVulnerabilitiesDisabled,
  onChange,
}: IHistoricalDataTeamControlsProps): JSX.Element => {
  return (
    <>
      <SectionHeader title="Activity & data retention" />
      <GitOpsModeTooltipWrapper
        renderChildren={(disableChildren) => (
          <Checkbox
            disabled={disableChildren || globalHostsActiveDisabled}
            onChange={onChange}
            name="disableHostsActive"
            value={disableHostsActive}
            parseTarget
            labelTooltipContent={
              globalHostsActiveDisabled
                ? "Disabled globally"
                : !disableChildren && (
                    <>
                      When enabled, Fleet stops collecting hosts-active
                      <br />
                      data for this fleet&apos;s contribution to the
                      <br />
                      dashboard chart.
                    </>
                  )
            }
          >
            Disable hosts active
          </Checkbox>
        )}
      />
      <GitOpsModeTooltipWrapper
        renderChildren={(disableChildren) => (
          <Checkbox
            disabled={disableChildren || globalVulnerabilitiesDisabled}
            onChange={onChange}
            name="disableVulnerabilities"
            value={disableVulnerabilities}
            parseTarget
            labelTooltipContent={
              globalVulnerabilitiesDisabled
                ? "Disabled globally"
                : !disableChildren && (
                    <>
                      When enabled, Fleet stops collecting vulnerability
                      <br />
                      exposure data for this fleet&apos;s contribution
                      <br />
                      to the dashboard chart.
                    </>
                  )
            }
          >
            Disable vulnerabilities
          </Checkbox>
        )}
      />
    </>
  );
};

export default HistoricalDataTeamControls;
