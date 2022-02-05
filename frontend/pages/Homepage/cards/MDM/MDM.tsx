import React, { useState } from "react";
import { useQuery } from "react-query";
import ReactTooltip from "react-tooltip";
import formatDistanceToNowStrict from "date-fns/formatDistanceToNowStrict";

import macadminsAPI from "services/entities/macadmins";
import {
  IMacadminAggregate,
  IMDMAggregateStatus,
  IDataTableMDMFormat,
} from "interfaces/macadmins";

import TableContainer from "components/TableContainer";
// @ts-ignore
import Spinner from "components/Spinner";
import generateTableHeaders from "./MDMTableConfig";
import QuestionIcon from "../../../../../assets/images/icon-question-16x16@2x.png";

interface IMDMCardProps {
  setShowMDMUI: (showMDMTitle: boolean) => void;
  showMDMUI: boolean;
}

const DEFAULT_SORT_DIRECTION = "desc";
const DEFAULT_SORT_HEADER = "hosts_count";
const PAGE_SIZE = 8;
const baseClass = "home-mdm";

const EmptyMDM = (): JSX.Element => (
  <div className={`${baseClass}__empty-mdm`}>
    <h1>Unable to detect MDM enrollment.</h1>
    <p>
      To see MDM versions, deploy&nbsp;
      <a
        href="https://fleetdm.com/docs/deploying/configuration#software-inventory"
        target="_blank"
        rel="noopener noreferrer"
      >
        Fleet&apos;s osquery installer
      </a>
      .
    </p>
  </div>
);

const renderLastUpdatedAt = (lastUpdatedAt: string) => {
  if (!lastUpdatedAt || lastUpdatedAt === "0001-01-01T00:00:00Z") {
    lastUpdatedAt = "never";
  } else {
    lastUpdatedAt = formatDistanceToNowStrict(new Date(lastUpdatedAt), {
      addSuffix: true,
    });
  }
  return (
    <span className="last-updated">
      {`Last updated ${lastUpdatedAt}`}
      <span className={`tooltip`}>
        <span
          className={`tooltip__tooltip-icon`}
          data-tip
          data-for="last-updated-tooltip"
          data-tip-disable={false}
        >
          <img alt="question icon" src={QuestionIcon} />
        </span>
        <ReactTooltip
          place="top"
          type="dark"
          effect="solid"
          backgroundColor="#3e4771"
          id="last-updated-tooltip"
          data-html
        >
          <span className={`tooltip__tooltip-text`}>
            Fleet periodically
            <br />
            queries all hosts
            <br />
            to retrieve MDM enrollment
          </span>
        </ReactTooltip>
      </span>
    </span>
  );
};

const MDM = ({ setShowMDMUI, showMDMUI }: IMDMCardProps): JSX.Element => {
  const [formattedMDMData, setFormattedMDMData] = useState<
    IDataTableMDMFormat[]
  >([]);

  const { isFetching: isMDMFetching } = useQuery<
    IMacadminAggregate,
    Error,
    IMDMAggregateStatus
  >(["MDM"], () => macadminsAPI.loadAll(), {
    keepPreviousData: true,
    select: (data: IMacadminAggregate) =>
      data.macadmins.mobile_device_management_enrollment_status,
    onSuccess: (data) => {
      setShowMDMUI(true);
      setFormattedMDMData([
        {
          status: "Enrolled (manual)",
          hosts: data.enrolled_manual_hosts_count,
        },
        {
          status: "Enrolled (automatic)",
          hosts: data.enrolled_automated_hosts_count,
        },
        { status: "Unenrolled", hosts: data.unenrolled_hosts_count },
      ]);
    },
  });

  const tableHeaders = generateTableHeaders();

  // Renders opaque information as host information is loading
  const opacity = showMDMUI ? { opacity: 1 } : { opacity: 0 };

  return (
    <div className={baseClass}>
      {!showMDMUI && (
        <div className="spinner">
          <Spinner />
        </div>
      )}
      <div style={opacity}>
        <TableContainer
          columns={tableHeaders}
          data={formattedMDMData}
          isLoading={isMDMFetching}
          defaultSortHeader={DEFAULT_SORT_HEADER}
          defaultSortDirection={DEFAULT_SORT_DIRECTION}
          hideActionButton
          resultsTitle={"MDM"}
          emptyComponent={EmptyMDM}
          showMarkAllPages={false}
          isAllPagesSelected={false}
          disableCount
          disableActionButton
          disablePagination
          pageSize={PAGE_SIZE}
        />
      </div>
    </div>
  );
};

export default MDM;
