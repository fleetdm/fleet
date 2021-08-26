import {
  REQUIRE_PASSWORD_RESET_FAILURE,
  REQUIRE_PASSWORD_RESET_SUCCESS,
  CREATE_USER_WITHOUT_INVITE_FAILURE,
  CREATE_USER_WITHOUT_INVITE_SUCCESS,
} from "./actions";
import config, { initialState } from "./config";

export default (state = initialState, { type, payload }) => {
  switch (type) {
    case REQUIRE_PASSWORD_RESET_SUCCESS:
      return {
        ...state,
        errors: {},
        loading: false,
        data: {
          ...state.data,
          [payload.user.id]: payload.user,
        },
      };
    case REQUIRE_PASSWORD_RESET_FAILURE:
      return {
        ...state,
        loading: false,
        errors: payload.errors,
      };
    case CREATE_USER_WITHOUT_INVITE_SUCCESS:
      return {
        ...state,
        errors: {},
        loading: false,
        data: {
          ...state.data,
          [payload.user.id]: payload.user,
        },
      };
    case CREATE_USER_WITHOUT_INVITE_FAILURE:
      return {
        ...state,
        loading: false,
        errors: payload.errors,
      };

    default:
      return config.reducer(state, { type, payload });
  }
};
