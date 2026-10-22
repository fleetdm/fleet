import { screen, waitFor } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { createMockCertificateAuthorityPartial } from "__mocks__/certificatesMock";
import { renderWithSetup } from "test/test-utils";

import EJBCAForm, { EMPTY_EJBCA_FORM_DATA } from "./EJBCAForm";

const getFileInput = (container: HTMLElement) =>
  container.querySelector('input[type="file"]') as HTMLInputElement;

const p12File = () =>
  new File([new Uint8Array([1, 2, 3])], "client.p12", {
    type: "application/x-pkcs12",
  });

describe("EJBCAForm", () => {
  it("shows every error on submit and doesn't submit when the form is empty", async () => {
    const onSubmit = jest.fn();
    const { user } = renderWithSetup(
      <EJBCAForm submitBtnText="Add CA" onSubmit={onSubmit} onCancel={noop} />
    );

    // the button stays enabled, so that clicking it shows what is missing
    await user.click(screen.getByRole("button", { name: "Add CA" }));

    expect(screen.getByText("Enter a name")).toBeVisible();
    expect(screen.getByText("Enter a URL")).toBeVisible();
    expect(
      screen.getByText("Upload a client certificate (.p12)")
    ).toBeVisible();
    expect(screen.getByText("Enter an EJBCA CA name")).toBeVisible();
    expect(screen.getByText("Enter a common name")).toBeVisible();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("submits what was entered, with the client certificate as base64", async () => {
    const onSubmit = jest.fn();
    const { user, container } = renderWithSetup(
      <EJBCAForm submitBtnText="Add CA" onSubmit={onSubmit} onCancel={noop} />
    );

    await user.type(screen.getByLabelText("Name"), "WIFI_CERTIFICATE");
    await user.type(screen.getByLabelText("URL"), "https://ejbca.example.com");
    await user.upload(getFileInput(container), p12File());
    await user.type(
      screen.getByLabelText("Client certificate password"),
      " p12-password "
    );
    await user.type(screen.getByLabelText("CA name"), "ManagementCA");
    await user.type(
      screen.getByLabelText("Certificate profile name"),
      "ENDUSER"
    );
    await user.type(screen.getByLabelText("End entity profile name"), "EMPTY");
    await user.type(
      screen.getByLabelText("Certificate common name (CN)"),
      "$FLEET_VAR_HOST_HARDWARE_SERIAL"
    );
    await user.click(screen.getByRole("button", { name: "Add CA" }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit).toHaveBeenCalledWith({
      ...EMPTY_EJBCA_FORM_DATA,
      name: "WIFI_CERTIFICATE",
      url: "https://ejbca.example.com",
      clientP12: "AQID",
      // the password is sent as typed, spaces included
      password: " p12-password ",
      caName: "ManagementCA",
      certificateProfileName: "ENDUSER",
      endEntityProfileName: "EMPTY",
      commonName: "$FLEET_VAR_HOST_HARDWARE_SERIAL",
    });
  });

  it("says when the name is used by another certificate authority", async () => {
    const { user } = renderWithSetup(
      <EJBCAForm
        certAuthorities={[
          createMockCertificateAuthorityPartial({
            name: "Taken_Name",
            type: "custom_scep_proxy",
          }),
        ]}
        submitBtnText="Add CA"
        onSubmit={noop}
        onCancel={noop}
      />
    );

    await user.type(screen.getByLabelText("Name"), "taken_name");
    await user.click(screen.getByRole("button", { name: "Add CA" }));

    expect(
      screen.getByText(
        "Enter a name that isn't used by another certificate authority"
      )
    ).toBeVisible();
  });

  it("shows the errors that the server reported for the fields", () => {
    renderWithSetup(
      <EJBCAForm
        serverErrors={{
          password: "The PKCS#12 password is incorrect or the file is damaged",
        }}
        submitBtnText="Add CA"
        onSubmit={noop}
        onCancel={noop}
      />
    );

    expect(
      screen.getByText(
        "The PKCS#12 password is incorrect or the file is damaged"
      )
    ).toBeVisible();
  });

  describe("when editing", () => {
    const editProps = {
      editingCertAuthorityId: 1,
      initialFormData: {
        ...EMPTY_EJBCA_FORM_DATA,
        name: "TEST_EJBCA",
        url: "https://ejbca.example.com",
        caName: "ManagementCA",
        certificateProfileName: "ENDUSER",
        endEntityProfileName: "EMPTY",
        commonName: "cn",
      },
      clientCertExpiresAt: "2099-01-01T00:00:00Z",
      submitBtnText: "Save",
      onCancel: noop,
    };

    it("shows when the stored client certificate expires, and asks for its password only with a new one", async () => {
      const { user, container } = renderWithSetup(
        <EJBCAForm {...editProps} onSubmit={noop} />
      );

      expect(screen.getByText("January 1, 2099")).toBeVisible();
      expect(screen.queryByLabelText("Client certificate password")).toBeNull();

      await user.upload(getFileInput(container), p12File());
      expect(
        screen.getByLabelText("Client certificate password")
      ).toBeVisible();
    });

    it("saves without a new client certificate", async () => {
      const onSubmit = jest.fn();
      const { user } = renderWithSetup(
        <EJBCAForm {...editProps} onSubmit={onSubmit} />
      );

      await user.click(screen.getByRole("button", { name: "Save" }));

      await waitFor(() =>
        expect(onSubmit).toHaveBeenCalledWith(editProps.initialFormData)
      );
    });
  });
});
