import { REFETCH_HOST_SUCCESS, REFETCH_HOST_FAILURE } from "./actions";
import config, { initialState } from "./config";

export default (state = initialState, { type, payload }) => {
  switch (type) {
    case REFETCH_HOST_SUCCESS:
      return {
        ...state,
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
