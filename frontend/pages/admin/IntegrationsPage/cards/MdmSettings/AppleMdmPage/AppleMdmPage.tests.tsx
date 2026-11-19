import { screen, waitFor } from "@testing-library/react";
import React from "react";
import { useQuery } from "react-query";

import { createMockMdmApple } from "__mocks__/appleMdm";
import createMockConfig from "__mocks__/configMock";
import mdmAppleAPI from "services/entities/mdm_apple";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import AppleMdmPage from "./AppleMdmPage";

const NEAR_EXPIRY_RENEW_DATE = "2026-11-12T00:00:00Z";
const RENEWED_RENEW_DATE = "2027-11-02T00:00:00Z";

const AppWideApnsBannerData = () => {
  const { data } = useQuery(["apns"], () => mdmAppleAPI.getAppleAPNInfo());
  return <p>App-wide APNs renew date: {data?.renew_date}</p>;
};

describe("AppleMdmPage", () => {
  const mockConfig = createMockConfig();
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        config: {
          ...mockConfig,
          mdm: { ...mockConfig.mdm, enabled_and_configured: true },
        },
      },
    },
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it("refreshes the app-wide APNs banner data after the certificate is renewed", async () => {
    let renewDate = NEAR_EXPIRY_RENEW_DATE;
    jest
      .spyOn(mdmAppleAPI, "getAppleAPNInfo")
      .mockImplementation(() =>
        Promise.resolve(createMockMdmApple({ renew_date: renewDate }))
      );
    jest
      .spyOn(mdmAppleAPI, "uploadApplePushCertificate")
      .mockImplementation(() => {
        renewDate = RENEWED_RENEW_DATE;
        return Promise.resolve({});
      });

    const { user, container } = render(
      <>
        <AppWideApnsBannerData />
        <AppleMdmPage router={createMockRouter()} />
      </>
    );

    expect(
      await screen.findByText(
        `App-wide APNs renew date: ${NEAR_EXPIRY_RENEW_DATE}`
      )
    ).toBeInTheDocument();

    await user.click(
      await screen.findByRole("button", { name: "Renew certificate" })
    );
    const input = container.querySelector(
      'input[type="file"]'
    ) as HTMLInputElement;
    await user.upload(
      input,
      new File(["cert"], "mdmcert.pem", { type: "application/x-pem-file" })
    );
    const [, modalRenewButton] = screen.getAllByRole("button", {
      name: "Renew certificate",
    });
    await user.click(modalRenewButton);

    await waitFor(() =>
      expect(
        screen.getByText(`App-wide APNs renew date: ${RENEWED_RENEW_DATE}`)
      ).toBeInTheDocument()
    );
  });
});
