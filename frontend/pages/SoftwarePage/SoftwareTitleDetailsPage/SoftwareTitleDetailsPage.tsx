import React, { useContext } from "react";

import MainContent from "components/MainContent";
import ViewAllHostsLink from "components/ViewAllHostsLink";
import { InjectedRouter, RouteComponentProps } from "react-router";
import { AppContext } from "context/app";

const baseClass = "SoftwareTitleDetailsPage";

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
  const {
    isPremiumTier,
    isSandboxMode,
    currentTeam,
    filteredSoftwarePath,
  } = useContext(AppContext);

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__wrapper`}>
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
      </div>
    </MainContent>
  );
  return <h1>script title details</h1>;
};

export default SoftwareTitleDetailsPage;
