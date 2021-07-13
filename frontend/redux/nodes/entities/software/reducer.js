import config, { initialState } from "./config";

export default (state = initialState, { type, payload }) => {
  switch (type) {
    default:
      return config.reducer(state, { type, payload });
  }
};
