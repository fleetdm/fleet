import expect, { createSpy, restoreSpies } from 'expect';
import reduxConfig from './reduxConfig';
import { reduxMockStore } from '../../../../test/helpers';
import schemas from './schemas';

const store = { entities: { users: {} } };
const user = { id: 1, email: 'hi@thegnar.co' };

describe('reduxConfig', () => {
  afterEach(restoreSpies);

  describe('dispatching the load action', () => {
    describe('successful load call', () => {
      const mockStore = reduxMockStore(store);
      const loadFunc = createSpy().andCall(() => {
        return Promise.resolve([user]);
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
        const dispatchedActionTypes = dispatchedActions.map(action => { return action.type; });

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
      const errors = { base: 'Unable to load users' };
      const loadFunc = createSpy().andCall(() => {
        return Promise.reject({ errors });
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
        const dispatchedActionTypes = dispatchedActions.map(action => { return action.type; });

        expect(dispatchedActionTypes).toInclude('users_LOAD_REQUEST');
        expect(dispatchedActionTypes).toNotInclude('users_LOAD_SUCCESS');
        expect(dispatchedActionTypes).toInclude('users_LOAD_FAILURE');
      });

      it('adds the returned errors to state', () => {
        const loadFailureAction = {
          type: 'users_LOAD_FAILURE',
          payload: {
            errors,
          },
        };
        const initialState = {
          loading: false,
          entities: {},
          errors: {},
        };
        const newState = reducer(initialState, loadFailureAction);

        expect(newState.errors).toEqual(errors);
      });
    });
  });
});
