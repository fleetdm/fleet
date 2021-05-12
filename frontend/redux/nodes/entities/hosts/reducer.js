import {
  REFETCH_HOST_START,
  REFETCH_HOST_SUCCESS,
  REFETCH_HOST_FAILURE,
} from "./actions";
import config, { initialState } from "./config";

// confirm initial state should come directly from config

// modeled this off of redux/nodes/osquery/reducer.js
export default (state = initialState, { type, payload }) => {
  switch (type) {
    case REFETCH_HOST_START:
      // what do we want to return, look at other reducers.js
      // what do we do if the endpoint returns the refetched data
      // do we want to pass back loading or refetching?
      return {
        ...state,
        // when should we set this to false, if ever, or use it in our code?
        refetching: true,
        loading: true,
      };
    case REFETCH_HOST_SUCCESS:
      return {
        ...state,
        options: payload.data,
        loading: false,
      };
    case REFETCH_HOST_FAILURE:
      return {
        ...state,
        errors: payload.errors,
        loading: false,
      };
    default:
      return config.reducer(state, { type, payload });
  }
};

// if it is a refetch action, update the state appropriately
// default reducer case should be call the reducer from config if it doesn't match the custom one, just call the reducer from config

// default returns state, but instead return the result of config.reducer with the args
