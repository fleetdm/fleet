import React, { createContext, useReducer, ReactNode } from "react";
import { find } from "lodash";

// @ts-ignore
import { osqueryTables } from "utilities/osquery_tables";
import { DEFAULT_POLICY } from "utilities/constants";
import { IOsqueryTable } from "interfaces/osquery_table";

type Props = {
  children: ReactNode;
};

type InitialStateType = {
  lastEditedQueryName: string;
  lastEditedQueryDescription: string;
  lastEditedQueryBody: string;
  setLastEditedQueryName: (value: string) => void;
  setLastEditedQueryDescription: (value: string) => void;
  setLastEditedQueryBody: (value: string) => void;
  policyTeamId: number;
  setPolicyTeamId: (id: number) => void;
  selectedOsqueryTable: IOsqueryTable;
  setSelectedOsqueryTable: (tableName: string) => void;
};

const initialState = {
  lastEditedQueryName: DEFAULT_POLICY.name,
  lastEditedQueryDescription: DEFAULT_POLICY.description,
  lastEditedQueryBody: DEFAULT_POLICY.query,
  setLastEditedQueryName: () => null,
  setLastEditedQueryDescription: () => null,
  setLastEditedQueryBody: () => null,
  policyTeamId: 0,
  setPolicyTeamId: () => null,
  selectedOsqueryTable: find(osqueryTables, { name: "users" }),
  setSelectedOsqueryTable: () => null,
};

const actions = {
  SET_LAST_EDITED_QUERY_INFO: "SET_LAST_EDITED_QUERY_INFO",
  SET_POLICY_TEAM_ID: "SET_POLICY_TEAM_ID",
  SET_SELECTED_OSQUERY_TABLE: "SET_SELECTED_OSQUERY_TABLE",
};

const reducer = (state: any, action: any) => {
  switch (action.type) {
    case actions.SET_POLICY_TEAM_ID:
      return {
        ...state,
        policyTeamId: action.id,
      };
    case actions.SET_SELECTED_OSQUERY_TABLE:
      return {
        ...state,
        selectedOsqueryTable: find(osqueryTables, { name: action.tableName }),
      };
    case actions.SET_LAST_EDITED_QUERY_INFO:
      return {
        ...state,
        lastEditedQueryName:
          typeof action.lastEditedQueryName === "undefined"
            ? state.lastEditedQueryName
            : action.lastEditedQueryName,
        lastEditedQueryDescription:
          typeof action.lastEditedQueryDescription === "undefined"
            ? state.lastEditedQueryDescription
            : action.lastEditedQueryDescription,
        lastEditedQueryBody:
          typeof action.lastEditedQueryBody === "undefined"
            ? state.lastEditedQueryBody
            : action.lastEditedQueryBody,
      };
    default:
      return state;
  }
};

export const PolicyContext = createContext<InitialStateType>(initialState);

const PolicyProvider = ({ children }: Props) => {
  const [state, dispatch] = useReducer(reducer, initialState);

  const value = {
    lastEditedQueryName: state.lastEditedQueryName,
    lastEditedQueryDescription: state.lastEditedQueryDescription,
    lastEditedQueryBody: state.lastEditedQueryBody,
    setLastEditedQueryName: (lastEditedQueryName: string) => {
      dispatch({
        type: actions.SET_LAST_EDITED_QUERY_INFO,
        lastEditedQueryName,
      });
    },
    setLastEditedQueryDescription: (lastEditedQueryDescription: string) => {
      dispatch({
        type: actions.SET_LAST_EDITED_QUERY_INFO,
        lastEditedQueryDescription,
      });
    },
    setLastEditedQueryBody: (lastEditedQueryBody: string) => {
      dispatch({
        type: actions.SET_LAST_EDITED_QUERY_INFO,
        lastEditedQueryBody,
      });
    },
    policyTeamId: state.policyTeamId,
    setPolicyTeamId: (id: number) => {
      dispatch({ type: actions.SET_POLICY_TEAM_ID, id });
    },
    selectedOsqueryTable: state.selectedOsqueryTable,
    setSelectedOsqueryTable: (tableName: string) => {
      dispatch({ type: actions.SET_SELECTED_OSQUERY_TABLE, tableName });
    },
  };

  return (
    <PolicyContext.Provider value={value}>{children}</PolicyContext.Provider>
  );
};

export default PolicyProvider;
