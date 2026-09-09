import React from "react";
import { screen } from "@testing-library/react";
import { renderWithSetup, createMockRouter } from "test/test-utils";

import createMockConfig from "__mocks__/configMock";

import GlobalHostStatusWebhook from "./GlobalHostStatusWebhook";

const REQUIRED_URL_ERROR = "Destination URL must be present";
const INVALID_URL_ERROR = "Destination URL is not a valid URL";
const URL_PLACEHOLDER = "https://server.com/example";
const ENABLE_LABEL = "Enable host status webhook";

const baseConfig = createMockConfig();

// The webhook starts disabled with an empty URL so we can exercise enabling it.
const disabledWebhookConfig = {
  ...baseConfig,
  webhook_settings: {
    ...baseConfig.webhook_settings,
    host_status_webhook: {
      enable_host_status_webhook: false,
      destination_url: "",
      host_percentage: 1,
      days_count: 1,
    },
  },
};

const renderCard = (handleSubmit = jest.fn()) => {
  const utils = renderWithSetup(
    <GlobalHostStatusWebhook
      appConfig={disabledWebhookConfig}
      handleSubmit={handleSubmit}
      isUpdatingSettings={false}
      router={createMockRouter()}
    />
  );
  return { ...utils, handleSubmit };
};

describe("GlobalHostStatusWebhook - Destination URL validation", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("does not show an error when the webhook is first enabled (#40410)", async () => {
    const { user } = renderCard();

    await user.click(screen.getByText(ENABLE_LABEL));

    expect(screen.getByPlaceholderText(URL_PLACEHOLDER)).toBeInTheDocument();
    expect(screen.queryByText(REQUIRED_URL_ERROR)).not.toBeInTheDocument();
    expect(screen.queryByText(INVALID_URL_ERROR)).not.toBeInTheDocument();
  });

  it("shows an error when the URL field is blurred while empty", async () => {
    const { user } = renderCard();

    await user.click(screen.getByText(ENABLE_LABEL));
    await user.click(screen.getByPlaceholderText(URL_PLACEHOLDER));
    await user.tab();

    expect(await screen.findByText(REQUIRED_URL_ERROR)).toBeInTheDocument();
  });

  it("shows an error when the URL field is blurred with an invalid URL", async () => {
    const { user } = renderCard();

    await user.click(screen.getByText(ENABLE_LABEL));
    await user.type(screen.getByPlaceholderText(URL_PLACEHOLDER), "not-a-url");
    await user.tab();

    expect(await screen.findByText(INVALID_URL_ERROR)).toBeInTheDocument();
  });

  it("clears the error once a URL is entered", async () => {
    const { user } = renderCard();

    await user.click(screen.getByText(ENABLE_LABEL));
    const urlInput = screen.getByPlaceholderText(URL_PLACEHOLDER);
    await user.click(urlInput);
    await user.tab();
    expect(await screen.findByText(REQUIRED_URL_ERROR)).toBeInTheDocument();

    await user.type(urlInput, "https://example.com");

    expect(screen.queryByText(REQUIRED_URL_ERROR)).not.toBeInTheDocument();
  });

  it("blocks submit and shows an error when the URL is empty", async () => {
    const { user, handleSubmit } = renderCard();

    await user.click(screen.getByText(ENABLE_LABEL));
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(await screen.findByText(REQUIRED_URL_ERROR)).toBeInTheDocument();
    expect(handleSubmit).not.toHaveBeenCalled();
  });

  it("submits when the URL is valid", async () => {
    const { user, handleSubmit } = renderCard();

    await user.click(screen.getByText(ENABLE_LABEL));
    await user.type(
      screen.getByPlaceholderText(URL_PLACEHOLDER),
      "https://example.com"
    );
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(handleSubmit).toHaveBeenCalled();
  });
});
