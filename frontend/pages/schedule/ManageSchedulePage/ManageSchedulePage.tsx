import React, { useEffect } from "react";
import { useDispatch } from "react-redux";
import { push } from "react-router-redux";
// @ts-ignore
import { fetchCurrentUser, logoutUser } from "redux/nodes/auth/actions";

import paths from "router/paths";
import Button from "components/buttons/Button";

const baseClass = "manage-schedule-page";

const ManageSchedulePage = (): JSX.Element => {
  return (
    <div className={baseClass}>
      <div className={`${baseClass}__wrap`}>
        <div className={`${baseClass}__lead-wrapper`}>
          Why is this not working?
        </div>
      </div>
    </div>
  );
};

export default ManageSchedulePage;
