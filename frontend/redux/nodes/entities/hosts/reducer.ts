import { PayloadAction } from "@reduxjs/toolkit";
import config from "./config";
import { TRANSFER_HOSTS_SUCCESS, TRANSFER_HOSTS_FAILURE } from "./actions";

// TODO: figure out redux typing
export default (
  state = config.initialState,
  { type, payload }: PayloadAction<any>
): any => {
  switch (type) {
    case TRANSFER_HOSTS_SUCCESS:
      return {
        ...state,
        errors: {},
        loading: false,
        data: {
          ...state.data,
        },
      };
    case TRANSFER_HOSTS_FAILURE:
      return {
        ...state,
        loading: false,
        errors: payload.errors,
      };
    default:
      return config.reducer(state, { type, payload });
  }
};
