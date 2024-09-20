import React, { createContext, useReducer, useMemo, ReactNode } from "react";

type Props = {
  children: ReactNode;
};

type InitialStateType = {
  resetSelectedRows: boolean;
  setResetSelectedRows: (update: boolean) => void;
};

const initialState = {
  resetSelectedRows: false,
  setResetSelectedRows: () => null,
};

const actions = {
  RESET_SELECTED_ROWS: "RESET_SELECTED_ROWS",
};

const reducer = (state: any, action: any) => {
  switch (action.type) {
    case actions.RESET_SELECTED_ROWS:
      return { ...state, resetSelectedRows: action.resetSelectedRows };
    default:
      return state;
  }
};

export const TableContext = createContext<InitialStateType>(initialState);

const TableProvider = ({ children }: Props) => {
  const [state, dispatch] = useReducer(reducer, initialState);

  const value = useMemo(
    () => ({
      resetSelectedRows: state.resetSelectedRows,
      setResetSelectedRows: (resetSelectedRows: boolean) => {
        dispatch({ type: actions.RESET_SELECTED_ROWS, resetSelectedRows });
      },
    }),
    [state.resetSelectedRows]
  );

  return (
    <TableContext.Provider value={value}>{children}</TableContext.Provider>
  );
};

export default TableProvider;
