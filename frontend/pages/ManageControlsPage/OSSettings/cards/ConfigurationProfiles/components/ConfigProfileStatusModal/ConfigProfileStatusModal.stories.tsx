/* eslint-disable @typescript-eslint/no-empty-function */
import React from "react";
import { Meta, StoryObj } from "@storybook/react";
import { AppContext } from "context/app";
import createMockConfig from "__mocks__/configMock";
import { IMdmConfig } from "interfaces/config";
import {
  QueryClient,
  QueryClientProvider,
  QueryClientProviderProps,
} from "react-query";
import configProfileAPI from "services/entities/config_profiles";
import ConfigProfileStatusModal from "./ConfigProfileStatusModal";

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});
type CustomQueryClientProviderProps = React.PropsWithChildren<QueryClientProviderProps>;
const CustomQueryClientProvider: React.FC<CustomQueryClientProviderProps> = QueryClientProvider;

// Storybook has no backend — short-circuit the network call react-query fires on mount
configProfileAPI.getConfigProfileStatus = () =>
  Promise.resolve({
    verified: 0,
    verifying: 2,
    pending: 1,
    failed: 3,
  });

const meta: Meta<typeof ConfigProfileStatusModal> = {
  component: ConfigProfileStatusModal,
  title: "Pages/ConfigurationProfiles/Components/ConfigProfileStatusModal",
  args: {
    name: "Storybook Profile",
    teamId: 0,
    uuid: "apple-profile",
    onClickResend: () => {},
    onExit: () => {},
    platform: "darwin",
  },
  argTypes: {
    onClickResend: { control: false },
    onExit: { control: false },
    teamId: { control: false },
    uuid: { control: false },
  },
  decorators: [
    (Story) => {
      const appContextValue = {
        isPremiumTier: true,
        config: createMockConfig({
          mdm: {
            enabled_and_configured: true,
          } as IMdmConfig,
        }),
        setConfig: () => {
          // Mock function for stories
        },
      };
      return (
        <CustomQueryClientProvider client={queryClient}>
          <AppContext.Provider value={appContextValue as any}>
            <Story />
          </AppContext.Provider>
        </CustomQueryClientProvider>
      );
    },
  ],
};

export default meta;
type Story = StoryObj<typeof ConfigProfileStatusModal>;

export const Default: Story = {};

export const MDMTurnedOff: Story = {
  args: {
    platform: "windows",
  },
};
