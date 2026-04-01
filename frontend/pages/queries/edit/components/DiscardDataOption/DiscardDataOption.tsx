import React, { useState } from "react";

import { QueryLoggingOption } from "interfaces/schedulable_query";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import Icon from "components/Icon";
import InfoBanner from "components/InfoBanner";
import TooltipWrapper from "components/TooltipWrapper";

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

  const isDisabled = queryReportsDisabled && !forceEditDiscardData;
  const isReportsLoggingIgnored =
    selectedLoggingType === "differential" ||
    selectedLoggingType === "differential_ignore_removals";

  const renderHelpText = () => (
    <>
      {isDisabled ? (
        <>
          This setting is ignored because reports in Fleet have been{" "}
          <TooltipWrapper
            tipContent={
              <>
                A Fleet administrator can enable reports under <br />
                <b>
                  Organization settings &gt; Advanced options &gt; Disable
                  reports
                </b>
                .
              </>
            }
          >
            globally disabled.
          </TooltipWrapper>
          <Button
            onClick={(e: React.MouseEvent) => {
              e.preventDefault();
              setForceEditDiscardData(true);
            }}
            variant="text-icon"
            size="small"
            className={`${baseClass}__edit-anyway`}
            iconStroke
          >
            <>
              Edit anyway
              <Icon
                name="chevron-right"
                color="ui-fleet-black-75"
                size="small"
              />
            </>
          </Button>
        </>
      ) : (
        "The most recent results for each host will not be available in Fleet."
      )}
    </>
  );

  return (
    <div className={baseClass}>
      {isReportsLoggingIgnored && (
        <InfoBanner>
          The <b>Discard data</b> setting is ignored when differential logging
          is enabled. This report&apos;s results will not be saved in Fleet.
        </InfoBanner>
      )}
      <Checkbox
        name="discardData"
        onChange={setDiscardData}
        value={discardData}
        disabled={isDisabled}
        helpText={renderHelpText()}
      >
        Discard data
      </Checkbox>
    </div>
  );
};

export default DiscardDataOption;
