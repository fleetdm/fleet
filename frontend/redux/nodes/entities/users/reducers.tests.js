import reducer from "./reducer";
import {
  requirePasswordResetFailure,
  requirePasswordResetSuccess,
} from "./actions";

const user = { id: 1, email: "zwass@Fleet.co", force_password_reset: false };

describe("Users - reducer", () => {
  describe("require password reset", () => {
    const initialState = {
      loading: false,
      errors: {},
      data: {
        [user.id]: user,
      },
    };

    it("updates state when request is successful", () => {
      const initState = {
        ...initialState,
        loading: true,
      };
      const newUser = { ...user, force_password_reset: true };
      const newState = reducer(initState, requirePasswordResetSuccess(newUser));

      expect(newState).toEqual({
        ...initState,
        loading: false,
        data: {
          [user.id]: newUser,
        },
      });
    });

    it("updates state when request fails", () => {
      const errors = { base: "Unable to require password reset" };
      const newState = reducer(
        initialState,
        requirePasswordResetFailure(errors)
      );

      expect(newState).toEqual({
        ...initialState,
        errors,
      });
    });
  });
});
