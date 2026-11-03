import React, { useEffect, useState } from "react";
import { Meta, StoryObj } from "@storybook/react";
import { QueryClient, QueryClientProvider } from "react-query";

import { INotificationView } from "interfaces/device_notification";

import DeviceNotificationCard from "./DeviceNotificationCard";

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

const APPS = [
  "1Password",
  "Docker Desktop",
  "Google Chrome",
  "Slack",
  "Zoom",
  "Firefox",
  "Visual Studio Code",
  "Notion",
  "Spotify",
  "Figma",
];

const buildItems = (count: number) =>
  APPS.slice(0, count).map((name, i) => ({
    software_title_id: i + 1,
    icon_url: null,
    name,
    display_name: name,
  }));

// Wide (400x100) placeholder logo so stories exercise the original bug's
// shape: before the fix, this wide SVG was pushed off the card's left edge
// when the error banner shared the actions row.
const WIDE_LOGO_URL = `data:image/svg+xml;utf8,${encodeURIComponent(
  `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 400 100">
    <rect width="400" height="100" fill="#192147" rx="12"/>
    <text x="200" y="60" text-anchor="middle" fill="#fff"
          font-family="Inter, sans-serif" font-size="36" font-weight="700">
      LOGOIPSUM
    </text>
  </svg>`
)}`;

const BASE_VIEW: INotificationView = {
  uuid: "storybook-notification",
  org_logo_url_light_mode: "",
  org_logo_url_dark_mode: "",
  title: "Save your work",
  description: "These apps will close and update in **1 hour**.",
  items: buildItems(4),
  actions: [
    { id: "remind", label: "Remind me 5 minutes before" },
    { id: "update_now", label: "Update now" },
  ],
  install_at: null,
};

const meta: Meta<typeof DeviceNotificationCard> = {
  title: "Pages/DeviceNotificationCard",
  component: DeviceNotificationCard,
  args: {
    view: BASE_VIEW,
    orgLogoURL: "",
    onPostAction: () => undefined,
  },
  // SoftwareIcon reads react-query internally, so every story needs a client.
  decorators: [
    (Story) => (
      <QueryClientProvider client={queryClient}>
        <Story />
      </QueryClientProvider>
    ),
  ],
};

export default meta;

type Story = StoryObj<typeof DeviceNotificationCard>;

export const Default: Story = {};

// Error-state stories expose an `itemCount` knob (1–6) so the number of
// apps in the list can be swept from the Storybook controls panel.
type ErrorStoryArgs = React.ComponentProps<typeof DeviceNotificationCard> & {
  itemCount: number;
};
type ErrorStory = StoryObj<ErrorStoryArgs>;

const itemCountArgType = {
  itemCount: {
    control: { type: "range" as const, min: 1, max: 6, step: 1 },
    description:
      "Number of apps shown in the list (storybook-only knob — in prod the server controls this).",
  },
};

const renderWithItemCount = ({ itemCount, view, ...rest }: ErrorStoryArgs) => (
  <DeviceNotificationCard
    {...rest}
    view={{ ...view, items: buildItems(itemCount) }}
  />
);

export const WithError: ErrorStory = {
  argTypes: itemCountArgType,
  args: { itemCount: 4, isPostError: true },
  render: renderWithItemCount,
};

// 10-app list that exceeds the 4.5-row cap and scrolls. With the error on,
// the list cap shrinks to 3.5 rows so the scroll area is shorter and the
// window doesn't grow as much.
export const LongListWithError: Story = {
  args: {
    view: { ...BASE_VIEW, items: buildItems(10) },
    isPostError: true,
  },
};

export const WideLogoWithError: ErrorStory = {
  argTypes: itemCountArgType,
  args: {
    itemCount: 4,
    orgLogoURL: WIDE_LOGO_URL,
    isPostError: true,
  },
  render: renderWithItemCount,
};

// Mirrors the real timing: error flips on 5 seconds after mount. Clicking
// Update now clears the error and shows the button's loading spinner for
// 5 seconds, then the error reappears — matching the "retry failed again"
// flow.
const ErrorAfterDelay = ({
  itemCount,
  view,
  ...rest
}: ErrorStoryArgs) => {
  const [isPostError, setIsPostError] = useState(false);
  const [isPosting, setIsPosting] = useState(false);

  // While not in the error state (either initial mount or an in-flight
  // retry), wait 5 seconds and then surface the error.
  useEffect(() => {
    if (isPostError) return undefined;
    const timer = window.setTimeout(() => {
      setIsPosting(false);
      setIsPostError(true);
    }, 5_000);
    return () => window.clearTimeout(timer);
  }, [isPostError]);

  const onPostAction = (_action: unknown, isPrimary: boolean) => {
    if (!isPrimary) return;
    setIsPostError(false);
    setIsPosting(true);
  };

  return (
    <DeviceNotificationCard
      {...rest}
      view={{ ...view, items: buildItems(itemCount) }}
      isPostError={isPostError}
      isPosting={isPosting}
      onPostAction={onPostAction}
    />
  );
};

export const ErrorAfter5Seconds: ErrorStory = {
  argTypes: itemCountArgType,
  args: { itemCount: 5 },
  render: (args) => <ErrorAfterDelay {...args} />,
};

export const WideLogoErrorAfter5Seconds: ErrorStory = {
  argTypes: itemCountArgType,
  args: { itemCount: 5, orgLogoURL: WIDE_LOGO_URL },
  render: (args) => <ErrorAfterDelay {...args} />,
};
