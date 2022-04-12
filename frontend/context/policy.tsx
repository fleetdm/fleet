import React, { createContext, useReducer, ReactNode } from "react";
import { find } from "lodash";

// @ts-ignore
import { osqueryTables } from "utilities/osquery_tables";
import { IOsqueryTable, DEFAULT_OSQUERY_TABLE } from "interfaces/osquery_table";
import { IPlatformString } from "interfaces/platform";

enum ACTIONS {
  SET_LAST_EDITED_QUERY_INFO = "SET_LAST_EDITED_QUERY_INFO",
  SET_POLICY_TEAM_ID = "SET_POLICY_TEAM_ID",
  SET_SELECTED_OSQUERY_TABLE = "SET_SELECTED_OSQUERY_TABLE",
}

interface ISetLastEditedQueryInfo {
  type: ACTIONS.SET_LAST_EDITED_QUERY_INFO;
  lastEditedQueryId?: number | null;
  lastEditedQueryName?: string;
  lastEditedQueryDescription?: string;
  lastEditedQueryBody?: string;
  lastEditedQueryResolution?: string;
  lastEditedQueryPlatform?: IPlatformString | null;
}

interface ISetPolicyTeamID {
  type: ACTIONS.SET_POLICY_TEAM_ID;
  id: number;
}

interface ISetSelectedOsqueryTable {
  type: ACTIONS.SET_SELECTED_OSQUERY_TABLE;
  tableName: string;
}

type IAction =
  | ISetLastEditedQueryInfo
  | ISetPolicyTeamID
  | ISetSelectedOsqueryTable;

type Props = {
  children: ReactNode;
};

type InitialStateType = {
  lastEditedQueryId: number | null;
  lastEditedQueryName: string;
  lastEditedQueryDescription: string;
  lastEditedQueryBody: string;
  lastEditedQueryResolution: string;
  lastEditedQueryPlatform: IPlatformString | null;
  setLastEditedQueryId: (value: number) => void;
  setLastEditedQueryName: (value: string) => void;
  setLastEditedQueryDescription: (value: string) => void;
  setLastEditedQueryBody: (value: string) => void;
  setLastEditedQueryResolution: (value: string) => void;
  setLastEditedQueryPlatform: (value: IPlatformString | null) => void;
  policyTeamId: number;
  setPolicyTeamId: (id: number) => void;
  selectedOsqueryTable: IOsqueryTable;
  setSelectedOsqueryTable: (tableName: string) => void;
};

const initialState = {
  lastEditedQueryId: null,
  lastEditedQueryName: "",
  lastEditedQueryDescription: "",
  lastEditedQueryBody: "",
  lastEditedQueryResolution: "",
  lastEditedQueryPlatform: null,
  setLastEditedQueryId: () => null,
  setLastEditedQueryName: () => null,
  setLastEditedQueryDescription: () => null,
  setLastEditedQueryBody: () => null,
  setLastEditedQueryResolution: () => null,
  setLastEditedQueryPlatform: () => null,
  policyTeamId: 0,
  setPolicyTeamId: () => null,
  selectedOsqueryTable:
    find(osqueryTables, { name: "users" }) || DEFAULT_OSQUERY_TABLE,
  setSelectedOsqueryTable: () => null,
};

const reducer = (state: InitialStateType, action: IAction) => {
  switch (action.type) {
    case ACTIONS.SET_POLICY_TEAM_ID:
      return {
        ...state,
        policyTeamId: action.id,
      };
    case ACTIONS.SET_SELECTED_OSQUERY_TABLE:
      return {
        ...state,
        selectedOsqueryTable: find(osqueryTables, { name: action.tableName }),
      };
    case ACTIONS.SET_LAST_EDITED_QUERY_INFO:
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
        lastEditedQueryResolution:
          typeof action.lastEditedQueryResolution === "undefined"
            ? state.lastEditedQueryResolution
            : action.lastEditedQueryResolution,
        lastEditedQueryPlatform:
          typeof action.lastEditedQueryPlatform === "undefined"
            ? state.lastEditedQueryPlatform
            : action.lastEditedQueryPlatform,
      };
    default:
      return state;
  }
};

export const PolicyContext = createContext<InitialStateType>(initialState);

const PolicyProvider = ({ children }: Props): JSX.Element => {
  const [state, dispatch] = useReducer(reducer, initialState);

  const value = {
    lastEditedQueryId: state.lastEditedQueryId,
    lastEditedQueryName: state.lastEditedQueryName,
    lastEditedQueryDescription: state.lastEditedQueryDescription,
    lastEditedQueryBody: state.lastEditedQueryBody,
    lastEditedQueryResolution: state.lastEditedQueryResolution,
    lastEditedQueryPlatform: state.lastEditedQueryPlatform,
    setLastEditedQueryId: (lastEditedQueryId: number) => {
      dispatch({
        type: ACTIONS.SET_LAST_EDITED_QUERY_INFO,
        lastEditedQueryId,
      });
    },
    setLastEditedQueryName: (lastEditedQueryName: string) => {
      dispatch({
        type: ACTIONS.SET_LAST_EDITED_QUERY_INFO,
        lastEditedQueryName,
      });
    },
    setLastEditedQueryDescription: (lastEditedQueryDescription: string) => {
      dispatch({
        type: ACTIONS.SET_LAST_EDITED_QUERY_INFO,
        lastEditedQueryDescription,
      });
    },
    setLastEditedQueryBody: (lastEditedQueryBody: string) => {
      dispatch({
        type: ACTIONS.SET_LAST_EDITED_QUERY_INFO,
        lastEditedQueryBody,
      });
    },
    setLastEditedQueryResolution: (lastEditedQueryResolution: string) => {
      dispatch({
        type: ACTIONS.SET_LAST_EDITED_QUERY_INFO,
        lastEditedQueryResolution,
      });
    },
    setLastEditedQueryPlatform: (
      lastEditedQueryPlatform: IPlatformString | null | undefined
    ) => {
      dispatch({
        type: ACTIONS.SET_LAST_EDITED_QUERY_INFO,
        lastEditedQueryPlatform,
      });
    },
    policyTeamId: state.policyTeamId,
    setPolicyTeamId: (id: number) => {
      dispatch({ type: ACTIONS.SET_POLICY_TEAM_ID, id });
    },
    selectedOsqueryTable: state.selectedOsqueryTable,
    setSelectedOsqueryTable: (tableName: string) => {
      dispatch({ type: ACTIONS.SET_SELECTED_OSQUERY_TABLE, tableName });
    },
  };

  return (
    <PolicyContext.Provider value={value}>{children}</PolicyContext.Provider>
  );
};

export default PolicyProvider;
