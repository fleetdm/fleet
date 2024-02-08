import React from "react";

import { HumanTimeDiffWithFleetLaunchCutoff } from "components/HumanTimeDiffWithDateTip";
import { uniqueId } from "lodash";
import ReactTooltip from "react-tooltip";
import { COLORS } from "styles/var/colors";
import Icon from "components/Icon";
import TextCell from "components/TableContainer/DataTable/TextCell";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";

const baseClass = "report-updated-cell";

interface IReportUpdatedCell {
  last_fetched?: string | null;
  interval?: number;
  discard_data?: boolean;
  automations_enabled?: boolean;
  should_link_to_hqr?: boolean;
}

const ReportUpdatedCell = ({
  last_fetched,
  interval,
  discard_data,
  automations_enabled,
  should_link_to_hqr,
}: IReportUpdatedCell) => {
  const renderCellValue = () => {
    // if this query doesn't have an interval, it either has a stored report from previous runs
    // and will link to that report, or won't be included in this data in the first place.
    if (interval) {
      if (discard_data && automations_enabled) {
        // this is also the only case where the row is NOT clickable with a link to the host's HQR
        // query runs, sends results to a logging dest, doesn't cache
        return (
          <TextCell
            classes={`${baseClass}__value no-report`}
            formatter={(val) => {
              const tooltipId = uniqueId();
              return (
                <>
                  <span data-tip data-for={tooltipId}>
                    {val}
                  </span>
                  <ReactTooltip
                    place="top"
                    effect="solid"
                    backgroundColor={COLORS["tooltip-bg"]}
                    id={tooltipId}
                  >
                    {
                      <>
                        Results from this query are not reported in Fleet.
                        <br />
                        Data is being sent to your log destination.
                      </>
                    }
                  </ReactTooltip>
                </>
              );
            }}
            value="No report"
          />
        );
      }

      // Query is scheduled to run on host, but hasn't yet
      if (!last_fetched) {
        const tipId = uniqueId();
        return (
          <TextCell
            value={DEFAULT_EMPTY_CELL_VALUE}
            formatter={(val) => (
              <>
                <span data-tip data-for={tipId}>
                  {val}
                </span>
                <ReactTooltip
                  id={tipId}
                  effect="solid"
                  backgroundColor={COLORS["tooltip-bg"]}
                  place="top"
                >
                  Fleet is collecting query results.
                  <br />
                  Check back later.
                </ReactTooltip>
              </>
            )}
            greyed
            classes={`${baseClass}__value`}
          />
        );
      }
    }

    // render with link to cached results (link handled by clickable parent row)
    return (
      <>
        <TextCell
          // last_fetched will be truthy at this point
          value={{ timeString: last_fetched ?? "" }}
          formatter={HumanTimeDiffWithFleetLaunchCutoff}
          classes={`${baseClass}__value`}
        />
      </>
    );
  };

  return (
    <span className={baseClass}>
      {renderCellValue()}
      {should_link_to_hqr && (
        // actual link functionality handled by clickable parent row
        <span
          className={`${baseClass}__link`}
          title="link to host query report"
        >
          <span className={`${baseClass}__link-text`}>View report</span>
          <Icon
            name="chevron-right"
            className={`${baseClass}__link-icon`}
            color="core-fleet-blue"
          />
        </span>
      )}
    </span>
  );
};

export default ReportUpdatedCell;
