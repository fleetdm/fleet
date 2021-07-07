import React, { useState, useCallback } from "react"; //, { useEffect }
import { useDispatch, useSelector } from "react-redux";

import { push } from "react-router-redux";

import paths from "router/paths";

const baseClass = "manage-schedule-page";

const Dashboard = (): JSX.Element => {
  // Links to packs page
  const dispatch = useDispatch();
  const { MANAGE_HOSTS } = paths;

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__wrapper body-wrap`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <h1 className={`${baseClass}__title`}>
                <span>Company Name</span>
              </h1>
              <div className={`${baseClass}__section hosts-section`}>
                <div className={`${baseClass}__section-title`}>Title 1</div>
                <div className={`${baseClass}__section-details`}>Details</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;
