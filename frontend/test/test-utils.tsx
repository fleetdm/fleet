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

interface ICustomRenderOptions {
  context?: {
    app?: Partial<IAppContext>;
    notification?: Partial<INotificationContext>;
  };
  withBackendMock?: boolean;
  withUserEvents?: boolean;
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

const createWrapperComponent = (
  CurrentWrapper: React.FC<any>, // TODO: types
  WrapperComponent: React.FC<any>, // TODO: types
  props: any // TODO: types
) => {
  return ({ children }: IChildrenProp) => (
    <WrapperComponent {...props}>
      <CurrentWrapper>{children}</CurrentWrapper>
    </WrapperComponent>
  );
};

interface IChildrenProp {
  children: React.ReactNode;
}
type RenderResultWithUser = RenderResult & { user?: UserEvent };

// TODO: types
export const createCustomRenderer = (renderOptions: ICustomRenderOptions) => {
  let CustomWrapperComponent = ({ children }: IChildrenProp) => <>{children}</>;

  if (renderOptions.withBackendMock) {
    const client = new QueryClient();
    CustomWrapperComponent = createWrapperComponent(
      CustomWrapperComponent,
      QueryClientProvider,
      { client }
    );
  }

  if (renderOptions.context !== undefined) {
    Object.keys(renderOptions.context).forEach((key) => {
      CustomWrapperComponent = createWrapperComponent(
        CustomWrapperComponent,
        CONTEXT_PROVIDER_MAP[key as ContextProviderKeys].Provider,
        { value: renderOptions.context?.[key as ContextProviderKeys] }
      );
    });
  }

  return (component: JSX.Element, options?: Omit<RenderOptions, "wrapper">) => {
    const renderResults: RenderResultWithUser = {
      ...render(component, { wrapper: CustomWrapperComponent, ...options }),
    };

    if (renderOptions.withUserEvents) {
      renderResults.user = userEvent.setup();
      return renderResults as RenderResultWithUser;
    }

    return renderResults as RenderResult;
  };
};
