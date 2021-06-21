import { push } from "react-router-redux";
import { join, omit, values } from "lodash";

import PATHS from "router/paths";
import queryActions from "redux/nodes/entities/queries/actions";
import { renderFlash } from "redux/nodes/notifications/actions";

export const fetchQuery = (dispatch, queryID) => {
  return dispatch(queryActions.load(queryID)).catch((errors) => {
    const { MANAGE_QUERIES } = PATHS;
    let errorMessage = join(values(omit(errors, "http_status")), ", ");

    if (errorMessage.includes("no rows in result set")) {
      errorMessage = "The query you requested does not exist in Fleet.";
    }

    dispatch(push(MANAGE_QUERIES));
    dispatch(renderFlash("error", errorMessage));

    return false;
  });
};

export default { fetchQuery };
