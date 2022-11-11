import React from "react";
import { render, RenderOptions, RenderResult } from "@testing-library/react";
import type { UserEvent } from "@testing-library/user-event/dist/types/setup/setup";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "react-query";

import { AppContext, IAppContext, initialState } from "context/app";
import {
  INotificationContext,
  NotificationContext,
} from "context/notification";

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
}

interface ICustomRenderOptions {
  context?: IContextOptions;
  withBackendMock?: boolean;
}

// TODO: types
// type RenderOptionsWithoutUserEvents = ICustomRenderOptions & {
//   withUserEvents: false;
// };

// type RenderOptionsWithUserEvents = ICustomRenderOptions & {
//   withUserEvents: true;
// };

const CONTEXT_PROVIDER_MAP = {
  app: AppContext,
  notification: NotificationContext,
};

type ContextProviderKeys = keyof typeof CONTEXT_PROVIDER_MAP;
interface IWrapperComponentProps {
  client?: QueryClient;
  value?: Partial<IAppContext> | Partial<INotificationContext>;
}

const createWrapperComponent = (
  CurrentWrapper: React.FC<any>, // TODO: types
  WrapperComponent: React.FC<any>, // TODO: types
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
export const createCustomRenderer = (renderOptions: ICustomRenderOptions) => {
  let CustomWrapperComponent = ({ children }: IChildrenProp) => <>{children}</>;

  if (renderOptions.withBackendMock) {
    CustomWrapperComponent = addQueryProviderWrapper(CustomWrapperComponent);
  }

  if (renderOptions.context !== undefined) {
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
