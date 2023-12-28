import React from "react";
import { useQuery } from "react-query";
import { RouteComponentProps } from "react-router";

import Spinner from "components/Spinner";
import TableDataError from "components/DataError";
import MainContent from "components/MainContent";

import SoftwareDetailsSummary from "../components/SoftwareDetailsSummary";
import SoftwareOSDetailsTable from "./SoftwareOSDetailsTable";

const baseClass = "software-os-details-page";

interface ISoftwareOSDetailsRouteParams {
  id: string;
}

type ISoftwareOSDetailsPageProps = RouteComponentProps<
  undefined,
  ISoftwareOSDetailsRouteParams
>;

const SoftwareOSDetailsPage = ({
  router,
  routeParams,
}: ISoftwareOSDetailsPageProps) => {
  // TODO: handle non integer values
  const softwareId = parseInt(routeParams.id, 10);

  const { data, isLoading, isError } = useQuery<
    ISoftwareTitleResponse,
    Error,
    ISoftwareTitle
  >(
    ["softwareById", softwareId],
    () => softwareAPI.getSoftwareTitle(softwareId),
    {
      select: (data) => data.software_title,
    }
  );

  const renderContent = () => {
    if (isLoading) {
      return <Spinner />;
    }

    if (isError) {
      return <TableDataError className={`${baseClass}__table-error`} />;
    }

    if (!data) {
      return null;
    }

    return (
      <>
        <SoftwareDetailsSummary
          id={softwareId}
          title={softwareTitle.name}
          type={formatSoftwareType(softwareTitle)}
          versions={softwareTitle.versions.length}
          hosts={softwareTitle.hosts_count}
          queryParam="software_title_id"
          name={softwareTitle.name}
          source={softwareTitle.source}
        />
        {/* TODO: can we use Card here for card styles */}
        <div className={`${baseClass}__versions-section`}>
          <h2>Vulnerabilities</h2>
          {/* <SoftwareOSDetailsTable
            router={router}
            data={softwareTitle.versions}
            isLoading={isSoftwareTitleLoading}
          /> */}
        </div>
      </>
    );
  };

  return (
    <MainContent className={baseClass}>
      <>{renderContent()}</>
    </MainContent>
  );
};

export default SoftwareOSDetailsPage;
