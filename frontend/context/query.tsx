import React, { createContext, useReducer, ReactNode } from "react";
import { find } from "lodash";

import { osqueryTables } from "utilities/osquery_tables";
import { DEFAULT_QUERY } from "utilities/constants";
import { DEFAULT_OSQUERY_TABLE, IOsQueryTable } from "interfaces/osquery_table";
import { SelectedPlatformString } from "interfaces/platform";
import { QueryLoggingOption } from "interfaces/schedulable_query";
import { DEFAULT_TARGETS, ITargets } from "interfaces/target";

type Props = {
  children: ReactNode;
};

type InitialStateType = {
  selectedOsqueryTable: IOsQueryTable;
  lastEditedQueryId: number | null;
  lastEditedQueryName: string;
  lastEditedQueryDescription: string;
  lastEditedQueryBody: string;
  lastEditedQueryObserverCanRun: boolean;
  lastEditedQueryFrequency: number;
  lastEditedQueryPlatforms: SelectedPlatformString;
  lastEditedQueryMinOsqueryVersion: string;
  lastEditedQueryLoggingType: QueryLoggingOption;
  lastEditedQueryTargets: ITargets;
  setLastEditedQueryId: (value: number | null) => void;
  setLastEditedQueryName: (value: string) => void;
  setLastEditedQueryDescription: (value: string) => void;
  setLastEditedQueryBody: (value: string) => void;
  setLastEditedQueryObserverCanRun: (value: boolean) => void;
  setLastEditedQueryFrequency: (value: number) => void;
  setLastEditedQueryPlatforms: (value: SelectedPlatformString) => void;
  setLastEditedQueryMinOsqueryVersion: (value: string) => void;
  setLastEditedQueryLoggingType: (value: string) => void;
  setSelectedOsqueryTable: (tableName: string) => void;
  setLastEditedQueryTargets: (value: ITargets) => void;
};

export type IQueryContext = InitialStateType;

const initialState = {
  selectedOsqueryTable:
    find(osqueryTables, { name: "users" }) || DEFAULT_OSQUERY_TABLE,
  lastEditedQueryId: null,
  lastEditedQueryName: DEFAULT_QUERY.name,
  lastEditedQueryDescription: DEFAULT_QUERY.description,
  lastEditedQueryBody: DEFAULT_QUERY.query,
  lastEditedQueryObserverCanRun: DEFAULT_QUERY.observer_can_run,
  lastEditedQueryFrequency: DEFAULT_QUERY.interval,
  lastEditedQueryPlatforms: DEFAULT_QUERY.platform,
  lastEditedQueryMinOsqueryVersion: DEFAULT_QUERY.min_osquery_version,
  lastEditedQueryLoggingType: DEFAULT_QUERY.logging,
  lastEditedQueryTargets: DEFAULT_TARGETS,
  setLastEditedQueryId: () => null,
  setLastEditedQueryName: () => null,
  setLastEditedQueryDescription: () => null,
  setLastEditedQueryBody: () => null,
  setLastEditedQueryObserverCanRun: () => null,
  setLastEditedQueryFrequency: () => null,
  setLastEditedQueryPlatforms: () => null,
  setLastEditedQueryMinOsqueryVersion: () => null,
  setLastEditedQueryLoggingType: () => null,
  setSelectedOsqueryTable: () => null,
  setLastEditedQueryTargets: () => null,
};

const actions = {
  SET_SELECTED_OSQUERY_TABLE: "SET_SELECTED_OSQUERY_TABLE",
  SET_LAST_EDITED_QUERY_INFO: "SET_LAST_EDITED_QUERY_INFO",
  SET_LAST_EDITED_QUERY_TARGETS: "SET_LAST_EDITED_QUERY_TARGETS",
} as const;

const reducer = (state: InitialStateType, action: any) => {
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
        lastEditedQueryFrequency:
          typeof action.lastEditedQueryFrequency === "undefined"
            ? state.lastEditedQueryFrequency
            : action.lastEditedQueryFrequency,
        lastEditedQueryPlatforms:
          typeof action.lastEditedQueryPlatforms === "undefined"
            ? state.lastEditedQueryPlatforms
            : action.lastEditedQueryPlatforms,
        lastEditedQueryMinOsqueryVersion:
          typeof action.lastEditedQueryMinOsqueryVersion === "undefined"
            ? state.lastEditedQueryMinOsqueryVersion
            : action.lastEditedQueryMinOsqueryVersion,
        lastEditedQueryLoggingType:
          typeof action.lastEditedQueryLoggingType === "undefined"
            ? state.lastEditedQueryLoggingType
            : action.lastEditedQueryLoggingType,
      };
    case actions.SET_LAST_EDITED_QUERY_TARGETS:
      return {
        ...state,
        lastEditedQueryTargets:
          typeof action.lastEditedQueryTargets === "undefined"
            ? state.lastEditedQueryTargets
            : action.lastEditedQueryTargets,
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
    lastEditedQueryId: state.lastEditedQueryId,
    lastEditedQueryName: state.lastEditedQueryName,
    lastEditedQueryDescription: state.lastEditedQueryDescription,
    lastEditedQueryBody: state.lastEditedQueryBody,
    lastEditedQueryObserverCanRun: state.lastEditedQueryObserverCanRun,
    lastEditedQueryFrequency: state.lastEditedQueryFrequency,
    lastEditedQueryPlatforms: state.lastEditedQueryPlatforms,
    lastEditedQueryMinOsqueryVersion: state.lastEditedQueryMinOsqueryVersion,
    lastEditedQueryLoggingType: state.lastEditedQueryLoggingType,
    lastEditedQueryTargets: state.lastEditedQueryTargets,
    setLastEditedQueryId: (lastEditedQueryId: number | null) => {
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
    setLastEditedQueryFrequency: (lastEditedQueryFrequency: number) => {
      dispatch({
        type: actions.SET_LAST_EDITED_QUERY_INFO,
        lastEditedQueryFrequency,
      });
    },
    setLastEditedQueryPlatforms: (lastEditedQueryPlatforms: string) => {
      dispatch({
        type: actions.SET_LAST_EDITED_QUERY_INFO,
        lastEditedQueryPlatforms,
      });
    },
    setLastEditedQueryMinOsqueryVersion: (
      lastEditedQueryMinOsqueryVersion: string
    ) => {
      dispatch({
        type: actions.SET_LAST_EDITED_QUERY_INFO,
        lastEditedQueryMinOsqueryVersion,
      });
    },
    setLastEditedQueryLoggingType: (lastEditedQueryLoggingType: string) => {
      dispatch({
        type: actions.SET_LAST_EDITED_QUERY_INFO,
        lastEditedQueryLoggingType,
      });
    },
    setLastEditedQueryTargets: (lastEditedQueryTargets: ITargets) => {
      dispatch({
        type: actions.SET_LAST_EDITED_QUERY_TARGETS,
        lastEditedQueryTargets,
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
