import ReduxConfig from "redux/nodes/entities/base/config";
import schemas from "redux/nodes/entities/base/schemas";
import { userStub } from "test/stubs";

describe("ReduxConfig - reducer", () => {
  const state = {
    loading: false,
    errors: {},
    data: {
      [userStub.id]: userStub,
    },
    originalOrder: [userStub.id],
  };

  describe("creating an entity", () => {
    const config = new ReduxConfig({
      entityName: "users",
      schema: schemas.USERS,
    });
    const { actions, reducer } = config;

    describe("successful action", () => {
      it("adds the user to state", () => {
        const createSuccessAction = actions.successAction(
          [userStub],
          actions.createSuccess
        );

        const newState = reducer(ReduxConfig.initialState, createSuccessAction);

        expect(newState).toEqual({
          loading: false,
          errors: {},
          data: {
            [userStub.id]: userStub,
          },
          originalOrder: [userStub.id],
        });
      });
    });

    describe("unsuccessful action", () => {
      it("adds the errors to state", () => {
        const errors = { base: "User is not authenticated" };
        const createFailureAction = actions.createFailure(errors);

        const newState = reducer(ReduxConfig.initialState, createFailureAction);

        expect(newState).toEqual({
          loading: false,
          errors,
          data: {},
          originalOrder: [],
        });
      });
    });
  });

  describe("destroying an entity", () => {
    const config = new ReduxConfig({
      entityName: "users",
      schema: schemas.USERS,
    });
    const { actions, reducer } = config;

    describe("successful action", () => {
      it("removes the user from state", () => {
        const destroySuccessAction = actions.destroySuccess(userStub.id);

        const newState = reducer(state, destroySuccessAction);

        expect(newState).toEqual({
          loading: false,
          errors: {},
          data: {},
          originalOrder: [],
        });
      });
    });

    describe("unsuccessful action", () => {
      it("adds the errors to state", () => {
        const errors = { base: "User is not authenticated" };
        const destroyFailureAction = actions.destroyFailure(errors);

        const newState = reducer(state, destroyFailureAction);

        expect(newState).toEqual({
          ...state,
          errors,
        });
      });
    });
  });

  describe("loading an entity", () => {
    const config = new ReduxConfig({
      entityName: "users",
      schema: schemas.USERS,
    });
    const { actions, reducer } = config;

    describe("successful action", () => {
      it("adds the user to state", () => {
        const loadSuccessAction = actions.successAction(
          [userStub],
          actions.loadSuccess
        );

        const newState = reducer(ReduxConfig.initialState, loadSuccessAction);

        expect(newState).toEqual({
          loading: false,
          errors: {},
          data: {
            [userStub.id]: userStub,
          },
          originalOrder: [userStub.id],
        });
      });
    });

    describe("unsuccessful action", () => {
      it("adds the errors to state", () => {
        const errors = { base: "User is not authenticated" };
        const loadFailureAction = actions.loadFailure(errors);

        const newState = reducer(ReduxConfig.initialState, loadFailureAction);

        expect(newState).toEqual({
          loading: false,
          errors,
          data: {},
          originalOrder: [],
        });
      });
    });
  });

  describe("loading all entities", () => {
    const config = new ReduxConfig({
      entityName: "users",
      schema: schemas.USERS,
    });
    const { actions, reducer } = config;
    const newUser = { id: 101, name: "Joe Schmoe" };

    describe("successful action", () => {
      it("replaces the users in state", () => {
        const loadAllSuccessAction = actions.successAction(
          [newUser],
          actions.loadAllSuccess
        );

        const newState = reducer(state, loadAllSuccessAction);

        expect(newState).toEqual({
          loading: false,
          errors: {},
          data: {
            101: newUser,
          },
          originalOrder: [newUser.id],
        });
      });
    });
  });

  describe("updating an entity", () => {
    const config = new ReduxConfig({
      entityName: "users",
      schema: schemas.USERS,
    });
    const { actions, reducer } = config;
    const newUser = { ...userStub, name: "Kolide", something: "else" };

    describe("successful action", () => {
      const updateSuccessAction = actions.successAction(
        [newUser],
        actions.updateSuccess
      );
      const newState = reducer(state, updateSuccessAction);

      it("replaces the user in state", () => {
        expect(newState).toEqual({
          loading: false,
          errors: {},
          data: {
            [userStub.id]: newUser,
          },
          originalOrder: [userStub.id],
        });
      });
    });

    describe("unsuccessful action", () => {
      const errors = { base: "User is not authenticated" };
      const updateFailureAction = actions.updateFailure(errors);

      const newState = reducer(state, updateFailureAction);

      it("adds the errors to state", () => {
        expect(newState).toEqual({
          ...state,
          errors,
        });
      });
    });
  });

  describe("clear errors", () => {
    const errorState = {
      loading: false,
      errors: {
        base: "User is not authenticated",
      },
      data: {
        [userStub.id]: userStub,
      },
    };
    const config = new ReduxConfig({
      entityName: "users",
      schema: schemas.USERS,
    });
    const { actions, reducer } = config;

    it("resets the entity errors", () => {
      const newState = reducer(errorState, actions.clearErrors());

      expect(newState).toEqual({
        ...errorState,
        errors: {},
      });
    });
  });
});
