import React from "react"; //, { useEffect }
import {
  useDispatch,
  //  , useSelector
} from "react-redux";

import { push } from "react-router-redux";
// @ts-ignore
import { IUser } from "interfaces/user";

// Will I need this? 5/28
// import permissionUtils from "utilities/permissions";
import paths from "router/paths";
import Button from "components/buttons/Button";
import NoSchedule from "./components/NoSchedule";
import ScheduleError from "./components/ScheduleError";

const baseClass = "manage-schedule-page";

// FAKE DATA ALERT

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

// 1. SET TO EMPTY IF YOU WANT TO SEE THE EMPTY STATE RENDER
const fakeDataLength0 = [1];

// 2. SET TO TRUE IF YOU WANT TO SEE THE ERROR STATE RENDER;
const fakeDataError = false;

// END FAKE DATA ALERT

const renderTable = (): JSX.Element => {
  // Schedule has not been set up for this instance yet.
  if (fakeDataLength0.length === 0) {
    return <NoSchedule />;
  }

  // Schedule has an error retrieving data.
  if (fakeDataError) {
    return <ScheduleError />;
  }

  return <div>Hi!</div>;
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
          {/* Hides CTA Buttons if no schedule or schedule error */}
          {fakeDataLength0.length !== 0 && !fakeDataError && (
            <div className={`${baseClass}__action-button-container`}>
              <Button
                variant="inverse"
                onClick={handleAdvanced}
                className={`${baseClass}__advanced-button`}
              >
                Advanced
              </Button>
              {/* TODO: SCHEDULE A QUERY MODAL */}
              <Button
                variant="brand"
                className={`${baseClass}__schedule-button`}
              >
                Schedule a query
              </Button>
            </div>
          )}
        </div>
        <div>{renderTable()}</div>
      </div>
    </div>
  );
};

export default ManageSchedulePage;
