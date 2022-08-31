import React, { useContext } from "react";
import { useErrorHandler } from "react-error-boundary";
import { useQuery } from "react-query";
import { Link } from "react-router";
import PATHS from "router/paths";

import { AppContext } from "context/app";
import { formatSoftwareType, ISoftware } from "interfaces/software";
import softwareAPI, {
  IGetSoftwareByIdResponse,
} from "services/entities/software";
import hostCountAPI from "services/entities/host_count";

import Spinner from "components/Spinner";
import MainContent from "components/MainContent";
import BackChevron from "../../../../assets/images/icon-chevron-down-9x6@2x.png";
import RightChevron from "../../../../assets/images/icon-chevron-right-9x6@2x.png";

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
  const { isPremiumTier } = useContext(AppContext);
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

  if (!software || isPremiumTier === undefined) {
    return <Spinner />;
  }

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__wrapper`}>
        <div className={`${baseClass}__header-links`}>
          <Link
            to={PATHS.MANAGE_SOFTWARE}
            className={`${baseClass}__back-link`}
          >
            <img src={BackChevron} alt="back chevron" id="back-chevron" />
            <span>Back to software</span>
          </Link>
        </div>
        <div className="header title">
          <div className="title__inner">
            <div className="name-container">
              <h1 className="name">{renderName(software)}</h1>
            </div>
          </div>
          <Link
            to={`${PATHS.MANAGE_HOSTS}?software_id=${software_id}`}
            className={`${baseClass}__hosts-link`}
          >
            <span>View all hosts</span>
            <img src={RightChevron} alt="right chevron" id="right-chevron" />
          </Link>
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
                <span className={`info-flex__data`}>{hostCount || "---"}</span>
              </div>
            </div>
          </div>
        </div>
        <Vulnerabilities
          isPremiumTier={isPremiumTier}
          isLoading={isFetchingSoftware}
          software={software}
        />
      </div>
    </MainContent>
  );
};

export default SoftwareDetailsPage;
