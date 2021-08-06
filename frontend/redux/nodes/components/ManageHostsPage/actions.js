import Fleet from "fleet";
import { formatErrorResponse } from "redux/nodes/entities/base/helpers";
import hostActions from "redux/nodes/entities/hosts/actions";
import labelActions from "redux/nodes/entities/labels/actions";

// Action Types
export const GET_STATUS_LABEL_COUNTS_FAILURE =
  "GET_STATUS_LABEL_COUNTS_FAILURE";
export const GET_STATUS_LABEL_COUNTS_SUCCESS =
  "GET_STATUS_LABEL_COUNTS_SUCCESS";
export const LOAD_STATUS_LABEL_COUNTS = "LOAD_STATUS_LABEL_COUNTS";

// Actions
export const loadStatusLabelCounts = { type: LOAD_STATUS_LABEL_COUNTS };
export const getStatusLabelCountsFailure = (errors) => {
  return {
    type: GET_STATUS_LABEL_COUNTS_FAILURE,
    payload: { errors },
  };
};
export const getStatusLabelCountsSuccess = (statusLabelCounts) => {
  return {
    type: GET_STATUS_LABEL_COUNTS_SUCCESS,
    payload: { status_labels: statusLabelCounts },
  };
};

export const silentGetStatusLabelCounts = (dispatch) => {
  return Fleet.statusLabels
    .getCounts()
    .then((counts) => {
      dispatch(getStatusLabelCountsSuccess(counts));

      return counts;
    })
    .catch((response) => {
      const errorsObject = formatErrorResponse(response);

      dispatch(getStatusLabelCountsFailure(errorsObject));

      throw errorsObject;
    });
};

export const getStatusLabelCounts = (dispatch) => {
  dispatch(loadStatusLabelCounts);

  return Fleet.statusLabels
    .getCounts()
    .then((counts) => {
      dispatch(getStatusLabelCountsSuccess(counts));

      return counts;
    })
    .catch((response) => {
      const errorsObject = formatErrorResponse(response);

      dispatch(getStatusLabelCountsFailure(errorsObject));

      throw errorsObject;
    });
};

export const getLabels = () => (dispatch) => {
  dispatch(labelActions.loadAll());
  dispatch(silentGetStatusLabelCounts);
};

export const getHosts = ({
  page,
  perPage,
  selectedLabel,
  globalFilter,
  sortBy,
}) => (dispatch) => {
  dispatch(
    hostActions.loadAll({ page, perPage, selectedLabel, globalFilter, sortBy })
  );
};

export default {
  getStatusLabelCounts,
  silentGetStatusLabelCounts,
  getHostTableData: getHosts,
};
