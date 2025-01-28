import React from "react";
import { InjectedRouter } from "react-router";
import { render, RenderOptions, RenderResult } from "@testing-library/react";
import type { UserEvent } from "@testing-library/user-event/dist/types/setup/setup";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "react-query";

import { AppContext, IAppContext, initialState } from "context/app";
import {
  INotificationContext,
  NotificationContext,
} from "context/notification";
import { IPolicyContext, PolicyContext } from "context/policy";
import { IQueryContext, QueryContext } from "context/query";

export const baseUrl = (path: string) => {
  return `/api/latest/fleet${path}`;
};

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

interface IContextOptions {
  app?: Partial<IAppContext>;
  notification?: Partial<INotificationContext>;
  policy?: Partial<IPolicyContext>;
  query?: Partial<IQueryContext>;
}

interface ICustomRenderOptions {
  context?: IContextOptions;
  withBackendMock?: boolean;
}

const CONTEXT_PROVIDER_MAP = {
  app: AppContext,
  notification: NotificationContext,
  policy: PolicyContext,
  query: QueryContext,
};

type ContextProviderKeys = keyof typeof CONTEXT_PROVIDER_MAP;
interface IWrapperComponentProps {
  client?: QueryClient;
  value?: Partial<IAppContext> | Partial<INotificationContext>;
}

const createWrapperComponent = (
  CurrentWrapper: React.FC<React.PropsWithChildren<any>>, // TODO: types
  WrapperComponent: React.FC<React.PropsWithChildren<any>>, // TODO: types
  props: IWrapperComponentProps
) => {
  return ({ children }: IChildrenProp) => (
    <WrapperComponent {...props}>
      <CurrentWrapper>{children}</CurrentWrapper>
    </WrapperComponent>
  );
};

interface IChildrenProp {
  children?: React.ReactNode;
}

type RenderResultWithUser = RenderResult & { user: UserEvent };

const addQueryProviderWrapper = (
  CustomWrapperComponent: ({ children }: IChildrenProp) => JSX.Element
) => {
  const client = new QueryClient();
  CustomWrapperComponent = createWrapperComponent(
    CustomWrapperComponent,
    QueryClientProvider,
    { client }
  );

  return CustomWrapperComponent;
};

const addContextWrappers = (
  contextObj: IContextOptions,
  CustomWrapperComponent: ({ children }: IChildrenProp) => JSX.Element
) => {
  Object.entries(contextObj).forEach(([key, value]) => {
    CustomWrapperComponent = createWrapperComponent(
      CustomWrapperComponent,
      CONTEXT_PROVIDER_MAP[key as ContextProviderKeys].Provider,
      { value }
    );
  });
  return CustomWrapperComponent;
};

/**
 * Creates a custom testing-library render function based on a configuration object.
 * It will help set up the react context and backend mock dependencies so that
 * you can easily set up a component for testing.
 *
 * This will also set up the @testing-library/user-events and expose a user object
 * you can use to perform user interactions.
 */
export const createCustomRenderer = (renderOptions?: ICustomRenderOptions) => {
  let CustomWrapperComponent = ({ children }: IChildrenProp) => <>{children}</>;

  if (renderOptions?.withBackendMock) {
    CustomWrapperComponent = addQueryProviderWrapper(CustomWrapperComponent);
  }

  if (renderOptions?.context !== undefined) {
    CustomWrapperComponent = addContextWrappers(
      renderOptions.context,
      CustomWrapperComponent
    );
  }

  return (
    component: React.ReactElement,
    options?: Omit<RenderOptions, "wrapper">
  ): RenderResultWithUser => {
    const renderResults: RenderResultWithUser = {
      user: userEvent.setup(),
      ...render(component, { wrapper: CustomWrapperComponent, ...options }),
    };

    return renderResults;
  };
};

/**
 * This is a convenince method that calls the render method from `@testing-library/react` and also
 * sets up the also `user-events`library and adds the user object to the returned object.
 */
// eslint-disable-next-line import/prefer-default-export
export const renderWithSetup = (component: JSX.Element) => {
  return {
    user: userEvent.setup(),
    ...render(component),
  };
};

const DEFAULT_MOCK_ROUTER: InjectedRouter = {
  push: jest.fn(),
  replace: jest.fn(),
  goBack: jest.fn(),
  goForward: jest.fn(),
  go: jest.fn(),
  setRouteLeaveHook: jest.fn(),
  isActive: jest.fn(),
  createHref: jest.fn(),
  createPath: jest.fn(),
};

export const createMockRouter = (overrides?: Partial<InjectedRouter>) => {
  return {
    ...DEFAULT_MOCK_ROUTER,
    ...overrides,
  };
};

/** helper method to generate a date "x" days ago. */
export const getPastDate = (days: number) => {
  const targetDate = new Date();
  targetDate.setDate(targetDate.getDate() - days);
  return targetDate.toISOString();
};

/** helper method to generate a date "x" days from now */
export const getFutureDate = (days: number) => {
  const targetDate = new Date();
  targetDate.setDate(targetDate.getDate() + days);
  return targetDate.toISOString();
};
