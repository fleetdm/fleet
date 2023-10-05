import Checkbox from "components/forms/fields/Checkbox";
import Icon from "components/Icon";
import InfoBanner from "components/InfoBanner";
import TooltipWrapper from "components/TooltipWrapper";
import { IConfig } from "interfaces/config";
import { QueryLoggingOption } from "interfaces/schedulable_query";
import React, { useState } from "react";
import { Link } from "react-router";

const baseClass = "discard-data-option";

interface IDiscardDataOption {
  appConfig?: IConfig;
  selectedLoggingType: QueryLoggingOption;
  discardData: boolean;
  setDiscardData: (value: boolean) => void;
  breakHelpText?: boolean;
}

const DiscardDataOption = ({
  appConfig,
  selectedLoggingType,
  discardData,
  setDiscardData,
  breakHelpText = false,
}: IDiscardDataOption) => {
  const [forceEditDiscardData, setForceEditDiscardData] = useState(false);
  const disable =
    appConfig?.server_settings.query_reports_disabled && !forceEditDiscardData;

  return (
    <div className={baseClass}>
      {["differential", "differential_ignore_removals"].includes(
        selectedLoggingType
      ) && (
        <InfoBanner color="purple-bold-border">
          <>
            The <b>Discard data</b> setting is ignored when differential logging
            is enabled. This <br />
            query&apos;s results will not be saved in Fleet.
          </>
        </InfoBanner>
      )}
      <Checkbox
        name="discardData"
        onChange={setDiscardData}
        value={discardData}
        wrapperClassName={
          disable ? `${baseClass}__disabled-discard-data-checkbox` : ""
        }
      >
        <b>Discard data</b>
      </Checkbox>
      <div className="help-text">
        {disable ? (
          <>
            This setting is ignored because query reports in Fleet have been{" "}
            <TooltipWrapper
              // TODO - use JSX once new tooltipwrapper is merged
              tipContent={
                "A Fleet administrator can enable query reports under <br />\
                  <b>Organization settings > Advanced options > Disable  query reports</b>."
              }
              position="bottom"
            >
              {"globally disabled."}
            </TooltipWrapper>{" "}
            <Link
              to={""}
              onClick={(e: React.MouseEvent) => {
                e.preventDefault();
                setForceEditDiscardData(true);
              }}
              className={`${baseClass}__edit-anyway`}
            >
              <>
                Edit anyway
                <Icon
                  name="chevron"
                  direction="right"
                  color="core-fleet-blue"
                  size="small"
                />
              </>
            </Link>
          </>
        ) : (
          <>
            The most recent results for each host will not be available in
            Fleet.
            {breakHelpText ? <br /> : " "}
            Data will still be sent to your log destination if{" "}
            <b>automations</b> are <b>on</b>.
          </>
        )}
      </div>
    </div>
  );
};

export default DiscardDataOption;
