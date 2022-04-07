import React, { createContext, useReducer, ReactNode } from "react";
import { find } from "lodash";

// @ts-ignore
import { osqueryTables } from "utilities/osquery_tables";
import { DEFAULT_QUERY } from "utilities/constants";
import { DEFAULT_OSQUERY_TABLE, IOsqueryTable } from "interfaces/osquery_table";
import { ISelectedTargets } from "interfaces/target";

type Props = {
  children: ReactNode;
};

type InitialStateType = {
  selectedTargetsByQueryId: Record<number, ISelectedTargets> | null;
  selectedOsqueryTable: IOsqueryTable;
  lastEditedQueryId: number | null;
  lastEditedQueryName: string;
  lastEditedQueryDescription: string;
  lastEditedQueryBody: string;
  lastEditedQueryObserverCanRun: boolean;
  setLastEditedQueryId: (value: number) => void;
  setLastEditedQueryName: (value: string) => void;
  setLastEditedQueryDescription: (value: string) => void;
  setLastEditedQueryBody: (value: string) => void;
  setLastEditedQueryObserverCanRun: (value: boolean) => void;
  setSelectedOsqueryTable: (tableName: string) => void;
  setSelectedTargetsByQueryId: (id: number, targets: ISelectedTargets) => void;
};

const initialState = {
  selectedOsqueryTable:
    find(osqueryTables, { name: "users" }) || DEFAULT_OSQUERY_TABLE,
  lastEditedQueryId: null,
  selectedTargetsByQueryId: null,
  lastEditedQueryName: DEFAULT_QUERY.name,
  lastEditedQueryDescription: DEFAULT_QUERY.description,
  lastEditedQueryBody: DEFAULT_QUERY.query,
  lastEditedQueryObserverCanRun: DEFAULT_QUERY.observer_can_run,
  setLastEditedQueryId: () => null,
  setLastEditedQueryName: () => null,
  setLastEditedQueryDescription: () => null,
  setLastEditedQueryBody: () => null,
  setLastEditedQueryObserverCanRun: () => null,
  setSelectedOsqueryTable: () => null,
  setSelectedTargetsByQueryId: () => null,
};

const actions = {
  SET_SELECTED_OSQUERY_TABLE: "SET_SELECTED_OSQUERY_TABLE",
  SET_SELECTED_TARGETS_BY_QUERY_ID: "SET_SELECTED_TARGETS_BY_QUERY_ID",
  SET_LAST_EDITED_QUERY_INFO: "SET_LAST_EDITED_QUERY_INFO",
} as const;

const reducer = (state: any, action: any) => {
  switch (action.type) {
    case actions.SET_SELECTED_OSQUERY_TABLE:
      return {
        ...state,
        selectedOsqueryTable:
          find(osqueryTables, { name: action.tableName }) ||
          DEFAULT_OSQUERY_TABLE,
      };
    case actions.SET_LAST_EDITED_QUERY_INFO:
      return {
        ...state,
        lastEditedQueryId:
          typeof action.lastEditedQueryId === "undefined"
            ? state.lastEditedQueryId
            : action.lastEditedQueryId,
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
    case actions.SET_SELECTED_TARGETS_BY_QUERY_ID:
      return {
        ...state,
        selectedTargetsByQueryId: {
          ...state.selectedTargetsByQueryID,
          [action.id]: action.targets,
        },
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
    selectedTargetsByQueryId: state.selectedTargetsByQueryId,
    lastEditedQueryId: state.lastEditedQueryId,
    lastEditedQueryName: state.lastEditedQueryName,
    lastEditedQueryDescription: state.lastEditedQueryDescription,
    lastEditedQueryBody: state.lastEditedQueryBody,
    lastEditedQueryObserverCanRun: state.lastEditedQueryObserverCanRun,
    setLastEditedQueryId: (lastEditedQueryId: number) => {
      dispatch({
        type: actions.SET_LAST_EDITED_QUERY_INFO,
        lastEditedQueryId,
      });
    },
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
    setSelectedTargetsByQueryId: (id: number, targets: ISelectedTargets) => {
      dispatch({ type: actions.SET_SELECTED_TARGETS_BY_QUERY_ID, id, targets });
    },
  };

  return (
    <QueryContext.Provider value={value}>{children}</QueryContext.Provider>
  );
};

export default QueryProvider;
