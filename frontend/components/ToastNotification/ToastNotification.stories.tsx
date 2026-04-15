import React from "react";
import { Meta, StoryObj } from "@storybook/react";

import ToastNotification, { notify } from ".";

import "../../index.scss";

const meta: Meta<typeof ToastNotification> = {
  component: ToastNotification,
  title: "Components/ToastNotification",
  // Opt out of the global `autodocs` tag. Sonner's `toast.xxx()` dispatches to
  // every `<Toaster />` mounted on the page, so rendering all stories together
  // on a docs page would fire the same toast in every story box.
  tags: ["!autodocs"],
  parameters: {
    layout: "fullscreen",
  },
  decorators: [
    (Story) => (
      <div style={{ padding: "24px", minHeight: "320px" }}>
        <Story />
      </div>
    ),
  ],
};

export default meta;

type Story = StoryObj<typeof ToastNotification>;

const TriggerButton = ({
  label,
  onClick,
}: {
  label: string;
  onClick: () => void;
}): JSX.Element => (
  <button
    type="button"
    onClick={onClick}
    style={{
      padding: "8px 16px",
      borderRadius: "6px",
      border: "1px solid #e2e4ea",
      background: "#ffffff",
      cursor: "pointer",
      fontFamily: "inherit",
      fontSize: "14px",
    }}
  >
    {label}
  </button>
);

/**
 * Default — renders the Toaster alone, no toasts visible.
 */
export const Default: Story = {
  render: () => <ToastNotification />,
};

/**
 * Success — click the button to fire a success toast.
 */
export const Success: Story = {
  render: () => (
    <>
      <ToastNotification />
      <TriggerButton
        label="Show success toast"
        onClick={() => notify.success("Successfully added script.")}
      />
    </>
  ),
};

/**
 * Error — click the button to fire a plain error toast (no detail payload).
 */
export const Error: Story = {
  render: () => (
    <>
      <ToastNotification />
      <TriggerButton
        label="Show error toast"
        onClick={() =>
          notify.error("Failed to save settings. Please try again.")
        }
      />
    </>
  ),
};

/**
 * AllVariants — trigger each variant side by side for quick visual comparison.
 */
export const AllVariants: Story = {
  render: () => (
    <>
      <ToastNotification />
      <div style={{ display: "flex", flexWrap: "wrap", gap: "8px" }}>
        <TriggerButton
          label="Success"
          onClick={() => notify.success("Successfully added script.")}
        />
        <TriggerButton
          label="Error"
          onClick={() =>
            notify.error("Failed to save settings. Please try again.")
          }
        />
        <TriggerButton label="Dismiss all" onClick={() => notify.dismiss()} />
      </div>
    </>
  ),
};

/**
 * ExpandableError — error toast with an HTTP response. The chevron expands
 * a panel showing the response body as syntax-highlighted JSON; the label
 * above it auto-derives from the status code ("Status: 422 Unprocessable
 * Entity").
 */
export const ExpandableError: Story = {
  render: () => (
    <>
      <ToastNotification />
      <TriggerButton
        label="Show expandable error"
        onClick={() =>
          notify.error("Failed to save policy.", {
            response: {
              status: 422,
              statusText: "Unprocessable Entity",
              data: {
                error: "violates foreign key constraint",
                code: 422,
                resource: "policy",
              },
            },
          })
        }
      />
    </>
  ),
};

/**
 * ExpandableErrorLargePayload — verifies the JSON panel scrolls internally
 * when the payload exceeds the panel's `max-height`.
 */
export const ExpandableErrorLargePayload: Story = {
  render: () => {
    const largeBody = {
      error: "Request validation failed",
      code: 422,
      timestamp: "2026-04-15T12:34:56Z",
      resource: "policy",
      request_id: "req_9f3b2a1e-7c4d-4e5f-a1b2-c3d4e5f6a7b8",
      errors: Array.from({ length: 15 }, (_, i) => ({
        field: `rules[${i}].query`,
        message:
          "Query contains an unsupported table reference and must be rewritten against the approved schema.",
        severity: i % 2 === 0 ? "error" : "warning",
        suggestion: {
          replace: "osquery_info",
          with: "fleet_info",
          docs:
            "https://fleetdm.com/docs/using-fleet/example-queries#fleet-info",
        },
      })),
      metadata: {
        environment: "production",
        region: "us-east-1",
        tenant: "acme-corp",
        user_id: 42,
      },
    };

    return (
      <>
        <ToastNotification />
        <TriggerButton
          label="Show expandable error (large payload)"
          onClick={() =>
            notify.error("Failed to validate policy rules.", {
              response: {
                status: 422,
                statusText: "Unprocessable Entity",
                data: largeBody,
              },
            })
          }
        />
      </>
    );
  },
};
