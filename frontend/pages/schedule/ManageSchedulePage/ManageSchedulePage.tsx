import React, { useEffect } from "react";
import { useDispatch, useSelector } from "react-redux";
import { push } from "react-router-redux";
// @ts-ignore
import { IUser } from "interfaces/user";

// Will I need this? 5/28
// import permissionUtils from "utilities/permissions";
import paths from "router/paths";
import Button from "components/buttons/Button";
import NoSchedule from "./components/NoSchedule";

const baseClass = "manage-schedule-page";

const renderTable = (handleAdvanced: any): any => {
  const fakeData = {
    scheduled: [
      {
        id: 1,
        query_id: 4,
        interval: 172800,
        last_executed: "2021-06-23T20:26:51Z",
      },
      {
        id: 2,
        query_id: 7,
        interval: 14400,
        last_executed: "2021-06-24T20:26:51Z",
      },
      {
        id: 3,
        query_id: 8,
        interval: 86400,
        last_executed: "2021-06-23T20:26:51Z",
      },
      {
        id: 4,
        query_id: 20,
        interval: 604800,
        last_executed: "2021-06-21T20:26:51Z",
      },
    ],
  };
  const fakeDataLength0 = [];

  // Schedule has not been set up for this instance yet.
  if (fakeDataLength0.length === 0) {
    return <NoSchedule />;
  }
};
interface IRootState {
  auth: {
    user: IUser;
  };
}

const ManageSchedulePage = (): JSX.Element => {
  const dispatch = useDispatch();
  const { MANAGE_PACKS } = paths;

  // Will I need this? 5/28
  // const user = useSelector((state: IRootState) => state.auth.user);

  const handleAdvanced = (event: any) => dispatch(push(MANAGE_PACKS));

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__wrapper body-wrap`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <h1 className={`${baseClass}__title`}>
                <span>Schedule</span>
              </h1>
              <div className={`${baseClass}__description`}>
                <p>
                  Schedule recurring queries for your hosts. Fleet’s query
                  schedule lets you add queries which are executed at regular
                  intervals.
                </p>
              </div>
            </div>
          </div>
          <div className={`${baseClass}__action-button-container`}>
            <Button
              variant="inverse"
              onClick={handleAdvanced}
              className={`${baseClass}__advanced-button`}
            >
              Advanced
            </Button>
            {/* TODO: SCHEDULE A QUERY MODAL */}
            <Button variant="brand" className={`${baseClass}__schedule-button`}>
              Schedule a query
            </Button>
          </div>
        </div>
        <div>{renderTable(handleAdvanced)}</div>
      </div>
    </div>
  );
};

export default ManageSchedulePage;
