import React from "react";
import { useQuery } from "react-query";
import ReactTooltip from "react-tooltip";
import formatDistanceToNowStrict from "date-fns/formatDistanceToNowStrict";

import macadminsAPI from "services/entities/macadmins";
import { IMacadminAggregate, IMunkiAggregate } from "interfaces/macadmins";

import TableContainer from "components/TableContainer";
// @ts-ignore
import Spinner from "components/Spinner";
import generateTableHeaders from "./MunkiTableConfig";
import QuestionIcon from "../../../../../assets/images/icon-question-16x16@2x.png";

interface IMunkiCardProps {
  setShowMunkiUI: (showMunkiTitle: boolean) => void;
  showMunkiUI: boolean;
}

const DEFAULT_SORT_DIRECTION = "desc";
const DEFAULT_SORT_HEADER = "hosts_count";
const PAGE_SIZE = 8;
const baseClass = "home-munki";

const EmptyMunki = (): JSX.Element => (
  <div className={`${baseClass}__empty-munki`}>
    <h1>Unable to detect Munki versions.</h1>
    <p>
      To see Munki versions, deploy&nbsp;
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
            to retrieve Munki versions
          </span>
        </ReactTooltip>
      </span>
    </span>
  );
};

const Munki = ({
  setShowMunkiUI,
  showMunkiUI,
}: IMunkiCardProps): JSX.Element => {
  const { isFetching: isMunkiFetching, data: munkiData } = useQuery<
    IMacadminAggregate,
    Error,
    IMunkiAggregate[]
  >(["munki"], () => macadminsAPI.loadAll(), {
    keepPreviousData: true,
    select: (data: IMacadminAggregate) => data.macadmins.munki_versions,
    onSuccess: (data) => {
      setShowMunkiUI(true);
    },
  });

  const tableHeaders = generateTableHeaders();

  // Renders opaque information as host information is loading
  const opacity = showMunkiUI ? { opacity: 1 } : { opacity: 0 };

  return (
    <div className={baseClass}>
      {!showMunkiUI && (
        <div className="spinner">
          <Spinner />
        </div>
      )}
      <div style={opacity}>
        <TableContainer
          columns={tableHeaders}
          data={munkiData || []}
          isLoading={isMunkiFetching}
          defaultSortHeader={DEFAULT_SORT_HEADER}
          defaultSortDirection={DEFAULT_SORT_DIRECTION}
          hideActionButton
          resultsTitle={"Munki"}
          emptyComponent={EmptyMunki}
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

export default Munki;
