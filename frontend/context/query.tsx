import React, { createContext, useReducer, ReactNode } from "react";
import { find } from "lodash";

// @ts-ignore
import { osqueryTables } from "utilities/osquery_tables";
import { DEFAULT_QUERY } from "utilities/constants";
import { IOsqueryTable } from "interfaces/osquery_table";

type Props = {
  children: ReactNode;
};

type InitialStateType = {
  selectedOsqueryTable: IOsqueryTable;
  lastEditedQueryName: string;
  lastEditedQueryDescription: string;
  lastEditedQueryBody: string;
  lastEditedQueryObserverCanRun: boolean;
  setLastEditedQueryName: (value: string) => void;
  setLastEditedQueryDescription: (value: string) => void;
  setLastEditedQueryBody: (value: string) => void;
  setLastEditedQueryObserverCanRun: (value: boolean) => void;
  setSelectedOsqueryTable: (tableName: string) => void;
};

const initialState = {
  selectedOsqueryTable: find(osqueryTables, { name: "users" }),
  lastEditedQueryName: DEFAULT_QUERY.name,
  lastEditedQueryDescription: DEFAULT_QUERY.description,
  lastEditedQueryBody: DEFAULT_QUERY.query,
  lastEditedQueryObserverCanRun: DEFAULT_QUERY.observer_can_run,
  setLastEditedQueryName: () => null,
  setLastEditedQueryDescription: () => null,
  setLastEditedQueryBody: () => null,
  setLastEditedQueryObserverCanRun: () => null,
  setSelectedOsqueryTable: () => null,
};

const actions = {
  SET_SELECTED_OSQUERY_TABLE: "SET_SELECTED_OSQUERY_TABLE",
  SET_LAST_EDITED_QUERY_INFO: "SET_LAST_EDITED_QUERY_INFO",
};

const reducer = (state: any, action: any) => {
  switch (action.type) {
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
        lastEditedQueryObserverCanRun:
          typeof action.lastEditedQueryObserverCanRun === "undefined"
            ? state.lastEditedQueryObserverCanRun
            : action.lastEditedQueryObserverCanRun,
      };
    default:
      return state;
  }
};

export const QueryContext = createContext<InitialStateType>(initialState);

const QueryProvider = ({ children }: Props) => {
  const [state, dispatch] = useReducer(reducer, initialState);

  const value = {
    selectedOsqueryTable: state.selectedOsqueryTable,
    lastEditedQueryName: state.lastEditedQueryName,
    lastEditedQueryDescription: state.lastEditedQueryDescription,
    lastEditedQueryBody: state.lastEditedQueryBody,
    lastEditedQueryObserverCanRun: state.lastEditedQueryObserverCanRun,
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
    setLastEditedQueryObserverCanRun: (
      lastEditedQueryObserverCanRun: boolean
    ) => {
      dispatch({
        type: actions.SET_LAST_EDITED_QUERY_INFO,
        lastEditedQueryObserverCanRun,
      });
    },
    setSelectedOsqueryTable: (tableName: string) => {
      dispatch({ type: actions.SET_SELECTED_OSQUERY_TABLE, tableName });
    },
  };

  return (
    <QueryContext.Provider value={value}>{children}</QueryContext.Provider>
  );
};

export default QueryProvider;
