import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import { notify } from "components/ToastNotification";
import mockServer from "test/mock-server";
import { baseUrl, createCustomRenderer } from "test/test-utils";

import OktaConditionalAccessModal, {
  validate,
} from "./OktaConditionalAccessModal";

jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    dismiss: jest.fn(),
  },
}));

const TEST_OKTA_IDP_ID = "okta-idp-123";
const TEST_OKTA_ACS_URL = "https://example.com/acs";
const TEST_OKTA_AUDIENCE_URI = "https://example.com";
const TEST_OKTA_CERTIFICATE = "cert-data";

const createOktaConfig = () =>
  createMockConfig({
    conditional_access: {
      microsoft_entra_tenant_id: "",
      microsoft_entra_connection_configured: false,
      okta_idp_id: TEST_OKTA_IDP_ID,
      okta_assertion_consumer_service_url: TEST_OKTA_ACS_URL,
      okta_audience_uri: TEST_OKTA_AUDIENCE_URI,
      okta_certificate: TEST_OKTA_CERTIFICATE,
      bypass_disabled: false,
    },
  });

const renderModal = () => {
  const onSuccess = jest.fn();
  const render = createCustomRenderer({
    withBackendMock: true,
    context: { app: { isPremiumTier: true, config: createOktaConfig() } },
  });
  const utils = render(
    <OktaConditionalAccessModal onCancel={jest.fn()} onSuccess={onSuccess} />
  );
  return { ...utils, onSuccess };
};

describe("OktaConditionalAccessModal validate", () => {
  const data = {
    okta_idp_id: "idp",
    okta_acs_url: "https://example.okta.com/acs",
    okta_audience_uri: "audience",
    okta_certificate: "cert",
  };

  it("accepts a valid ACS URL with surrounding whitespace", () => {
    const errors = validate({
      ...data,
      okta_acs_url: ` ${data.okta_acs_url} `,
    });

    expect(errors.okta_acs_url).toBeUndefined();
  });

  it("still rejects an invalid ACS URL", () => {
    const errors = validate({ ...data, okta_acs_url: " not a url " });

    expect(errors.okta_acs_url).toBe(
      "Enter a valid URL with http or https scheme"
    );
  });
});

describe("OktaConditionalAccessModal", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("prefills every field from the saved Okta settings", () => {
    renderModal();

    expect(screen.getByLabelText("IdP ID")).toHaveValue(TEST_OKTA_IDP_ID);
    expect(screen.getByLabelText("Assertion consumer service URL")).toHaveValue(
      TEST_OKTA_ACS_URL
    );
    expect(screen.getByLabelText("Audience URI")).toHaveValue(
      TEST_OKTA_AUDIENCE_URI
    );
  });

  it("saves edited settings with the saved certificate, without re-uploading it", async () => {
    // Typed through `as` so TS doesn't narrow it to null; it's set in the handler.
    let capturedBody = null as {
      conditional_access?: { okta_idp_id?: string; okta_certificate?: string };
    } | null;
    const updatedConfig = createOktaConfig();
    mockServer.use(
      http.patch(baseUrl("/config"), async ({ request }) => {
        capturedBody = (await request.json()) as typeof capturedBody;
        return HttpResponse.json(updatedConfig);
      })
    );

    const { user, onSuccess } = renderModal();

    const idpIdInput = screen.getByLabelText("IdP ID");
    await user.clear(idpIdInput);
    await user.type(idpIdInput, "new-idp-id");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalled();
    });
    expect(capturedBody?.conditional_access).toEqual(
      expect.objectContaining({
        okta_idp_id: "new-idp-id",
        okta_certificate: TEST_OKTA_CERTIFICATE,
      })
    );
    expect(notify.success).toHaveBeenCalledWith(
      "Successfully updated Okta conditional access"
    );
  });

  it("shows the server's reason when the signing certificate download fails", async () => {
    mockServer.use(
      http.get(baseUrl("/conditional_access/idp/signing_cert"), () =>
        HttpResponse.json(
          {
            message: "Bad request",
            errors: [{ name: "base", reason: "signing certificate expired" }],
          },
          { status: 400 }
        )
      )
    );

    const { user } = renderModal();

    await user.click(screen.getByRole("button", { name: /Download/ }));

    await waitFor(() => {
      expect(notify.error).toHaveBeenCalledWith(
        expect.stringContaining("signing certificate expired"),
        expect.anything()
      );
    });
  });
});
