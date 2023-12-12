import React, { useContext } from "react";
import { RouteComponentProps } from "react-router";
import { useQuery } from "react-query";

import { AppContext } from "context/app";
import { ISoftwareTitle } from "interfaces/software";
import softwareAPI, {
  ISoftwareTitleResponse,
} from "services/entities/software";

import MainContent from "components/MainContent";
import TableDataError from "components/DataError";

import SoftwareDetailsSummary from "../components/SoftwareDetailsSummary";
import SoftwareTitleDetailsTable from "./SoftwareTitleDetailsTable";

const baseClass = "software-title-details-page";

interface ISoftwareTitleDetailsRouteParams {
  id: string;
}

type ISoftwareTitleDetailsPageProps = RouteComponentProps<
  undefined,
  ISoftwareTitleDetailsRouteParams
>;

const SoftwareTitleDetailsPage = ({
  router,
  routeParams,
}: ISoftwareTitleDetailsPageProps) => {
  // TODO: handle non integer values
  const softwareId = parseInt(routeParams.id, 10);

  const {
    data: softwareTitle,
    isLoading: isSoftwareTitleLoading,
    isError: isSoftwareTitleError,
  } = useQuery<ISoftwareTitleResponse, Error, ISoftwareTitle>(
    ["softwareById", softwareId],
    () => softwareAPI.getSoftwareTitle(softwareId),
    {
      select: (data) => data.software_title,
    }
  );

  if (!softwareTitle) {
    return null;
  }

  return (
    <MainContent className={baseClass}>
      {isSoftwareTitleError ? (
        <TableDataError className={`${baseClass}__table-error`} />
      ) : (
        <>
          <SoftwareDetailsSummary
            id={softwareId}
            title={softwareTitle.name}
            type={softwareTitle.source}
            versions={softwareTitle.versions.length}
            hosts={softwareTitle.hosts_count}
            queryParam="software_title_id"
            name={softwareTitle.name}
            source={softwareTitle.source}
          />
          {/* TODO: can we use Card here for card styles */}
          <div className={`${baseClass}__versions-section`}>
            <h2>Versions</h2>
            <SoftwareTitleDetailsTable
              router={router}
              data={softwareTitle.versions}
              isLoading={isSoftwareTitleLoading}
            />
          </div>
        </>
      )}
    </MainContent>
  );
};

export default SoftwareTitleDetailsPage;
