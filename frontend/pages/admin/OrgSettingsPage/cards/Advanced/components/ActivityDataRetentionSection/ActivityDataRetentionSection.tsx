import React, { useContext, useMemo } from "react";
import SettingsSection from "pages/admin/components/SettingsSection";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import Checkbox from "components/forms/fields/Checkbox";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import { getCustomDropdownOptions } from "utilities/helpers";
import { ACTIVITY_EXPIRY_WINDOW_DROPDOWN_OPTIONS } from "utilities/constants";
import { AppContext } from "context/app";

import { IAdvancedSectionProps } from "../../Advanced";

const ActivityDataRetentionSection = ({
  onInputChange,
  formData,
}: IAdvancedSectionProps) => {
  const { isPremiumTier } = useContext(AppContext);

  const {
    disableQueryReports,
    deleteActivities,
    activityExpiryWindow,
    preserveHostActivitiesOnReenrollment,
    disableHostsActive,
    disableVulnerabilities,
  } = formData;

  const activityExpiryWindowOptions = useMemo(
    () =>
      getCustomDropdownOptions(
        ACTIVITY_EXPIRY_WINDOW_DROPDOWN_OPTIONS,
        activityExpiryWindow,
        // it's safe to assume that frequency is a number
        (frequency: number | string) => `${frequency as number} days`
      ),
    // intentionally leave activityExpiryWindow out of the dependencies, so that the custom
    // options are maintained even if the user changes the frequency in the UI
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [deleteActivities]
  );

  return (
    <SettingsSection title="Activity & data retention">
      <GitOpsModeTooltipWrapper
        position="left"
        renderChildren={(disableChildren) => (
          <Checkbox
            disabled={disableChildren}
            onChange={onInputChange}
            name="deleteActivities"
            value={deleteActivities}
            parseTarget
            labelTooltipContent={
              !disableChildren && (
                <>
                  When enabled, allows automatic cleanup of
                  <br />
                  audit logs older than the number of days
                  <br />
                  specified.{" "}
                  <em>
                    (Default: <strong>Off</strong>)
                  </em>
                </>
              )
            }
          >
            Delete activities
          </Checkbox>
        )}
      />
      {deleteActivities && (
        <GitOpsModeTooltipWrapper
          position="left"
          isInputField
          renderChildren={(disableChildren) => (
            <Dropdown
              disabled={disableChildren}
              searchable={false}
              options={activityExpiryWindowOptions}
              onChange={onInputChange}
              placeholder="Select"
              value={activityExpiryWindow}
              label="Max activity age"
              name="activityExpiryWindow"
              parseTarget
            />
          )}
        />
      )}
      <GitOpsModeTooltipWrapper
        position="left"
        renderChildren={(disableChildren) => (
          <Checkbox
            disabled={disableChildren}
            onChange={onInputChange}
            name="preserveHostActivitiesOnReenrollment"
            value={preserveHostActivitiesOnReenrollment}
            parseTarget
            labelTooltipContent={
              !disableChildren && (
                <>
                  <>
                    When enabled, preserves host activities after
                    <br />
                    a wipe and re-enrollment. Currently only
                    <br />
                    supported for company-owned (AB) Apple
                    <br />
                    hosts.{" "}
                    <strong>Delete activities &gt; Max activity age </strong>
                    <br />
                    still applies.{" "}
                    <em>
                      (Default: <b>Off</b>)
                    </em>
                  </>
                </>
              )
            }
          >
            Preserve host activities on re-enrollment
          </Checkbox>
        )}
      />
      <GitOpsModeTooltipWrapper
        position="left"
        renderChildren={(disableChildren) => (
          <Checkbox
            disabled={disableChildren}
            onChange={() =>
              onInputChange({
                name: "disableQueryReports",
                value: !disableQueryReports,
              })
            }
            name="disableQueryReports"
            value={!disableQueryReports}
            labelTooltipContent={
              !disableChildren && (
                <>
                  Disabling stored results will decrease database usage,
                  <br />
                  but will prevent you from accessing report results in
                  <br />
                  Fleet and will delete existing results. This can also be
                  <br />
                  disabled on a per-report basis.{" "}
                  <em>
                    (Default: <b>On</b>)
                  </em>
                </>
              )
            }
            helpText="Disabling this setting will delete all existing report results in Fleet."
          >
            Store report results
          </Checkbox>
        )}
      />
      <GitOpsModeTooltipWrapper
        position="left"
        renderChildren={(disableChildren) => (
          <Checkbox
            disabled={disableChildren}
            onChange={() =>
              onInputChange({
                name: "disableHostsActive",
                value: !disableHostsActive,
              })
            }
            name="disableHostsActive"
            value={!disableHostsActive}
            labelTooltipContent={
              !disableChildren && (
                <>
                  When disabled, Fleet stops collecting hourly hosts online
                  <br />
                  data used by the dashboard chart.{" "}
                  <em>
                    (Default: <strong>On</strong>)
                  </em>
                </>
              )
            }
          >
            Hosts online historical reporting
          </Checkbox>
        )}
      />
      {isPremiumTier && (
        <GitOpsModeTooltipWrapper
          position="left"
          renderChildren={(disableChildren) => (
            <Checkbox
              disabled={disableChildren}
              onChange={() =>
                onInputChange({
                  name: "disableVulnerabilities",
                  value: !disableVulnerabilities,
                })
              }
              name="disableVulnerabilities"
              value={!disableVulnerabilities}
              labelTooltipContent={
                !disableChildren && (
                  <>
                    When disabled, Fleet stops collecting historical
                    <br />
                    vulnerability exposure data used by the dashboard chart.{" "}
                    <em>
                      (Default: <strong>On</strong>)
                    </em>
                  </>
                )
              }
            >
              Vulnerability exposure historical reporting
            </Checkbox>
          )}
        />
      )}
    </SettingsSection>
  );
};

export default ActivityDataRetentionSection;
