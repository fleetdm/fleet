import reducer, { initialState } from "./reducer";
import {
  selectOsqueryTable,
  setQueryText,
  setSelectedTargets,
  setSelectedTargetsQuery,
} from "./actions";

describe("QueryPages - reducer", () => {
  it("sets the initial state", () => {
    expect(reducer(undefined, { type: "SOME_ACTION" })).toEqual(initialState);
  });

  describe("selectOsqueryTable action", () => {
    it("sets the selectedOsqueryTable attribute", () => {
      const selectOsqueryTableAction = selectOsqueryTable("groups");
      expect(reducer(initialState, selectOsqueryTableAction)).toEqual({
        queryText: initialState.queryText,
        selectedOsqueryTable:
          selectOsqueryTableAction.payload.selectedOsqueryTable,
        selectedTargets: [],
        selectedTargetsQuery: "",
      });
    });
  });

  describe("setQueryText action", () => {
    it("sets the queryText attribute", () => {
      const queryText = "SELECT * FROM users";
      const setQueryTextAction = setQueryText(queryText);
      expect(reducer(initialState, setQueryTextAction)).toEqual({
        queryText,
        selectedOsqueryTable: initialState.selectedOsqueryTable,
        selectedTargets: [],
        selectedTargetsQuery: "",
      });
    });
  });

  describe("setSelectedTargets action", () => {
    it("sets the selectedTarges attribute", () => {
      const selectedTargets = [{ label: "MacOs" }];
      const setSelectedTargetsAction = setSelectedTargets(selectedTargets);
      expect(reducer(initialState, setSelectedTargetsAction)).toEqual({
        ...initialState,
        selectedTargets,
      });
    });
  });

  describe("setSelectedTargetsQuery action", () => {
    it("sets the selectedTarges attribute", () => {
      const setSelectedTargetsQueryAction = setSelectedTargetsQuery("192");
      expect(reducer(initialState, setSelectedTargetsQueryAction)).toEqual({
        ...initialState,
        selectedTargetsQuery: "192",
      });
    });
  });
});
