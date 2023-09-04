import React, { useContext, useEffect } from "react";
import { useErrorHandler } from "react-error-boundary";
import { useQuery } from "react-query";
import PATHS from "router/paths";

import { AppContext } from "context/app";
import {
  formatSoftwareType,
  ISoftware,
  IGetSoftwareByIdResponse,
} from "interfaces/software";
import { APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";
import softwareAPI from "services/entities/software";
import hostCountAPI from "services/entities/host_count";

import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import Spinner from "components/Spinner";
import BackLink from "components/BackLink";
import MainContent from "components/MainContent";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import Vulnerabilities from "./components/Vulnerabilities";

const baseClass = "software-details-page";

interface ISoftwareDetailsProps {
  params: {
    software_id: string;
  };
}

const SoftwareDetailsPage = ({
  params: { software_id },
}: ISoftwareDetailsProps): JSX.Element => {
  const {
    isPremiumTier,
    isSandboxMode,
    currentTeam,
    filteredSoftwarePath,
  } = useContext(AppContext);

  const handlePageError = useErrorHandler();

  const { data: software, isFetching: isFetchingSoftware } = useQuery<
    IGetSoftwareByIdResponse,
    Error,
    ISoftware
  >(
    ["softwareById", software_id],
    () => softwareAPI.getSoftwareById(software_id),
    {
      select: (data) => data.software,
      onError: (err) => handlePageError(err),
    }
  );

  const { data: hostCount } = useQuery<{ count: number }, Error, number>(
    ["hostCountBySoftwareId", software_id],
    () => hostCountAPI.load({ softwareId: parseInt(software_id, 10) }),
    { select: (data) => data.count }
  );

  const renderName = (sw: ISoftware) => {
    const { name, version } = sw;
    if (!name) {
      return "--";
    }
    if (!version) {
      return name;
    }

    return `${name}, ${version}`;
  };

  // Updates title that shows up on browser tabs
  useEffect(() => {
    // e.g., Software horizon, 5.2.0 details | Fleet for osquery
    document.title = `Software details | ${
      software && renderName(software)
    } | Fleet for osquery`;
  }, [location.pathname, software]);

  if (!software || isPremiumTier === undefined) {
    return <Spinner />;
  }

  // Function instead of constant eliminates race condition with filteredSoftwarePath
  const backToSoftwarePath = () => {
    if (filteredSoftwarePath) {
      return filteredSoftwarePath;
    }
    return currentTeam && currentTeam?.id > APP_CONTEXT_NO_TEAM_ID
      ? `${PATHS.MANAGE_SOFTWARE}?team_id=${currentTeam?.id}`
      : PATHS.MANAGE_SOFTWARE;
  };

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__wrapper`}>
        <div className={`${baseClass}__header-links`}>
          <BackLink text="Back to software" path={backToSoftwarePath()} />
        </div>
        <div className="header title">
          <div className="title__inner">
            <div className="name-container">
              <h1 className="name">{renderName(software)}</h1>
            </div>
          </div>
          <ViewAllHostsLink
            queryParams={{ software_id }}
            className={`${baseClass}__hosts-link`}
          />
        </div>
        <div className="section info">
          <div className="info__inner">
            <div className="info-flex">
              <div className="info-flex__item info-flex__item--title">
                <span className="info-flex__header">Type</span>
                <span className={`info-flex__data`}>
                  {formatSoftwareType(software.source)}
                </span>
              </div>
              <div className="info-flex__item info-flex__item--title">
                <span className="info-flex__header">Hosts</span>
                <span className={`info-flex__data`}>
                  {hostCount || DEFAULT_EMPTY_CELL_VALUE}
                </span>
              </div>
            </div>
          </div>
        </div>
        <Vulnerabilities
          isPremiumTier={isPremiumTier}
          isSandboxMode={isSandboxMode}
          isLoading={isFetchingSoftware}
          software={software}
        />
      </div>
    </MainContent>
  );
};

export default SoftwareDetailsPage;
