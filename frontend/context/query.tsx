import React, { createContext, useReducer, ReactNode } from "react";
import { find } from "lodash";

import { osqueryTables } from "utilities/osquery_tables";
import { DEFAULT_QUERY } from "utilities/constants";
import { DEFAULT_OSQUERY_TABLE, IOsQueryTable } from "interfaces/osquery_table";
import { CommaSeparatedPlatformString } from "interfaces/platform";
import { QueryLoggingOption } from "interfaces/schedulable_query";
import {
  DEFAULT_TARGETS,
  DEFAULT_TARGETS_BY_TYPE,
  ISelectedTargetsByType,
  ITarget,
} from "interfaces/target";

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
  lastEditedQueryAutomationsEnabled: boolean;
  lastEditedQueryPlatforms: CommaSeparatedPlatformString;
  lastEditedQueryMinOsqueryVersion: string;
  lastEditedQueryLoggingType: QueryLoggingOption;
  lastEditedQueryDiscardData: boolean;
  editingExistingQuery?: boolean;
  selectedQueryTargets: ITarget[]; // Mimicks old selectedQueryTargets still used for policies for SelectTargets.tsx and running a live query
  selectedQueryTargetsByType: ISelectedTargetsByType; // New format by type for cleaner app wide state
  setLastEditedQueryId: (value: number | null) => void;
  setLastEditedQueryName: (value: string) => void;
  setLastEditedQueryDescription: (value: string) => void;
  setLastEditedQueryBody: (value: string) => void;
  setLastEditedQueryObserverCanRun: (value: boolean) => void;
  setLastEditedQueryFrequency: (value: number) => void;
  setLastEditedQueryAutomationsEnabled: (value: boolean) => void;
  setLastEditedQueryPlatforms: (value: CommaSeparatedPlatformString) => void;
  setLastEditedQueryMinOsqueryVersion: (value: string) => void;
  setLastEditedQueryLoggingType: (value: string) => void;
  setLastEditedQueryDiscardData: (value: boolean) => void;
  setSelectedOsqueryTable: (tableName: string) => void;
  setSelectedQueryTargets: (value: ITarget[]) => void;
  setSelectedQueryTargetsByType: (value: ISelectedTargetsByType) => void;
  setEditingExistingQuery: (value: boolean) => void;
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
  lastEditedQueryAutomationsEnabled: DEFAULT_QUERY.automations_enabled,
  lastEditedQueryPlatforms: DEFAULT_QUERY.platform,
  lastEditedQueryMinOsqueryVersion: DEFAULT_QUERY.min_osquery_version,
  lastEditedQueryLoggingType: DEFAULT_QUERY.logging,
  lastEditedQueryDiscardData: DEFAULT_QUERY.discard_data,
  editingExistingQuery: DEFAULT_QUERY.editingExistingQuery ?? false,
  selectedQueryTargets: DEFAULT_TARGETS,
  selectedQueryTargetsByType: DEFAULT_TARGETS_BY_TYPE,
  setLastEditedQueryId: () => null,
  setLastEditedQueryName: () => null,
  setLastEditedQueryDescription: () => null,
  setLastEditedQueryBody: () => null,
  setLastEditedQueryObserverCanRun: () => null,
  setLastEditedQueryFrequency: () => null,
  setLastEditedQueryAutomationsEnabled: () => null,
  setLastEditedQueryPlatforms: () => null,
  setLastEditedQueryMinOsqueryVersion: () => null,
  setLastEditedQueryLoggingType: () => null,
  setLastEditedQueryDiscardData: () => null,
  setSelectedOsqueryTable: () => null,
  setSelectedQueryTargets: () => null,
  setSelectedQueryTargetsByType: () => null,
  setEditingExistingQuery: () => null,
};

const actions = {
  SET_SELECTED_OSQUERY_TABLE: "SET_SELECTED_OSQUERY_TABLE",
  SET_LAST_EDITED_QUERY_INFO: "SET_LAST_EDITED_QUERY_INFO",
  SET_SELECTED_QUERY_TARGETS: "SET_SELECTED_QUERY_TARGETS",
  SET_SELECTED_QUERY_TARGETS_BY_TYPE: "SET_SELECTED_QUERY_TARGETS_BY_TYPE",
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
        lastEditedQueryAutomationsEnabled:
          typeof action.lastEditedQueryAutomationsEnabled === "undefined"
            ? state.lastEditedQueryAutomationsEnabled
            : action.lastEditedQueryAutomationsEnabled,
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
        lastEditedQueryDiscardData:
          typeof action.lastEditedQueryDiscardData === "undefined"
            ? state.lastEditedQueryDiscardData
            : action.lastEditedQueryDiscardData,
        editingExistingQuery:
          typeof action.editingExistingQuery === "undefined"
            ? state.editingExistingQuery
            : action.editingExistingQuery,
      };
    case actions.SET_SELECTED_QUERY_TARGETS:
      return {
        ...state,
        selectedQueryTargets:
          typeof action.selectedQueryTargets === "undefined"
            ? state.selectedQueryTargets
            : action.selectedQueryTargets,
      };
    case actions.SET_SELECTED_QUERY_TARGETS_BY_TYPE:
      return {
        ...state,
        selectedQueryTargetsByType:
          typeof action.selectedQueryTargetsByType === "undefined"
            ? state.selectedQueryTargetsByType
            : action.selectedQueryTargetsByType,
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
    lastEditedQueryAutomationsEnabled: state.lastEditedQueryAutomationsEnabled,
    lastEditedQueryPlatforms: state.lastEditedQueryPlatforms,
    lastEditedQueryMinOsqueryVersion: state.lastEditedQueryMinOsqueryVersion,
    lastEditedQueryLoggingType: state.lastEditedQueryLoggingType,
    lastEditedQueryDiscardData: state.lastEditedQueryDiscardData,
    editingExistingQuery: state.editingExistingQuery,
    selectedQueryTargets: state.selectedQueryTargets,
    selectedQueryTargetsByType: state.selectedQueryTargetsByType,
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
    setLastEditedQueryAutomationsEnabled: (
      lastEditedQueryAutomationsEnabled: boolean
    ) => {
      dispatch({
        type: actions.SET_LAST_EDITED_QUERY_INFO,
        lastEditedQueryAutomationsEnabled,
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
    setLastEditedQueryDiscardData: (lastEditedQueryDiscardData: boolean) => {
      dispatch({
        type: actions.SET_LAST_EDITED_QUERY_INFO,
        lastEditedQueryDiscardData,
      });
    },
    setEditingExistingQuery: (editingExistingQuery: boolean) => {
      dispatch({
        type: actions.SET_LAST_EDITED_QUERY_INFO,
        editingExistingQuery,
      });
    },
    setSelectedQueryTargets: (selectedQueryTargets: ITarget[]) => {
      dispatch({
        type: actions.SET_SELECTED_QUERY_TARGETS,
        selectedQueryTargets,
      });
    },
    setSelectedQueryTargetsByType: (
      selectedQueryTargetsByType: ISelectedTargetsByType
    ) => {
      dispatch({
        type: actions.SET_SELECTED_QUERY_TARGETS_BY_TYPE,
        selectedQueryTargetsByType,
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
