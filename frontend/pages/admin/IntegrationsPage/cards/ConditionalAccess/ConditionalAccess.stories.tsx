/* eslint-disable @typescript-eslint/no-explicit-any */
import React from "react";
import { Meta, StoryObj } from "@storybook/react";
import {
  QueryClient,
  QueryClientProvider,
  QueryClientProviderProps,
} from "react-query";

import createMockConfig from "__mocks__/configMock";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import Button from "components/buttons/Button";

import ConditionalAccess from "./ConditionalAccess";
import SectionCard from "../MdmSettings/components/SectionCard";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
    },
  },
});

// Workaround for React Query v3 with React 18 - explicitly add children prop
// See frontend/router/index.tsx for the same pattern
type CustomQueryClientProviderProps = React.PropsWithChildren<QueryClientProviderProps>;
const CustomQueryClientProvider: React.FC<CustomQueryClientProviderProps> = QueryClientProvider;

const mockNotificationContext = {
  renderFlash: () => {
    // Mock function for stories
  },
  hideFlash: () => {
    // Mock function for stories
  },
};

const meta: Meta<typeof ConditionalAccess> = {
  title: "Components/ConditionalAccess",
  component: ConditionalAccess,
};

export default meta;

type Story = StoryObj<typeof ConditionalAccess>;

export const NotConfigured: Story = {
  name: "Not configured (premium tier)",
  decorators: [
    (Story) => {
      const appContextValue = {
        isPremiumTier: true,
        config: createMockConfig({
          conditional_access: {
            microsoft_entra_tenant_id: "",
            microsoft_entra_connection_configured: false,
            okta_idp_id: "",
            okta_assertion_consumer_service_url: "",
            okta_audience_uri: "",
            okta_certificate: "",
          },
        }),
        setConfig: () => {
          // Mock function for stories
        },
      };

      return (
        <CustomQueryClientProvider client={queryClient}>
          <AppContext.Provider value={appContextValue as any}>
            <NotificationContext.Provider
              value={mockNotificationContext as any}
            >
              <Story />
            </NotificationContext.Provider>
          </AppContext.Provider>
        </CustomQueryClientProvider>
      );
    },
  ],
};

// This story demonstrates the intermediate "pending" state on the Entra card
// after form submission while awaiting OAuth completion in another tab
export const AwaitingOAuthCompletion: Story = {
  name: "Awaiting OAuth completion (pending state)",
  render: () => {
    const appContextValue = {
      isPremiumTier: true,
      config: createMockConfig({
        conditional_access: {
          microsoft_entra_tenant_id: "",
          microsoft_entra_connection_configured: false,
          okta_idp_id: "",
          okta_assertion_consumer_service_url: "",
          okta_audience_uri: "",
          okta_certificate: "",
        },
      }),
      setConfig: () => {
        // Mock function for stories
      },
    };

    // Simulate the pending state by rendering SectionCard with pending icon
    return (
      <CustomQueryClientProvider client={queryClient}>
        <AppContext.Provider value={appContextValue as any}>
          <NotificationContext.Provider value={mockNotificationContext as any}>
            <div className="conditional-access">
              <div className="section-header">
                <h2 className="section-header__title">Conditional access</h2>
              </div>
              <p className="conditional-access__page-description">
                Block hosts failing policies from logging in with single
                sign-on. Once connected, enable or disable on the Policies page.
              </p>
              <div className="conditional-access__cards">
                <SectionCard
                  header="Okta"
                  cta={
                    <Button
                      onClick={() => {
                        // Mock function for stories
                      }}
                    >
                      Connect
                    </Button>
                  }
                >
                  Connect Okta to enable conditional access.
                </SectionCard>
                <SectionCard iconName="pending-outline" cta={undefined}>
                  To complete your integration, follow the instructions in the
                  other tab, then refresh this page to verify.
                </SectionCard>
              </div>
            </div>
          </NotificationContext.Provider>
        </AppContext.Provider>
      </CustomQueryClientProvider>
    );
  },
};

export const EntraConfigured: Story = {
  name: "Microsoft Entra configured",
  decorators: [
    (Story) => {
      const appContextValue = {
        isPremiumTier: true,
        config: createMockConfig({
          conditional_access: {
            microsoft_entra_tenant_id: "abcd-1234-efgh-5678",
            microsoft_entra_connection_configured: true,
            okta_idp_id: "",
            okta_assertion_consumer_service_url: "",
            okta_audience_uri: "",
            okta_certificate: "",
          },
        }),
        setConfig: () => {
          // Mock function for stories
        },
      };

      return (
        <CustomQueryClientProvider client={queryClient}>
          <AppContext.Provider value={appContextValue as any}>
            <NotificationContext.Provider
              value={mockNotificationContext as any}
            >
              <Story />
            </NotificationContext.Provider>
          </AppContext.Provider>
        </CustomQueryClientProvider>
      );
    },
  ],
};

export const OktaConfigured: Story = {
  name: "Okta configured",
  decorators: [
    (Story) => {
      const appContextValue = {
        isPremiumTier: true,
        config: createMockConfig({
          conditional_access: {
            microsoft_entra_tenant_id: "",
            microsoft_entra_connection_configured: false,
            okta_idp_id: "okta-idp-123456",
            okta_assertion_consumer_service_url:
              "https://fleet.example.com/api/v1/saml/acs",
            okta_audience_uri: "https://fleet.example.com",
            okta_certificate:
              "-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----",
          },
        }),
        setConfig: () => {
          // Mock function for stories
        },
      };

      return (
        <CustomQueryClientProvider client={queryClient}>
          <AppContext.Provider value={appContextValue as any}>
            <NotificationContext.Provider
              value={mockNotificationContext as any}
            >
              <Story />
            </NotificationContext.Provider>
          </AppContext.Provider>
        </CustomQueryClientProvider>
      );
    },
  ],
};

export const BothConfigured: Story = {
  name: "Both providers configured",
  decorators: [
    (Story) => {
      const appContextValue = {
        isPremiumTier: true,
        config: createMockConfig({
          conditional_access: {
            microsoft_entra_tenant_id: "abcd-1234-efgh-5678",
            microsoft_entra_connection_configured: true,
            okta_idp_id: "okta-idp-123456",
            okta_assertion_consumer_service_url:
              "https://fleet.example.com/api/v1/saml/acs",
            okta_audience_uri: "https://fleet.example.com",
            okta_certificate:
              "-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----",
          },
        }),
        setConfig: () => {
          // Mock function for stories
        },
      };

      return (
        <CustomQueryClientProvider client={queryClient}>
          <AppContext.Provider value={appContextValue as any}>
            <NotificationContext.Provider
              value={mockNotificationContext as any}
            >
              <Story />
            </NotificationContext.Provider>
          </AppContext.Provider>
        </CustomQueryClientProvider>
      );
    },
  ],
};

export const FreeTier: Story = {
  name: "Free tier (premium feature)",
  decorators: [
    (Story) => {
      const appContextValue = {
        isPremiumTier: false,
        config: createMockConfig({}),
        setConfig: () => {
          // Mock function for stories
        },
      };

      return (
        <CustomQueryClientProvider client={queryClient}>
          <AppContext.Provider value={appContextValue as any}>
            <NotificationContext.Provider
              value={mockNotificationContext as any}
            >
              <Story />
            </NotificationContext.Provider>
          </AppContext.Provider>
        </CustomQueryClientProvider>
      );
    },
  ],
};
