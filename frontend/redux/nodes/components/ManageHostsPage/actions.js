import Kolide from 'kolide';
import { formatErrorResponse } from 'redux/nodes/entities/base/helpers';

// Action Types
export const GET_STATUS_LABEL_COUNTS_FAILURE = 'GET_STATUS_LABEL_COUNTS_FAILURE';
export const GET_STATUS_LABEL_COUNTS_SUCCESS = 'GET_STATUS_LABEL_COUNTS_SUCCESS';
export const LOAD_STATUS_LABEL_COUNTS = 'LOAD_STATUS_LABEL_COUNTS';
export const SET_DISPLAY = 'SET_DISPLAY';

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

export const getStatusLabelCounts = (dispatch) => {
  dispatch(loadStatusLabelCounts);

  return Kolide.statusLabels.getCounts()
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

export const setDisplay = (display) => {
  return {
    type: SET_DISPLAY,
    payload: {
      display,
    },
  };
};

export default { getStatusLabelCounts, setDisplay };
