import { reduxMockStore } from "test/helpers";

const errorAction = {
  type: "users_LOAD_FAILURE",
  payload: {
    errors: {
      http_status: 500,
      base: "Something went wrong",
    },
  },
};
const errorActionThunk = (dispatch) => {
  dispatch(errorAction);

  return Promise.reject();
};

describe("redirect - middleware", () => {
  it("redirect to /500 when a 500 error message is dispatched", () => {
    const mockStore = reduxMockStore();
    const expectedRedirectAction = {
      type: "@@router/CALL_HISTORY_METHOD",
      payload: {
        args: ["/500"],
        method: "push",
      },
    };

    mockStore.dispatch(errorActionThunk).catch(() => false);

    expect(mockStore.getActions()).toContainEqual(expectedRedirectAction);
  });
});
