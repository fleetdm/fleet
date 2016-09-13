import expect from 'expect';
import reducer, { initialState } from './reducer';
import { loginRequest } from './actions';

describe('Auth - reducer', () => {
  it('sets the initial state', () => {
    const state = reducer(undefined, { type: 'FOO' });

    expect(state).toEqual(initialState);
  });

  it('changes loading to true for the userLogin action', () => {
    const state = reducer(initialState, loginRequest);

    expect(state).toEqual({
      ...initialState,
      loading: true,
    });
  });
});
