import React, { useContext } from "react";
import { AppContext } from "context/app";

import paths from "router/paths";
import { Link } from "react-router";
import HostsSummary from "./HostsSummary";
import ActivityFeed from "./ActivityFeed";

import LinkArrow from "../../../assets/images/icon-arrow-right-vibrant-blue-10x18@2x.png";

const baseClass = "homepage";

const Homepage = (): JSX.Element => {
  const { MANAGE_HOSTS } = paths;
  const { config } = useContext(AppContext);

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__header-wrap`}>
        <div className={`${baseClass}__header`}>
          <h1 className={`${baseClass}__title`}>
            <span>{config?.org_name}</span>
          </h1>
        </div>
      </div>
      <div className={`${baseClass}__section one-column`}>
        <div className={`${baseClass}__info-card`}>
          <div className={`${baseClass}__section-title`}>
            <h2>Hosts</h2>
            <Link to={MANAGE_HOSTS} className={`${baseClass}__host-link`}>
              <span>View all hosts</span>
              <img src={LinkArrow} alt="link arrow" id="link-arrow" />
            </Link>
          </div>
          <HostsSummary />
        </div>
      </div>
      <div className={`${baseClass}__section one-column`}>
        <div className={`${baseClass}__info-card`}>
          <div className={`${baseClass}__section-title`}>
            <h2>Activity</h2>
          </div>
          <ActivityFeed />
        </div>
      </div>
    </div>
  );
};

export default Homepage;
