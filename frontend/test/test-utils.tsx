import React from "react";
import { InjectedRouter } from "react-router";
import { Location } from "history";
import {
  render,
  RenderOptions,
  RenderResult,
  waitFor,
} from "@testing-library/react";
import type { UserEvent } from "@testing-library/user-event";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "react-query";

import { AppContext, IAppContext, initialState } from "context/app";
import { IPolicyContext, PolicyContext } from "context/policy";
import { IQueryContext, QueryContext } from "context/query";
import { IRouterLocation } from "interfaces/routing";

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

// recursively make all fields in T optional
type DeepPartial<T> = T extends object
  ? {
      [P in keyof T]?: DeepPartial<T[P]>;
    }
  : T;

interface IContextOptions {
  // DeepPartial allows inclusion of only fields needed for testing, even if such a partial type
  // is not acceptable in actual application code
  app?: DeepPartial<IAppContext>;
  policy?: Partial<IPolicyContext>;
  query?: Partial<IQueryContext>;
}

interface ICustomRenderOptions {
  context?: IContextOptions;
  withBackendMock?: boolean;
}

const CONTEXT_PROVIDER_MAP = {
  app: AppContext,
  policy: PolicyContext,
  query: QueryContext,
};

type ContextProviderKeys = keyof typeof CONTEXT_PROVIDER_MAP;
interface IWrapperComponentProps {
  client?: QueryClient;
  value?: Partial<IAppContext>;
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
  const client = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false, // important: no automatic retries in tests
        cacheTime: 0, // optional but makes behavior deterministic
      },
    },
  });

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

export const createMockLocation = (overrides?: Partial<Location>): Location => {
  return {
    pathname: "/",
    search: "",
    hash: "",
    query: {},
    state: undefined,
    action: "POP",
    key: "",
    ...overrides,
  };
};

export const createMockLocationExperimental = (
  overrides?: Partial<IRouterLocation>
): IRouterLocation => {
  // Default values for the location object
  const defaultLocation: IRouterLocation = {
    pathname: "/",
    host: "localhost:8080",
    hostname: "localhost",
    port: "8080",
    protocol: "http:",
  };

  return {
    ...defaultLocation,
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

// Fleet's Modal renders no dialog role, so testing-library's byRole query cannot reach it. Scope by the shared modal
// container class instead. Tests open one modal at a time, so a single match is the whole contract.
const MODAL_CONTAINER_SELECTOR = ".modal__modal_container";

/**
 * Waits for a modal to open and returns its container, for scoping queries with `within`.
 */
export const getOpenModal = (): Promise<HTMLElement> =>
  waitFor(() => {
    const modal = document.querySelector<HTMLElement>(MODAL_CONTAINER_SELECTOR);
    if (!modal) {
      throw new Error("Modal not yet rendered");
    }
    return modal;
  });

/**
 * Returns the open modal's container, or null when none is open. Use to assert a modal has closed — going through this
 * rather than querying the class directly keeps such assertions from passing vacuously if the container class changes.
 */
export const queryOpenModal = (): HTMLElement | null =>
  document.querySelector(MODAL_CONTAINER_SELECTOR);

export const waitForLoadingToFinish = async (container: HTMLElement) => {
  await waitFor(() => {
    expect(container.querySelector(".loading-overlay")).not.toBeInTheDocument();
  });
};
