import Kolide from 'kolide';
import { formatErrorResponse } from 'redux/nodes/entities/base/helpers';
import hostActions from 'redux/nodes/entities/hosts/actions';
import labelActions from 'redux/nodes/entities/labels/actions';

// Action Types
export const GET_STATUS_LABEL_COUNTS_FAILURE = 'GET_STATUS_LABEL_COUNTS_FAILURE';
export const GET_STATUS_LABEL_COUNTS_SUCCESS = 'GET_STATUS_LABEL_COUNTS_SUCCESS';
export const LOAD_STATUS_LABEL_COUNTS = 'LOAD_STATUS_LABEL_COUNTS';
export const SET_PAGINATION = 'SET_PAGINATION';
export const GET_HOSTS = 'GET_HOSTS';

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

export const setPaginationSuccess = (page, perPage, selectedLabel) => {
  return {
    type: SET_PAGINATION,
    payload: {
      page,
      perPage,
      selectedLabel,
    },
  };
};

export const setPagination = (page, perPage, selectedLabel, globalFilter, orderKey, isDesc) => (dispatch) => {
  const promises = [
    dispatch(hostActions.loadAll(page, perPage, selectedLabel, globalFilter, orderKey, isDesc)),
    dispatch(labelActions.silentLoadAll()),
    dispatch(silentGetStatusLabelCounts),
  ];

  Promise.all(promises).then(dispatch(setPaginationSuccess(page, perPage, selectedLabel)));
};

export const getLabels = () => (dispatch) => {
  const promises = [
    dispatch(labelActions.loadAll()),
    dispatch(silentGetStatusLabelCounts),
  ];
};

// export const getHostTableData = (page, perPage, selectedLabel, globalFilter, orderKey, isDesc) => (dispatch) => {
export const getHostTableData = (page, perPage, selectedLabel, globalFilter, sortBy) => (dispatch) => {
  const promises = [
    dispatch(hostActions.loadAll(page, perPage, selectedLabel, globalFilter, sortBy)),
  ];

  // Promise.all(promises).then(dispatch(setPaginationSuccess(page, perPage, selectedLabel)));
};

export default { getStatusLabelCounts, setPagination, silentGetStatusLabelCounts, getHostTableData };
