import React, { useState, useCallback } from "react"; //, { useEffect }
import { useDispatch, useSelector } from "react-redux";

import { push } from "react-router-redux";
import { IUser } from "interfaces/user";

import paths from "router/paths";

const baseClass = "dashboard";

interface RootState {
  auth: {
    user: IUser;
  };
}

const Dashboard = (): JSX.Element => {
  // Links to packs page
  const dispatch = useDispatch();
  const { MANAGE_HOSTS } = paths;

  const user = useSelector((state: RootState) => state.auth.user);

  console.log("USER", user);
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__wrapper body-wrap`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              {user.teams && (
                <h1 className={`${baseClass}__title`}>
                  <span>Team Name</span>
                </h1>
              )}
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
