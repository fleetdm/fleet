import { VERSION_FAILURE, VERSION_START, VERSION_SUCCESS } from "./actions";

export const initialState = {
  data: {},
  errors: {},
  loading: false,
};

const reducer = (state = initialState, { type, payload }) => {
  switch (type) {
    case VERSION_START:
      return {
        ...state,
        loading: true,
      };
    case VERSION_SUCCESS:
      return {
        ...state,
        data: payload.data,
        loading: false,
      };
    case VERSION_FAILURE:
      return {
        ...state,
        errors: payload.errors,
        loading: false,
      };
    default:
      return state;
  }
};

export default reducer;
