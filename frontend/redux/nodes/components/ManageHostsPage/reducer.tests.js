import expect from 'expect';

import { setDisplay } from './actions';
import reducer, { initialState } from './reducer';

describe('ManageHostsPage - reducer', () => {
  it('sets the initial state', () => {
    expect(reducer(undefined, { type: 'SOME_ACTION' })).toEqual(initialState);
  });

  describe('#setDisplay', () => {
    it('sets the display in state', () => {
      expect(reducer(initialState, setDisplay('List'))).toEqual({
        ...initialState,
        display: 'List',
      });
    });
  });
});
