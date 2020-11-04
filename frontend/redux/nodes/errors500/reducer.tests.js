import expect from 'expect';

import reducer from './reducer';

describe('Errors - reducer', () => {
  it('Updates state with errors object when an action that has a payload with an errors object is dispatched', () => {
    const payload = {
      errors: {
        base: "inserting pack: Error 1136: Column count doesn't match value count at row 1",
        http_status: 500,
      },
    };
    const packsCreateFailureAction = { type: 'packs_CREATE_FAILURE', payload };
    const initialState = {
      errors: null,
    };
    const newState = reducer(initialState, packsCreateFailureAction);

    expect(newState).toEqual({
      errors: {
        base: "inserting pack: Error 1136: Column count doesn't match value count at row 1",
        http_status: 500,
      },
    });
  });

  it('Updates state by setting errors to null when the RESET_ERRORS action is dipatched', () => {
    const errorsState = {
      errors: {
        base: "inserting pack: Error 1136: Column count doesn't match value count at row 1",
        http_status: 500,
      },
    };
    const newState = reducer(errorsState, { type: 'RESET_ERRORS' });
    expect(newState).toEqual({
      errors: null,
    });
  });
});
