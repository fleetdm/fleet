import React, { createContext, useReducer, ReactNode } from "react";
import { find } from "lodash";

// @ts-ignore
import { osqueryTables } from "utilities/osquery_tables";
import { IOsqueryTable } from "interfaces/osquery_table";

type Props = {
  children: ReactNode;
};

type InitialStateType = {
  selectedOsqueryTable: IOsqueryTable;
  setSelectedOsqueryTable: (tableName: string) => void;
};

const initialState = {
  selectedOsqueryTable: find(osqueryTables, { name: "users" }),
  setSelectedOsqueryTable: () => null,
};

const actions = {
  SET_SELECTED_OSQUERY_TABLE: "SET_SELECTED_OSQUERY_TABLE",
};

const reducer = (state: any, action: any) => {
  switch (action.type) {
    case actions.SET_SELECTED_OSQUERY_TABLE:
      return { 
        ...state, 
        selectedOsqueryTable: find(osqueryTables, { name: action.tableName }),
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
    setSelectedOsqueryTable: (tableName: string) => {
      dispatch({ type: actions.SET_SELECTED_OSQUERY_TABLE, tableName });
    },
  };

  return (
    <QueryContext.Provider value={value}>{children}</QueryContext.Provider>
  );
};

export default QueryProvider;
