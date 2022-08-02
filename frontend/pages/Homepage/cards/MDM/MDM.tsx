import React, { useState } from "react";
import { useQuery } from "react-query";

import macadminsAPI from "services/entities/macadmins";
import { IMacadminAggregate, IDataTableMDMFormat } from "interfaces/macadmins";

import TableContainer from "components/TableContainer";
import Spinner from "components/Spinner";
import TableDataError from "components/DataError";
import LastUpdatedText from "components/LastUpdatedText";
import generateTableHeaders from "./MDMTableConfig";

interface IMDMCardProps {
  showMDMUI: boolean;
  currentTeamId: number | undefined;
  setShowMDMUI: (showMDMTitle: boolean) => void;
  setTitleDetail?: (content: JSX.Element | string | null) => void;
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
        href="https://fleetdm.com/docs/using-fleet/adding-hosts#osquery-installer"
        target="_blank"
        rel="noopener noreferrer"
      >
        Fleet&apos;s osquery installer
      </a>
      .
    </p>
  </div>
);

const MDM = ({
  showMDMUI,
  currentTeamId,
  setShowMDMUI,
  setTitleDetail,
}: IMDMCardProps): JSX.Element => {
  const [formattedMDMData, setFormattedMDMData] = useState<
    IDataTableMDMFormat[]
  >([]);

  const { isFetching: isMDMFetching, error: errorMDM } = useQuery<
    IMacadminAggregate,
    Error
  >(["MDM", currentTeamId], () => macadminsAPI.loadAll(currentTeamId), {
    keepPreviousData: true,
    onSuccess: (data) => {
      const {
        counts_updated_at,
        mobile_device_management_enrollment_status,
      } = data.macadmins;
      const {
        enrolled_manual_hosts_count,
        enrolled_automated_hosts_count,
        unenrolled_hosts_count,
      } = mobile_device_management_enrollment_status;

      setShowMDMUI(true);
      setTitleDetail &&
        setTitleDetail(
          <LastUpdatedText
            lastUpdatedAt={counts_updated_at}
            whatToRetrieve={"MDM enrollment"}
          />
        );
      setFormattedMDMData([
        {
          status: "Enrolled (manual)",
          hosts: enrolled_manual_hosts_count,
        },
        {
          status: "Enrolled (automatic)",
          hosts: enrolled_automated_hosts_count,
        },
        { status: "Unenrolled", hosts: unenrolled_hosts_count },
      ]);
    },
    onError: () => {
      setShowMDMUI(true);
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
        {errorMDM ? (
          <TableDataError card />
        ) : (
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
        )}
      </div>
    </div>
  );
};

export default MDM;
