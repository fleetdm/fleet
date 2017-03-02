import expect, { createSpy, restoreSpies } from 'expect';
import { find } from 'lodash';

import ReduxConfig from 'redux/nodes/entities/base/config';
import { reduxMockStore } from 'test/helpers';
import schemas from 'redux/nodes/entities/base/schemas';
import { userStub } from 'test/stubs';

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
const standardError = {
  status: 422,
  message: {
    message: 'Unauthenticated',
    errors: [
      {
        name: 'base',
        reason: 'User is not authenticated',
      },
    ],
  },
};

describe('ReduxConfig - thunks', () => {
  describe('#create', () => {
    afterEach(() => restoreSpies());

    const apiResponse = [userStub];

    describe('successful call', () => {
      const createFunc = createSpy()
        .andCall(() => Promise.resolve(apiResponse));
      const config = new ReduxConfig({
        createFunc,
        entityName: 'users',
        schema: schemas.USERS,
      });
      const mockStore = reduxMockStore(store);

      it('calls the createFunc', () => {
        const params = { first_name: 'Mike' };
        return mockStore.dispatch(config.actions.create(params))
        .then(() => {
          expect(createFunc).toHaveBeenCalledWith(params);
        });
      });

      it('dispatches the correct actions', () => {
        return mockStore.dispatch(config.actions.create())
          .then(() => {
            const dispatchedActions = mockStore.getActions();
            const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });
            const successAction = find(dispatchedActions, { type: 'users_CREATE_SUCCESS' });

            expect(dispatchedActionTypes).toInclude('users_CREATE_REQUEST');
            expect(dispatchedActionTypes).toInclude('users_CREATE_SUCCESS');
            expect(dispatchedActionTypes).toNotInclude('users_CREATE_FAILURE');
            expect(successAction.payload).toEqual({
              data: {
                users: {
                  [userStub.id]: userStub,
                },
              },
            });
          });
      });
    });

    describe('unsuccessful call', () => {
      const createFunc = createSpy()
        .andCall(() => Promise.reject(standardError));
      const config = new ReduxConfig({
        createFunc,
        entityName: 'users',
        schema: schemas.USERS,
      });

      it('calls the createFunc', () => {
        const mockStore = reduxMockStore(store);
        const params = { first_name: 'Mike' };

        return mockStore.dispatch(config.actions.create(params))
          .catch(() => {
            expect(createFunc).toHaveBeenCalledWith(params);
          });
      });

      it('dispatches the correct actions', () => {
        const mockStore = reduxMockStore(store);

        return mockStore.dispatch(config.actions.create())
          .catch(() => {
            const dispatchedActions = mockStore.getActions();
            const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });

            expect(dispatchedActionTypes).toInclude('users_CREATE_REQUEST');
            expect(dispatchedActionTypes).toInclude('users_CREATE_FAILURE');
            expect(dispatchedActionTypes).toNotInclude('users_CREATE_SUCCESS');

            const createFailureAction = find(dispatchedActions, { type: 'users_CREATE_FAILURE' });

            expect(createFailureAction.payload).toEqual({
              errors: {
                http_status: 422,
                base: 'User is not authenticated',
              },
            });
          });
      });
    });
  });

  describe('#destroy', () => {
    afterEach(() => restoreSpies());

    describe('successful call', () => {
      const destroyFunc = createSpy()
        .andCall(() => Promise.resolve());
      const config = new ReduxConfig({
        destroyFunc,
        entityName: 'users',
        schema: schemas.USERS,
      });
      const mockStore = reduxMockStore(store);

      it('calls the destroyFunc', () => {
        const params = { id: 1 };

        return mockStore.dispatch(config.actions.destroy(params))
        .then(() => {
          expect(destroyFunc).toHaveBeenCalledWith(params);
        });
      });

      it('dispatches the correct actions', () => {
        return mockStore.dispatch(config.actions.destroy())
          .then(() => {
            const dispatchedActions = mockStore.getActions();
            const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });

            expect(dispatchedActionTypes).toInclude('users_DESTROY_REQUEST');
            expect(dispatchedActionTypes).toInclude('users_DESTROY_SUCCESS');
            expect(dispatchedActionTypes).toNotInclude('users_DESTROY_FAILURE');
          });
      });
    });

    describe('unsuccessful call', () => {
      const destroyFunc = createSpy()
        .andCall(() => Promise.reject(standardError));
      const config = new ReduxConfig({
        destroyFunc,
        entityName: 'users',
        schema: schemas.USERS,
      });

      it('calls the destroyFunc', () => {
        const mockStore = reduxMockStore(store);
        const params = { id: 1 };

        return mockStore.dispatch(config.actions.destroy(params))
          .catch(() => {
            expect(destroyFunc).toHaveBeenCalledWith(params);
          });
      });

      it('dispatches the correct actions', () => {
        const mockStore = reduxMockStore(store);

        return mockStore.dispatch(config.actions.destroy())
          .catch(() => {
            const dispatchedActions = mockStore.getActions();
            const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });

            expect(dispatchedActionTypes).toInclude('users_DESTROY_REQUEST');
            expect(dispatchedActionTypes).toInclude('users_DESTROY_FAILURE');
            expect(dispatchedActionTypes).toNotInclude('users_DESTROY_SUCCESS');

            const destroyFailureAction = find(dispatchedActions, { type: 'users_DESTROY_FAILURE' });

            expect(destroyFailureAction.payload).toEqual({
              errors: {
                http_status: 422,
                base: 'User is not authenticated',
              },
            });
          });
      });
    });
  });

  describe('#load', () => {
    afterEach(() => restoreSpies());

    describe('successful call', () => {
      const loadFunc = createSpy()
        .andCall(() => Promise.resolve());
      const config = new ReduxConfig({
        entityName: 'users',
        loadFunc,
        schema: schemas.USERS,
      });
      const mockStore = reduxMockStore(store);

      it('calls the loadFunc', () => {
        const params = { id: 1 };

        return mockStore.dispatch(config.actions.load(params))
        .then(() => {
          expect(loadFunc).toHaveBeenCalledWith(params);
        });
      });

      it('dispatches the correct actions', () => {
        return mockStore.dispatch(config.actions.load())
          .then(() => {
            const dispatchedActions = mockStore.getActions();
            const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });

            expect(dispatchedActionTypes).toInclude('users_LOAD_REQUEST');
            expect(dispatchedActionTypes).toInclude('users_LOAD_SUCCESS');
            expect(dispatchedActionTypes).toNotInclude('users_LOAD_FAILURE');
          });
      });
    });

    describe('unsuccessful call', () => {
      const loadFunc = createSpy()
        .andCall(() => Promise.reject(standardError));
      const config = new ReduxConfig({
        entityName: 'users',
        loadFunc,
        schema: schemas.USERS,
      });

      it('calls the loadFunc', () => {
        const mockStore = reduxMockStore(store);
        const params = { id: 1 };

        return mockStore.dispatch(config.actions.load(params))
          .catch(() => {
            expect(loadFunc).toHaveBeenCalledWith(params);
          });
      });

      it('dispatches the correct actions', () => {
        const mockStore = reduxMockStore(store);

        return mockStore.dispatch(config.actions.load())
          .catch(() => {
            const dispatchedActions = mockStore.getActions();
            const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });

            expect(dispatchedActionTypes).toInclude('users_LOAD_REQUEST');
            expect(dispatchedActionTypes).toInclude('users_LOAD_FAILURE');
            expect(dispatchedActionTypes).toNotInclude('users_LOAD_SUCCESS');

            const loadFailureAction = find(dispatchedActions, { type: 'users_LOAD_FAILURE' });

            expect(loadFailureAction.payload).toEqual({
              errors: {
                http_status: 422,
                base: 'User is not authenticated',
              },
            });
          });
      });
    });
  });

  describe('#loadAll', () => {
    afterEach(() => restoreSpies());

    describe('successful call', () => {
      const loadAllFunc = createSpy()
        .andCall(() => Promise.resolve());
      const config = new ReduxConfig({
        entityName: 'users',
        loadAllFunc,
        schema: schemas.USERS,
      });
      const mockStore = reduxMockStore(store);

      it('calls the loadAllFunc', () => {
        const params = { id: 1 };

        return mockStore.dispatch(config.actions.loadAll(params))
        .then(() => {
          expect(loadAllFunc).toHaveBeenCalledWith(params);
        });
      });

      it('dispatches the correct actions', () => {
        return mockStore.dispatch(config.actions.loadAll())
          .then(() => {
            const dispatchedActions = mockStore.getActions();
            const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });

            expect(dispatchedActionTypes).toInclude('users_LOAD_REQUEST');
            expect(dispatchedActionTypes).toInclude('users_LOAD_ALL_SUCCESS');
            expect(dispatchedActionTypes).toNotInclude('users_LOAD_FAILURE');
          });
      });
    });

    describe('unsuccessful call', () => {
      const loadAllFunc = createSpy()
        .andCall(() => Promise.reject(standardError));
      const config = new ReduxConfig({
        entityName: 'users',
        loadAllFunc,
        schema: schemas.USERS,
      });

      it('calls the loadAllFunc', () => {
        const mockStore = reduxMockStore(store);
        const params = { id: 1 };

        return mockStore.dispatch(config.actions.loadAll(params))
          .catch(() => {
            expect(loadAllFunc).toHaveBeenCalledWith(params);
          });
      });

      it('dispatches the correct actions', () => {
        const mockStore = reduxMockStore(store);

        return mockStore.dispatch(config.actions.loadAll())
          .catch(() => {
            const dispatchedActions = mockStore.getActions();
            const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });

            expect(dispatchedActionTypes).toInclude('users_LOAD_REQUEST');
            expect(dispatchedActionTypes).toInclude('users_LOAD_FAILURE');
            expect(dispatchedActionTypes).toNotInclude('users_LOAD_ALL_SUCCESS');

            const loadAllFailureAction = find(dispatchedActions, { type: 'users_LOAD_FAILURE' });

            expect(loadAllFailureAction.payload).toEqual({
              errors: {
                http_status: 422,
                base: 'User is not authenticated',
              },
            });
          });
      });
    });
  });

  describe('#update', () => {
    afterEach(() => restoreSpies());

    describe('successful call', () => {
      const updateFunc = createSpy()
        .andCall(() => Promise.resolve());
      const config = new ReduxConfig({
        entityName: 'users',
        schema: schemas.USERS,
        updateFunc,
      });
      const mockStore = reduxMockStore(store);

      it('calls the updateFunc', () => {
        const params = { id: 1 };

        return mockStore.dispatch(config.actions.update(params))
        .then(() => {
          expect(updateFunc).toHaveBeenCalledWith(params);
        });
      });

      it('dispatches the correct actions', () => {
        return mockStore.dispatch(config.actions.update())
          .then(() => {
            const dispatchedActions = mockStore.getActions();
            const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });

            expect(dispatchedActionTypes).toInclude('users_UPDATE_REQUEST');
            expect(dispatchedActionTypes).toInclude('users_UPDATE_SUCCESS');
            expect(dispatchedActionTypes).toNotInclude('users_UPDATE_FAILURE');
          });
      });
    });

    describe('unsuccessful call', () => {
      const updateFunc = createSpy()
        .andCall(() => Promise.reject(standardError));
      const config = new ReduxConfig({
        entityName: 'users',
        schema: schemas.USERS,
        updateFunc,
      });

      it('calls the updateFunc', () => {
        const mockStore = reduxMockStore(store);
        const params = { id: 1 };

        return mockStore.dispatch(config.actions.update(params))
          .catch(() => {
            expect(updateFunc).toHaveBeenCalledWith(params);
          });
      });

      it('dispatches the correct actions', () => {
        const mockStore = reduxMockStore(store);

        return mockStore.dispatch(config.actions.update())
          .catch(() => {
            const dispatchedActions = mockStore.getActions();
            const dispatchedActionTypes = dispatchedActions.map((action) => { return action.type; });

            expect(dispatchedActionTypes).toInclude('users_UPDATE_REQUEST');
            expect(dispatchedActionTypes).toInclude('users_UPDATE_FAILURE');
            expect(dispatchedActionTypes).toNotInclude('users_UPDATE_SUCCESS');

            const updateFailureAction = find(dispatchedActions, { type: 'users_UPDATE_FAILURE' });

            expect(updateFailureAction.payload).toEqual({
              errors: {
                http_status: 422,
                base: 'User is not authenticated',
              },
            });
          });
      });
    });
  });
});

