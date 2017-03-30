import {
  RESET_OPTIONS_START,
  RESET_OPTIONS_SUCCESS,
  RESET_OPTIONS_FAILURE,
} from './actions';

import config, { initialState } from './config';

export default (state = initialState, { type, payload }) => {
  switch (type) {
    case RESET_OPTIONS_START:
      return {
        ...state,
        errors: {},
        loading: true,
        data: {
          ...state.data,
        },
      };
    case RESET_OPTIONS_SUCCESS:
      return {
        ...state,
        errors: {},
        loading: false,
        data: payload.configOptions,
      };
    case RESET_OPTIONS_FAILURE:
      return {
        ...state,
        errors: payload.errors,
      };
    default:
      return config.reducer(state, { type, payload });
  }
};
