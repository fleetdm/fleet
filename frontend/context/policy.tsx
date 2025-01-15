import React, {
  createContext,
  useReducer,
  ReactNode,
  useMemo,
  useCallback,
} from "react";
import { find } from "lodash";

import { osqueryTables } from "utilities/osquery_tables";
import { IOsQueryTable, DEFAULT_OSQUERY_TABLE } from "interfaces/osquery_table";
import { CommaSeparatedPlatformString } from "interfaces/platform";

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
  lastEditedQueryCritical?: boolean;
  lastEditedQueryPlatform?: CommaSeparatedPlatformString | null;
  defaultPolicy?: boolean;
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
  lastEditedQueryCritical: boolean;
  lastEditedQueryPlatform: CommaSeparatedPlatformString | null;
  defaultPolicy: boolean;
  setLastEditedQueryId: (value: number | null) => void;
  setLastEditedQueryName: (value: string) => void;
  setLastEditedQueryDescription: (value: string) => void;
  setLastEditedQueryBody: (value: string) => void;
  setLastEditedQueryResolution: (value: string) => void;
  setLastEditedQueryCritical: (value: boolean) => void;
  setLastEditedQueryPlatform: (
    value: CommaSeparatedPlatformString | null
  ) => void;
  setDefaultPolicy: (value: boolean) => void;
  policyTeamId: number;
  setPolicyTeamId: (id: number) => void;
  selectedOsqueryTable: IOsQueryTable;
  setSelectedOsqueryTable: (tableName: string) => void;
};

const initTable =
  osqueryTables.find((table) => table.name === "users") ||
  DEFAULT_OSQUERY_TABLE;

export type IPolicyContext = InitialStateType;

const initialState = {
  lastEditedQueryId: null,
  lastEditedQueryName: "",
  lastEditedQueryDescription: "",
  lastEditedQueryBody: "",
  lastEditedQueryResolution: "",
  lastEditedQueryCritical: false,
  lastEditedQueryPlatform: null,
  defaultPolicy: false,
  setLastEditedQueryId: () => null,
  setLastEditedQueryName: () => null,
  setLastEditedQueryDescription: () => null,
  setLastEditedQueryBody: () => null,
  setLastEditedQueryResolution: () => null,
  setLastEditedQueryCritical: () => null,
  setLastEditedQueryPlatform: () => null,
  setDefaultPolicy: () => null,
  policyTeamId: 0,
  setPolicyTeamId: () => null,
  selectedOsqueryTable: initTable,
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
        selectedOsqueryTable:
          find(osqueryTables, { name: action.tableName }) ||
          DEFAULT_OSQUERY_TABLE,
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
        lastEditedQueryCritical:
          typeof action.lastEditedQueryCritical === "undefined"
            ? state.lastEditedQueryCritical
            : action.lastEditedQueryCritical,
        lastEditedQueryPlatform:
          typeof action.lastEditedQueryPlatform === "undefined"
            ? state.lastEditedQueryPlatform
            : action.lastEditedQueryPlatform,
        defaultPolicy:
          typeof action.defaultPolicy === "undefined"
            ? state.defaultPolicy
            : action.defaultPolicy,
      };
    default:
      return state;
  }
};

// TODO: Can we remove policyTeamId in favor of always using URL team_id param?
export const PolicyContext = createContext<InitialStateType>(initialState);

const PolicyProvider = ({ children }: Props): JSX.Element => {
  const [state, dispatch] = useReducer(reducer, initialState);

  const setPolicyTeamId = useCallback((id: number) => {
    dispatch({ type: ACTIONS.SET_POLICY_TEAM_ID, id });
  }, []);

  const setLastEditedQueryId = useCallback(
    (lastEditedQueryId: number | null) => {
      dispatch({
        type: ACTIONS.SET_LAST_EDITED_QUERY_INFO,
        lastEditedQueryId,
      });
    },
    []
  );

  const setLastEditedQueryName = useCallback((lastEditedQueryName: string) => {
    dispatch({
      type: ACTIONS.SET_LAST_EDITED_QUERY_INFO,
      lastEditedQueryName,
    });
  }, []);

  const setLastEditedQueryDescription = useCallback(
    (lastEditedQueryDescription: string) => {
      dispatch({
        type: ACTIONS.SET_LAST_EDITED_QUERY_INFO,
        lastEditedQueryDescription,
      });
    },
    []
  );
  const setLastEditedQueryBody = useCallback((lastEditedQueryBody: string) => {
    dispatch({
      type: ACTIONS.SET_LAST_EDITED_QUERY_INFO,
      lastEditedQueryBody,
    });
  }, []);
  const setLastEditedQueryResolution = useCallback(
    (lastEditedQueryResolution: string) => {
      dispatch({
        type: ACTIONS.SET_LAST_EDITED_QUERY_INFO,
        lastEditedQueryResolution,
      });
    },
    []
  );
  const setLastEditedQueryCritical = useCallback(
    (lastEditedQueryCritical: boolean) => {
      dispatch({
        type: ACTIONS.SET_LAST_EDITED_QUERY_INFO,
        lastEditedQueryCritical,
      });
    },
    []
  );
  const setLastEditedQueryPlatform = useCallback(
    (
      lastEditedQueryPlatform: CommaSeparatedPlatformString | null | undefined
    ) => {
      dispatch({
        type: ACTIONS.SET_LAST_EDITED_QUERY_INFO,
        lastEditedQueryPlatform,
      });
    },
    []
  );
  const setDefaultPolicy = useCallback((defaultPolicy: boolean) => {
    dispatch({
      type: ACTIONS.SET_LAST_EDITED_QUERY_INFO,
      defaultPolicy,
    });
  }, []);

  const setSelectedOsqueryTable = useCallback((tableName: string) => {
    dispatch({ type: ACTIONS.SET_SELECTED_OSQUERY_TABLE, tableName });
  }, []);

  const value = useMemo(
    () => ({
      lastEditedQueryId: state.lastEditedQueryId,
      lastEditedQueryName: state.lastEditedQueryName,
      lastEditedQueryDescription: state.lastEditedQueryDescription,
      lastEditedQueryBody: state.lastEditedQueryBody,
      lastEditedQueryResolution: state.lastEditedQueryResolution,
      lastEditedQueryCritical: state.lastEditedQueryCritical,
      lastEditedQueryPlatform: state.lastEditedQueryPlatform,
      defaultPolicy: state.defaultPolicy,
      setLastEditedQueryId,
      setLastEditedQueryName,
      setLastEditedQueryDescription,
      setLastEditedQueryBody,
      setLastEditedQueryResolution,
      setLastEditedQueryCritical,
      setLastEditedQueryPlatform,
      setDefaultPolicy,
      policyTeamId: state.policyTeamId,
      setPolicyTeamId,
      selectedOsqueryTable: state.selectedOsqueryTable,
      setSelectedOsqueryTable,
    }),
    [
      setDefaultPolicy,
      setLastEditedQueryBody,
      setLastEditedQueryCritical,
      setLastEditedQueryDescription,
      setLastEditedQueryId,
      setLastEditedQueryName,
      setLastEditedQueryPlatform,
      setLastEditedQueryResolution,
      setPolicyTeamId,
      setSelectedOsqueryTable,
      state.defaultPolicy,
      state.lastEditedQueryBody,
      state.lastEditedQueryCritical,
      state.lastEditedQueryDescription,
      state.lastEditedQueryId,
      state.lastEditedQueryName,
      state.lastEditedQueryPlatform,
      state.lastEditedQueryResolution,
      state.policyTeamId,
      state.selectedOsqueryTable,
    ]
  );

  return (
    <PolicyContext.Provider value={value}>{children}</PolicyContext.Provider>
  );
};

export default PolicyProvider;
