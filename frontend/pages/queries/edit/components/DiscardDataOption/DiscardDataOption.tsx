import Checkbox from "components/forms/fields/Checkbox";
import Icon from "components/Icon";
import InfoBanner from "components/InfoBanner";
import TooltipWrapper from "components/TooltipWrapper";
import { QueryLoggingOption } from "interfaces/schedulable_query";
import React, { useState } from "react";
import { Link } from "react-router";

const baseClass = "discard-data-option";

interface IDiscardDataOptionProps {
  queryReportsDisabled: boolean;
  selectedLoggingType: QueryLoggingOption;
  discardData: boolean;
  setDiscardData: (value: boolean) => void;
}

const DiscardDataOption = ({
  queryReportsDisabled,
  selectedLoggingType,
  discardData,
  setDiscardData,
}: IDiscardDataOptionProps) => {
  const [forceEditDiscardData, setForceEditDiscardData] = useState(false);
  const disable = queryReportsDisabled && !forceEditDiscardData;

  const renderHelpText = () => (
    <div className="help-text">
      {disable ? (
        <>
          This setting is ignored because query reports in Fleet have been{" "}
          <TooltipWrapper
            tipContent={
              <>
                A Fleet administrator can enable query reports under <br />
                <b>
                  Organization settings &gt; Advanced options &gt; Disable query
                  reports
                </b>
                .
              </>
            }
          >
            {"globally disabled."}
          </TooltipWrapper>{" "}
          <Link
            to=""
            onClick={(e: React.MouseEvent) => {
              e.preventDefault();
              setForceEditDiscardData(true);
            }}
            className={`${baseClass}__edit-anyway`}
          >
            <>
              Edit anyway
              <Icon name="chevron-right" color="core-fleet-blue" size="small" />
            </>
          </Link>
        </>
      ) : (
        "The most recent results for each host will not be available in Fleet."
      )}
    </div>
  );
  return (
    <div className={baseClass}>
      {["differential", "differential_ignore_removals"].includes(
        selectedLoggingType
      ) && (
        <InfoBanner color="purple-bold-border">
          <>
            The <b>Discard data</b> setting is ignored when differential logging
            is enabled. This query&apos;s results will not be saved in Fleet.
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
        helpText={renderHelpText()}
      >
        Discard data
      </Checkbox>
    </div>
  );
};

export default DiscardDataOption;
