import {
  GET_STATUS_LABEL_COUNTS_FAILURE,
  GET_STATUS_LABEL_COUNTS_SUCCESS,
  LOAD_STATUS_LABEL_COUNTS,
  SET_DISPLAY,
} from './actions';

export const initialState = {
  display: 'Grid',
  status_labels: {
    errors: {},
    loading_counts: false,
    online_count: 0,
    offline_count: 0,
    mia_count: 0,
  },
};

export default (state = initialState, { type, payload }) => {
  switch (type) {
    case GET_STATUS_LABEL_COUNTS_FAILURE:
      return {
        ...state,
        status_labels: {
          ...state.status_labels,
          errors: payload.errors,
          loading_counts: false,
        },
      };
    case GET_STATUS_LABEL_COUNTS_SUCCESS:
      return {
        ...state,
        status_labels: {
          ...payload.status_labels,
          errors: {},
          loading_counts: false,
        },
      };
    case LOAD_STATUS_LABEL_COUNTS:
      return {
        ...state,
        status_labels: {
          ...state.status_labels,
          loading_counts: true,
        },
      };
    case SET_DISPLAY:
      return {
        ...state,
        display: payload.display,
      };
    default:
      return state;
  }
};
