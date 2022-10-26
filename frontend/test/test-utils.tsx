import React from "react";
import { render, RenderOptions } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { AppContext, IAppContext, initialState } from "context/app";

type RenderOptionsWithProviderProps = RenderOptions & {
  contextValue: Partial<IAppContext>;
};

/**
 * A custom render method that provides a configurable App context when testing components
 */
// eslint-disable-next-line import/prefer-default-export
export const renderWithAppContext = (
  component: React.ReactNode,
  { contextValue, ...renderOptions }: RenderOptionsWithProviderProps
) => {
  const value: IAppContext = { ...initialState, ...contextValue };
  return render(
    <AppContext.Provider value={value}>{component}</AppContext.Provider>,
    renderOptions
  );
};

/**
  A collection of utilities to enable easier writting of tests
 */

// eslint-disable-next-line import/prefer-default-export
export const renderWithSetup = (component: JSX.Element) => {
  return {
    user: userEvent.setup(),
    ...render(component),
  };
};
