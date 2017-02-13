import expect, { createSpy, restoreSpies } from 'expect';
import { find } from 'lodash';

import { formatErrorResponse } from 'redux/nodes/entities/base/helpers';
import reduxConfig from 'redux/nodes/entities/base/reduxConfig';
import { reduxMockStore } from 'test/helpers';
import schemas from 'redux/nodes/entities/base/schemas';

const store = {
  entities: {
    invites: {
      errors: {},
      data: {},
      loading: false,
    },
    users: {
      errors: {},
      data: {},
      loading: false,
    },
  },
};
const invite = { id: 1, name: 'Gnar Dog', email: 'hi@thegnar.co' };
const user = { id: 1, email: 'hi@thegnar.co' };
const unauthenticatedError = {
  status: 401,
  message: {
    message: 'Unauthenticated',
    errors: [{ base: 'User is not authenticated' }],
  },
};

describe('reduxConfig', () => {
  afterEach(restoreSpies);

  describe('dispatching the clear errors action', () => {
    it('sets the errors to an empty object', () => {
      const initialState = {
        data: {},
        errors: {
          base: 'Unable to find the current user',
        },
        loading: false,
      };

      const config = reduxConfig({
        entityName: 'users',
        schema: schemas.USERS,
      });
      const { actions, reducer } = config;
      const newState = reducer(initialState, actions.clearErrors);

      expect(newState.errors).toEqual({});
    });
  });

  describe('dispatching the create action', () => {
    describe('successful create call', () => {
      const mockStore = reduxMockStore(store);
      const createFunc = createSpy().andCall(() => {
        return Promise.resolve([user]);
      });

      const config = reduxConfig({
        createFunc,
        entityName: 'users',
        schema: schemas.USERS,
      });
      const { actions, reducer } = config;

      it('calls the createFunc', () => {
        mockStore.dispatch(actions.create());

        expect(createFunc).toHaveBeenCalled();
      });

      it('dispatches the correct actions', () => {
        mockStore.dispatch(actions.create());

        const dispatchedActions = mockStore.getActions();
        const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });

        expect(dispatchedActionTypes).toInclude('users_CREATE_REQUEST');
        expect(dispatchedActionTypes).toInclude('users_CREATE_SUCCESS');
        expect(dispatchedActionTypes).toNotInclude('users_CREATE_FAILURE');
      });

      it('adds the returned user to state', () => {
        const createSuccessAction = {
          type: 'users_CREATE_SUCCESS',
          payload: {
            data: {
              users: {
                [user.id]: user,
              },
            },
          },
        };
        const initialState = {
          loading: false,
          entities: {},
          errors: {},
        };
        const newState = reducer(initialState, createSuccessAction);

        expect(newState.data[user.id]).toEqual(user);
      });
    });

    describe('unsuccessful create call', () => {
      const mockStore = reduxMockStore(store);

      describe('unauthenticated error', () => {
        const createFunc = createSpy().andCall(() => Promise.reject(unauthenticatedError));
        const config = reduxConfig({
          createFunc,
          entityName: 'users',
          schema: schemas.USERS,
        });
        const { actions } = config;

        it('dispatches the LOGOUT_SUCCESS action', (done) => {
          mockStore.dispatch(actions.create())
            .then(done)
            .catch(() => {
              const dispatchedActions = mockStore.getActions();

              expect(dispatchedActions).toInclude({ type: 'LOGOUT_SUCCESS' });
              done();
            });
        });
      });

      const errors = [
        { name: 'first_name',
          reason: 'is not valid',
        },
        { name: 'last_name',
          reason: 'must be changed or something',
        },
      ];
      const errorResponse = {
        message: {
          message: 'Validation Failed',
          errors,
        },
      };
      const formattedErrors = formatErrorResponse(errorResponse);
      const createFunc = createSpy().andCall(() => {
        return Promise.reject(errorResponse);
      });
      const config = reduxConfig({
        createFunc,
        entityName: 'users',
        schema: schemas.USERS,
      });
      const { actions, reducer } = config;

      it('calls the createFunc', () => {
        mockStore.dispatch(actions.create())
          .catch(() => false);

        expect(createFunc).toHaveBeenCalled();
      });

      it('dispatches the correct actions', () => {
        mockStore.dispatch(actions.create())
          .catch(() => false);

        const dispatchedActions = mockStore.getActions();
        const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });

        expect(dispatchedActionTypes).toInclude('users_CREATE_REQUEST');
        expect(dispatchedActionTypes).toNotInclude('users_CREATE_SUCCESS');

        const createFailureAction = find(dispatchedActions, { type: 'users_CREATE_FAILURE' });

        expect(createFailureAction.payload).toEqual({
          errors: formattedErrors,
        });
      });

      it('adds the returned errors to state', () => {
        const createFailureAction = {
          type: 'users_CREATE_FAILURE',
          payload: {
            errors: formattedErrors,
          },
        };
        const initialState = {
          loading: false,
          entities: {},
          errors: {},
        };
        const newState = reducer(initialState, createFailureAction);

        expect(newState.errors).toEqual(formattedErrors);
      });
    });
  });

  describe('#silentCreate action', () => {
    describe('successful create call', () => {
      const mockStore = reduxMockStore(store);
      const createFunc = createSpy().andReturn(Promise.resolve([user]));

      const config = reduxConfig({
        createFunc,
        entityName: 'users',
        schema: schemas.USERS,
      });
      const { actions } = config;

      it('dispatches the correct actions', (done) => {
        mockStore.dispatch(actions.silentCreate())
          .then(() => {
            const dispatchedActions = mockStore.getActions();
            const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });

            expect(dispatchedActionTypes).toInclude('users_CREATE_SUCCESS');
            expect(dispatchedActionTypes).toNotInclude('users_CREATE_REQUEST');
            expect(dispatchedActionTypes).toNotInclude('users_CREATE_FAILURE');
            done();
          })
          .catch(done);
      });
    });

    describe('unsuccessful create call', () => {
      const mockStore = reduxMockStore(store);
      const errors = [
        { name: 'first_name',
          reason: 'is not valid',
        },
        { name: 'last_name',
          reason: 'must be changed or something',
        },
      ];
      const errorResponse = {
        message: {
          message: 'Validation Failed',
          errors,
        },
      };
      const formattedErrors = formatErrorResponse(errorResponse);
      const createFunc = createSpy().andReturn(Promise.reject(errorResponse));
      const config = reduxConfig({
        createFunc,
        entityName: 'users',
        schema: schemas.USERS,
      });
      const { actions } = config;

      it('dispatches the correct actions', (done) => {
        mockStore.dispatch(actions.silentCreate())
          .then(done)
          .catch(() => {
            const dispatchedActions = mockStore.getActions();
            const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });

            expect(dispatchedActionTypes).toNotInclude('users_CREATE_REQUEST');
            expect(dispatchedActionTypes).toNotInclude('users_CREATE_SUCCESS');

            const createFailureAction = find(dispatchedActions, { type: 'users_CREATE_FAILURE' });

            expect(createFailureAction.payload).toEqual({
              errors: formattedErrors,
            });

            done();
          });
      });
    });
  });

  describe('dispatching the update action', () => {
    describe('successful update call', () => {
      const mockStore = reduxMockStore(store);
      const updateFunc = createSpy().andCall(() => {
        return Promise.resolve([{ ...user, updated: true }]);
      });

      const config = reduxConfig({
        updateFunc,
        entityName: 'users',
        schema: schemas.USERS,
      });
      const { actions, reducer } = config;

      it('calls the updateFunc', () => {
        mockStore.dispatch(actions.update(user));

        expect(updateFunc).toHaveBeenCalledWith(user);
      });

      it('dispatches the correct actions', () => {
        mockStore.dispatch(actions.update());

        const dispatchedActions = mockStore.getActions();
        const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });

        expect(dispatchedActionTypes).toInclude('users_UPDATE_REQUEST');
        expect(dispatchedActionTypes).toInclude('users_UPDATE_SUCCESS');
        expect(dispatchedActionTypes).toNotInclude('users_UPDATE_FAILURE');
      });

      it('adds the returned user to state', () => {
        const updateSuccessAction = {
          type: 'users_UPDATE_SUCCESS',
          payload: {
            data: {
              users: {
                [user.id]: { ...user, updated: true },
              },
            },
          },
        };
        const initialState = {
          loading: false,
          entities: {},
          errors: {},
        };
        const newState = reducer(initialState, updateSuccessAction);

        expect(newState.data[user.id]).toEqual({ ...user, updated: true });
      });
    });

    describe('unsuccessful update call', () => {
      describe('unauthenticated error', () => {
        const mockStore = reduxMockStore(store);
        const updateFunc = createSpy().andCall(() => Promise.reject(unauthenticatedError));
        const config = reduxConfig({ updateFunc, entityName: 'users', schema: schemas.USERS });
        const { actions } = config;

        it('dispatches the LOGOUT_SUCCESS action', (done) => {
          mockStore.dispatch(actions.update())
            .then(done)
            .catch(() => {
              const dispatchedActions = mockStore.getActions();

              expect(dispatchedActions).toInclude({ type: 'LOGOUT_SUCCESS' });

              done();
            });
        });
      });

      describe('unprocessable entity', () => {
        const mockStore = reduxMockStore(store);
        const errors = [
          { name: 'first_name',
            reason: 'is not valid',
          },
          { name: 'last_name',
            reason: 'must be changed or something',
          },
        ];
        const errorResponse = {
          status: 422,
          message: {
            message: 'Validation Failed',
            errors,
          },
        };
        const formattedErrors = formatErrorResponse(errorResponse);
        const updateFunc = createSpy().andCall(() => {
          return Promise.reject(errorResponse);
        });
        const config = reduxConfig({
          entityName: 'users',
          schema: schemas.USERS,
          updateFunc,
        });
        const { actions, reducer } = config;

        it('calls the updateFunc', () => {
          mockStore.dispatch(actions.update(user))
            .catch(() => false);

          expect(updateFunc).toHaveBeenCalledWith(user);
        });

        it('dispatches the correct actions', () => {
          mockStore.dispatch(actions.update())
            .catch(() => false);

          const dispatchedActions = mockStore.getActions();
          const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });

          expect(dispatchedActionTypes).toInclude('users_UPDATE_REQUEST');
          expect(dispatchedActionTypes).toNotInclude('users_UPDATE_SUCCESS');

          const updateFailureAction = find(dispatchedActions, { type: 'users_UPDATE_FAILURE' });

          expect(updateFailureAction.payload).toEqual({
            errors: formattedErrors,
          });
        });

        it('adds the returned errors to state', () => {
          const updateFailureAction = {
            type: 'users_UPDATE_FAILURE',
            payload: {
              errors: formattedErrors,
            },
          };
          const initialState = {
            loading: false,
            entities: {},
            errors: {},
          };
          const newState = reducer(initialState, updateFailureAction);

          expect(newState.errors).toEqual(formattedErrors);
        });
      });
    });
  });

  describe('#silentUpdate', () => {
    describe('successful call', () => {
      const mockStore = reduxMockStore(store);
      const updateFunc = createSpy().andReturn(Promise.resolve([{ ...user, updated: true }]));
      const config = reduxConfig({
        updateFunc,
        entityName: 'users',
        schema: schemas.USERS,
      });
      const { actions } = config;

      it('dispatches the correct actions', (done) => {
        mockStore.dispatch(actions.silentUpdate())
          .then(() => {
            const dispatchedActions = mockStore.getActions();
            const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });

            expect(dispatchedActionTypes).toNotInclude('users_UPDATE_REQUEST');
            expect(dispatchedActionTypes).toInclude('users_UPDATE_SUCCESS');
            expect(dispatchedActionTypes).toNotInclude('users_UPDATE_FAILURE');

            done();
          })
          .catch(done);
      });
    });

    describe('unsuccessful call', () => {
      describe('unprocessable entitiy', () => {
        const mockStore = reduxMockStore(store);

        const errors = [
          { name: 'first_name',
            reason: 'is not valid',
          },
          { name: 'last_name',
            reason: 'must be changed or something',
          },
        ];
        const errorResponse = {
          status: 422,
          message: {
            message: 'Validation Failed',
            errors,
          },
        };
        const formattedErrors = formatErrorResponse(errorResponse);
        const updateFunc = createSpy().andReturn(Promise.reject(errorResponse));
        const config = reduxConfig({
          entityName: 'users',
          schema: schemas.USERS,
          updateFunc,
        });
        const { actions } = config;

        it('dispatches the correct actions', (done) => {
          mockStore.dispatch(actions.silentUpdate())
            .then(done)
            .catch(() => {
              const dispatchedActions = mockStore.getActions();
              const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });

              expect(dispatchedActionTypes).toNotInclude('users_UPDATE_REQUEST');
              expect(dispatchedActionTypes).toNotInclude('users_UPDATE_SUCCESS');

              const updateFailureAction = find(dispatchedActions, { type: 'users_UPDATE_FAILURE' });

              expect(updateFailureAction.payload).toEqual({
                errors: formattedErrors,
              });

              done();
            });
        });
      });
    });
  });

  describe('dispatching the destroy action', () => {
    describe('successful destroy call', () => {
      const mockStore = reduxMockStore(store);
      const destroyFunc = createSpy().andCall(() => {
        return Promise.resolve();
      });

      const config = reduxConfig({
        destroyFunc,
        entityName: 'invites',
        schema: schemas.INVITES,
      });
      const { actions, reducer } = config;

      it('calls the destroyFunc', () => {
        mockStore.dispatch(actions.destroy({ inviteID: invite.id }));

        expect(destroyFunc).toHaveBeenCalled();
      });

      it('dispatches the correct actions', () => {
        mockStore.dispatch(actions.destroy({ inviteID: invite.id }));

        const dispatchedActions = mockStore.getActions();
        const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });

        expect(dispatchedActionTypes).toInclude('invites_DESTROY_REQUEST');
        expect(dispatchedActionTypes).toInclude('invites_DESTROY_SUCCESS');
        expect(dispatchedActionTypes).toNotInclude('invites_DESTROY_FAILURE');
      });

      it('removes the returned invite from state', () => {
        const destroySuccessAction = {
          type: 'invites_DESTROY_SUCCESS',
          payload: {
            id: 1,
          },
        };
        const initialState = {
          data: {
            [invite.id]: invite,
            2: { id: 2, name: 'Jason Meller' },
          },
          errors: {},
          loading: false,
        };
        const newState = reducer(initialState, destroySuccessAction);

        expect(newState.data).toEqual({
          2: { id: 2, name: 'Jason Meller' },
        });
      });
    });

    describe('unsuccessful destroy call', () => {
      const mockStore = reduxMockStore(store);
      const errors = [
        {
          name: 'base',
          reason: 'Unable to create user',
        },
      ];
      const errorResponse = {
        message: {
          message: 'Validation Failed',
          errors,
        },
      };
      const destroyFunc = createSpy().andCall(() => {
        return Promise.reject(errorResponse);
      });
      const formattedErrors = formatErrorResponse(errorResponse);
      const config = reduxConfig({
        destroyFunc,
        entityName: 'users',
        schema: schemas.USERS,
      });
      const { actions, reducer } = config;

      it('calls the createFunc', () => {
        mockStore.dispatch(actions.destroy())
          .catch(() => false);

        expect(destroyFunc).toHaveBeenCalled();
      });

      it('dispatches the correct actions', () => {
        mockStore.dispatch(actions.destroy())
          .catch(() => false);

        const dispatchedActions = mockStore.getActions();
        const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });
        const destroyFailureAction = find(dispatchedActions, { type: 'users_DESTROY_FAILURE' });

        expect(dispatchedActionTypes).toInclude('users_DESTROY_REQUEST');
        expect(dispatchedActionTypes).toNotInclude('users_DESTROY_SUCCESS');

        expect(destroyFailureAction.payload).toEqual({
          errors: formattedErrors,
        });
      });

      it('adds the returned errors to state', () => {
        const destroyFailureAction = {
          type: 'users_DESTROY_FAILURE',
          payload: {
            errors: formattedErrors,
          },
        };
        const initialState = {
          loading: false,
          entities: {},
          errors: {},
        };
        const newState = reducer(initialState, destroyFailureAction);

        expect(newState.errors).toEqual(formattedErrors);
      });
    });
  });

  describe('#silentDestroy', () => {
    describe('successful call', () => {
      const mockStore = reduxMockStore(store);
      const destroyFunc = createSpy().andReturn(Promise.resolve());
      const config = reduxConfig({
        destroyFunc,
        entityName: 'invites',
        schema: schemas.INVITES,
      });
      const { actions } = config;

      it('dispatches the correct actions', (done) => {
        mockStore.dispatch(actions.silentDestroy({ inviteID: invite.id }))
          .then(() => {
            const dispatchedActions = mockStore.getActions();
            const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });

            expect(dispatchedActionTypes).toNotInclude('invites_DESTROY_REQUEST');
            expect(dispatchedActionTypes).toInclude('invites_DESTROY_SUCCESS');
            expect(dispatchedActionTypes).toNotInclude('invites_DESTROY_FAILURE');

            done();
          })
          .catch(done);
      });
    });

    describe('unsuccessful call', () => {
      const mockStore = reduxMockStore(store);
      const errors = [
        {
          name: 'base',
          reason: 'Unable to create user',
        },
      ];
      const errorResponse = {
        message: {
          message: 'Validation Failed',
          errors,
        },
      };
      const destroyFunc = createSpy().andReturn(Promise.reject(errorResponse));
      const formattedErrors = formatErrorResponse(errorResponse);
      const config = reduxConfig({
        destroyFunc,
        entityName: 'users',
        schema: schemas.USERS,
      });
      const { actions } = config;

      it('dispatches the correct actions', (done) => {
        mockStore.dispatch(actions.silentDestroy())
          .then(done)
          .catch(() => {
            const dispatchedActions = mockStore.getActions();
            const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });
            const destroyFailureAction = find(dispatchedActions, { type: 'users_DESTROY_FAILURE' });

            expect(dispatchedActionTypes).toNotInclude('users_DESTROY_REQUEST');
            expect(dispatchedActionTypes).toNotInclude('users_DESTROY_SUCCESS');

            expect(destroyFailureAction.payload).toEqual({
              errors: formattedErrors,
            });

            done();
          });
      });
    });
  });

  describe('dispatching the load action', () => {
    describe('successful load call', () => {
      const mockStore = reduxMockStore(store);
      const loadFunc = createSpy().andCall(() => {
        return Promise.resolve(user);
      });

      const config = reduxConfig({
        entityName: 'users',
        loadFunc,
        schema: schemas.USERS,
      });
      const { actions, reducer } = config;

      it('calls the loadFunc', () => {
        mockStore.dispatch(actions.load());

        expect(loadFunc).toHaveBeenCalled();
      });

      it('dispatches the correct actions', () => {
        mockStore.dispatch(actions.load());

        const dispatchedActions = mockStore.getActions();
        const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });

        expect(dispatchedActionTypes).toInclude('users_LOAD_REQUEST');
        expect(dispatchedActionTypes).toInclude('users_LOAD_SUCCESS');
        expect(dispatchedActionTypes).toNotInclude('users_LOAD_FAILURE');
      });

      it('adds the returned user to state', () => {
        const loadSuccessAction = {
          type: 'users_LOAD_SUCCESS',
          payload: {
            data: {
              users: {
                [user.id]: user,
              },
            },
          },
        };
        const initialState = {
          loading: false,
          entities: {},
          errors: {},
        };
        const newState = reducer(initialState, loadSuccessAction);

        expect(newState.data[user.id]).toEqual(user);
      });
    });

    describe('unsuccessful load call', () => {
      const mockStore = reduxMockStore(store);
      const errors = [
        {
          name: 'base',
          reason: 'Unable to load users',
        },
      ];
      const errorResponse = {
        message: {
          message: 'Something went wrong',
          errors,
        },
      };
      const formattedErrors = formatErrorResponse(errorResponse);
      const loadFunc = createSpy().andCall(() => {
        return Promise.reject(errorResponse);
      });
      const config = reduxConfig({
        entityName: 'users',
        loadFunc,
        schema: schemas.USERS,
      });
      const { actions, reducer } = config;

      it('calls the loadFunc', () => {
        mockStore.dispatch(actions.load())
          .catch(() => false);

        expect(loadFunc).toHaveBeenCalled();
      });

      it('dispatches the correct actions', () => {
        mockStore.dispatch(actions.load())
          .catch(() => false);

        const dispatchedActions = mockStore.getActions();
        const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });
        const loadFailureAction = find(dispatchedActions, { type: 'users_LOAD_FAILURE' });

        expect(dispatchedActionTypes).toInclude('users_LOAD_REQUEST');
        expect(dispatchedActionTypes).toNotInclude('users_LOAD_SUCCESS');
        expect(loadFailureAction.payload).toEqual({
          errors: formattedErrors,
        });
      });

      it('adds the returned errors to state', () => {
        const loadFailureAction = {
          type: 'users_LOAD_FAILURE',
          payload: {
            errors: formattedErrors,
          },
        };
        const initialState = {
          loading: false,
          entities: {},
          errors: {},
        };
        const newState = reducer(initialState, loadFailureAction);

        expect(newState.errors).toEqual(formattedErrors);
      });
    });
  });

  describe('dispatching the loadAll action', () => {
    describe('successful load call', () => {
      const mockStore = reduxMockStore(store);
      const loadAllFunc = createSpy().andCall(() => {
        return Promise.resolve([user]);
      });

      const config = reduxConfig({
        entityName: 'users',
        loadAllFunc,
        schema: schemas.USERS,
      });
      const { actions, reducer } = config;

      it('calls the loadAllFunc', () => {
        mockStore.dispatch(actions.loadAll());

        expect(loadAllFunc).toHaveBeenCalled();
      });

      it('dispatches the correct actions', () => {
        mockStore.dispatch(actions.loadAll());

        const dispatchedActions = mockStore.getActions();
        const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });

        expect(dispatchedActionTypes).toInclude('users_LOAD_REQUEST');
        expect(dispatchedActionTypes).toNotInclude('users_LOAD_SUCCESS');
        expect(dispatchedActionTypes).toInclude('users_LOAD_ALL_SUCCESS');
        expect(dispatchedActionTypes).toNotInclude('users_LOAD_FAILURE');
      });

      it('adds the returned user to state', () => {
        const loadSuccessAction = {
          type: 'users_LOAD_SUCCESS',
          payload: {
            data: {
              users: {
                [user.id]: user,
              },
            },
          },
        };
        const initialState = {
          loading: false,
          entities: {},
          errors: {},
        };
        const newState = reducer(initialState, loadSuccessAction);

        expect(newState.data[user.id]).toEqual(user);
      });
    });

    describe('unsuccessful loadAll call', () => {
      const mockStore = reduxMockStore(store);
      const errors = [
        {
          name: 'base',
          reason: 'Unable to load users',
        },
      ];
      const errorResponse = {
        message: {
          message: 'Cannot get users',
          errors,
        },
      };
      const formattedErrors = formatErrorResponse(errorResponse);
      const loadAllFunc = createSpy().andCall(() => {
        return Promise.reject(errorResponse);
      });
      const config = reduxConfig({
        entityName: 'users',
        loadAllFunc,
        schema: schemas.USERS,
      });
      const { actions, reducer } = config;

      it('calls the loadAllFunc', () => {
        mockStore.dispatch(actions.loadAll())
          .catch(() => false);

        expect(loadAllFunc).toHaveBeenCalled();
      });

      it('dispatches the correct actions', () => {
        mockStore.dispatch(actions.loadAll())
          .catch(() => false);

        const dispatchedActions = mockStore.getActions();
        const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });
        const loadAllFailureAction = find(dispatchedActions, { type: 'users_LOAD_FAILURE' });

        expect(dispatchedActionTypes).toInclude('users_LOAD_REQUEST');
        expect(dispatchedActionTypes).toNotInclude('users_LOAD_SUCCESS');

        expect(loadAllFailureAction.payload).toEqual({
          errors: formattedErrors,
        });
      });

      it('adds the returned errors to state', () => {
        const loadAllFailureAction = {
          type: 'users_LOAD_FAILURE',
          payload: {
            errors: formattedErrors,
          },
        };
        const initialState = {
          loading: false,
          entities: {},
          errors: {},
        };
        const newState = reducer(initialState, loadAllFailureAction);

        expect(newState.errors).toEqual(formattedErrors);
      });
    });
  });

  describe('dispatching the silentLoadAll action', () => {
    describe('successful loadAll call', () => {
      const mockStore = reduxMockStore(store);
      const loadAllFunc = createSpy().andCall(() => {
        return Promise.resolve([user]);
      });

      const config = reduxConfig({
        entityName: 'users',
        loadAllFunc,
        schema: schemas.USERS,
      });
      const { actions } = config;

      it('calls the loadAllFunc', () => {
        mockStore.dispatch(actions.silentLoadAll());

        expect(loadAllFunc).toHaveBeenCalled();
      });

      it('dispatches the correct actions', () => {
        mockStore.dispatch(actions.silentLoadAll());

        const dispatchedActions = mockStore.getActions();
        const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });

        expect(dispatchedActionTypes).toEqual(['users_LOAD_ALL_SUCCESS']);
      });
    });

    describe('unsuccessful loadAll call', () => {
      const mockStore = reduxMockStore(store);
      const errors = [
        {
          name: 'base',
          reason: 'Unable to load users',
        },
      ];
      const errorResponse = {
        message: {
          message: 'Cannot get users',
          errors,
        },
      };
      const formattedErrors = formatErrorResponse(errorResponse);
      const loadAllFunc = createSpy().andCall(() => {
        return Promise.reject(errorResponse);
      });
      const config = reduxConfig({
        entityName: 'users',
        loadAllFunc,
        schema: schemas.USERS,
      });
      const { actions } = config;

      it('calls the loadAllFunc', () => {
        mockStore.dispatch(actions.silentLoadAll())
          .catch(() => false);

        expect(loadAllFunc).toHaveBeenCalled();
      });

      it('dispatches the correct actions', () => {
        mockStore.dispatch(actions.silentLoadAll())
          .catch(() => false);

        const dispatchedActions = mockStore.getActions();
        const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });
        const loadAllFailureAction = find(dispatchedActions, { type: 'users_LOAD_FAILURE' });

        expect(dispatchedActionTypes).toEqual(['users_LOAD_FAILURE']);

        expect(loadAllFailureAction.payload).toEqual({
          errors: formattedErrors,
        });
      });
    });
  });
});
