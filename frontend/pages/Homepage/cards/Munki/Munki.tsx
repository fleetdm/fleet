import React, { useState } from "react";
import { useQuery } from "react-query";

import macadminsAPI from "services/entities/macadmins";
import { IMacadminAggregate, IMunkiAggregate } from "interfaces/macadmins";

import TableContainer from "components/TableContainer";
// @ts-ignore
import Spinner from "components/Spinner";
import renderLastUpdatedAt from "../../components/LastUpdatedText";
import generateTableHeaders from "./MunkiTableConfig";

interface IMunkiCardProps {
  showMunkiUI: boolean;
  currentTeamId: number | undefined;
  setShowMunkiUI: (showMunkiTitle: boolean) => void;
  setTitleDetail?: (content: JSX.Element | string | null) => void;
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

const Munki = ({
  showMunkiUI,
  currentTeamId,
  setShowMunkiUI,
  setTitleDetail,
}: IMunkiCardProps): JSX.Element => {
  const [munkiData, setMunkiData] = useState<IMunkiAggregate[]>([]);

  const { isFetching: isMunkiFetching } = useQuery<IMacadminAggregate, Error>(
    ["munki", currentTeamId],
    () => macadminsAPI.loadAll(currentTeamId),
    {
      keepPreviousData: true,
      onSuccess: (data) => {
        const { counts_updated_at, munki_versions } = data.macadmins;

        setMunkiData(munki_versions);
        setShowMunkiUI(true);
        setTitleDetail &&
          setTitleDetail(
            renderLastUpdatedAt(counts_updated_at, "Munki versions")
          );
      },
    }
  );

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
