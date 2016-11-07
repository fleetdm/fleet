import expect from 'expect';

import reducer, { initialState } from './reducer';
import {
  selectOsqueryTable,
  setQueryText,
  setSelectedTargets,
} from './actions';

describe('QueryPages - reducer', () => {
  it('sets the initial state', () => {
    expect(reducer(undefined, { type: 'SOME_ACTION' })).toEqual(initialState);
  });

  context('selectOsqueryTable action', () => {
    it('sets the selectedOsqueryTable attribute', () => {
      const selectOsqueryTableAction = selectOsqueryTable('groups');
      expect(reducer(initialState, selectOsqueryTableAction)).toEqual({
        queryText: initialState.queryText,
        selectedOsqueryTable: selectOsqueryTableAction.payload.selectedOsqueryTable,
        selectedTargets: [],
      });
    });
  });

  context('setQueryText action', () => {
    it('sets the queryText attribute', () => {
      const queryText = 'SELECT * FROM users';
      const setQueryTextAction = setQueryText(queryText);
      expect(reducer(initialState, setQueryTextAction)).toEqual({
        queryText,
        selectedOsqueryTable: initialState.selectedOsqueryTable,
        selectedTargets: [],
      });
    });
  });

  context('setSelectedTargets action', () => {
    it('sets the selectedTarges attribute', () => {
      const selectedTargets = [{ label: 'MacOs' }];
      const setSelectedTargetsAction = setSelectedTargets(selectedTargets);
      expect(reducer(initialState, setSelectedTargetsAction)).toEqual({
        ...initialState,
        selectedTargets,
      });
    });
  });
});

