import React from "react";
import { noop } from "lodash";
import { screen, waitFor } from "@testing-library/react";

import { createMockConfig } from "__mocks__/configMock";
import { createCustomRenderer } from "test/test-utils";
import configAPI from "services/entities/config";
import { IEndUserAuthentication } from "interfaces/config";

import EndUserAuthSection from "./EndUserAuthSection";

jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    batch: jest.fn(),
    dismiss: jest.fn(),
  },
}));

const EMPTY_END_USER_AUTH: IEndUserAuthentication = {
  idp_name: "",
  entity_id: "",
  metadata_url: "",
  metadata: "",
  issuer_uri: "",
};

const CONFIGURED_END_USER_AUTH: IEndUserAuthentication = {
  idp_name: "Okta",
  entity_id: "https://fleet.example.com",
  metadata_url: "https://idp.example.com/metadata",
  metadata: "",
  issuer_uri: "",
};

const renderEndUserAuthSection = ({
  endUserAuth = EMPTY_END_USER_AUTH,
  onDirtyChange = noop,
  isPremiumTier = true,
}: {
  endUserAuth?: IEndUserAuthentication;
  onDirtyChange?: (hasUnsavedChanges: boolean) => void;
  isPremiumTier?: boolean;
} = {}) => {
  // Deliberately left holding the empty default: the component must read the
  // saved config from its prop, not from here.
  const render = createCustomRenderer({
    context: { app: { isPremiumTier, config: createMockConfig() } },
  });

  return render(
    <EndUserAuthSection
      endUserAuth={endUserAuth}
      onDirtyChange={onDirtyChange}
      onSubmit={noop}
    />
  );
};

describe("EndUserAuthSection", () => {
  const updateSpy = jest.spyOn(configAPI, "update");

  beforeEach(() => {
    updateSpy.mockClear();
    updateSpy.mockResolvedValue(createMockConfig());
  });

  afterAll(() => {
    updateSpy.mockRestore();
  });

  it("shows the premium paywall on free tier", () => {
    renderEndUserAuthSection({ isPremiumTier: false });

    expect(screen.queryByRole("button", { name: "Save" })).toBeNull();
  });

  it("seeds the fields from the config prop rather than AppContext", () => {
    renderEndUserAuthSection({ endUserAuth: CONFIGURED_END_USER_AUTH });

    expect(screen.getByLabelText("Identity provider name")).toHaveValue("Okta");
    expect(screen.getByLabelText("Entity ID")).toHaveValue(
      "https://fleet.example.com"
    );
    expect(screen.getByLabelText("Metadata URL")).toHaveValue(
      "https://idp.example.com/metadata"
    );
  });

  it("drops the errors already on screen as soon as the form is fully emptied", async () => {
    const { user } = renderEndUserAuthSection({
      endUserAuth: CONFIGURED_END_USER_AUTH,
    });

    const idpName = screen.getByLabelText("Identity provider name");
    const entityId = screen.getByLabelText("Entity ID");
    const metadataUrl = screen.getByLabelText("Metadata URL");

    await user.clear(idpName);
    await user.tab();
    expect(
      screen.getByText("Enter an identity provider name")
    ).toBeInTheDocument();

    await user.clear(entityId);
    await user.clear(metadataUrl);

    expect(screen.queryByText("Enter an identity provider name")).toBeNull();
    expect(screen.queryByText("Enter an entity ID")).toBeNull();
  });

  it("keeps Save enabled on an empty form", () => {
    renderEndUserAuthSection();

    expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
  });

  it("keeps Save enabled when required fields are only partially filled", async () => {
    const { user } = renderEndUserAuthSection();

    await user.type(screen.getByLabelText("Identity provider name"), "Okta");

    expect(screen.getByRole("button", { name: "Save" })).toBeEnabled();
  });

  it("does not light up untouched fields when moving out of one field", async () => {
    const { user } = renderEndUserAuthSection();

    await user.type(
      screen.getByLabelText("Entity ID"),
      "https://fleet.example.com"
    );
    await user.tab();

    expect(screen.queryByText("Enter an identity provider name")).toBeNull();
    expect(screen.queryByText("Enter metadata or a metadata URL")).toBeNull();
  });

  it("shows an error on the blurred field once its own value is invalid", async () => {
    const { user } = renderEndUserAuthSection({
      endUserAuth: CONFIGURED_END_USER_AUTH,
    });

    await user.clear(screen.getByLabelText("Identity provider name"));
    await user.tab();

    expect(
      screen.getByText("Enter an identity provider name")
    ).toBeInTheDocument();
    expect(screen.queryByText("Enter an entity ID")).toBeNull();
  });

  it("clears a field's error when the field is focused again", async () => {
    const { user } = renderEndUserAuthSection({
      endUserAuth: CONFIGURED_END_USER_AUTH,
    });

    const idpName = screen.getByLabelText("Identity provider name");
    await user.clear(idpName);
    await user.tab();
    expect(
      screen.getByText("Enter an identity provider name")
    ).toBeInTheDocument();

    await user.click(idpName);

    expect(screen.queryByText("Enter an identity provider name")).toBeNull();
  });

  it("reveals every error on submit and does not call the API", async () => {
    const { user } = renderEndUserAuthSection();

    await user.type(screen.getByLabelText("Identity provider name"), "Okta");
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(screen.getByText("Enter an entity ID")).toBeInTheDocument();
    expect(
      screen.getAllByText("Enter metadata or a metadata URL").length
    ).toBeGreaterThan(0);
    expect(updateSpy).not.toHaveBeenCalled();
  });

  it("reports unsaved changes only while the form differs from what was saved", async () => {
    const onDirtyChange = jest.fn();
    const { user } = renderEndUserAuthSection({ onDirtyChange });

    const idpName = screen.getByLabelText("Identity provider name");
    await user.type(idpName, "Okta");
    expect(onDirtyChange).toHaveBeenLastCalledWith(true);

    await user.clear(idpName);
    expect(onDirtyChange).toHaveBeenLastCalledWith(false);
  });

  it("stops reporting unsaved changes after a successful save", async () => {
    const onDirtyChange = jest.fn();
    const { user } = renderEndUserAuthSection({
      endUserAuth: CONFIGURED_END_USER_AUTH,
      onDirtyChange,
    });

    await user.type(
      screen.getByLabelText("Identity provider name"),
      " Renamed"
    );
    expect(onDirtyChange).toHaveBeenLastCalledWith(true);

    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(onDirtyChange).toHaveBeenLastCalledWith(false));
  });

  it("submits trimmed values", async () => {
    const { user } = renderEndUserAuthSection();

    await user.type(
      screen.getByLabelText("Identity provider name"),
      "  Okta  "
    );
    await user.type(
      screen.getByLabelText("Entity ID"),
      "  https://fleet.example.com  "
    );
    await user.type(
      screen.getByLabelText("Metadata URL"),
      "  https://idp.example.com/metadata  "
    );
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(updateSpy).toHaveBeenCalledWith({
        mdm: {
          end_user_authentication: {
            idp_name: "Okta",
            entity_id: "https://fleet.example.com",
            metadata_url: "https://idp.example.com/metadata",
            metadata: "",
          },
        },
      })
    );
  });

  it("saves an emptied form so the configuration can be cleared", async () => {
    const { user } = renderEndUserAuthSection({
      endUserAuth: CONFIGURED_END_USER_AUTH,
    });

    await user.clear(screen.getByLabelText("Identity provider name"));
    await user.clear(screen.getByLabelText("Entity ID"));
    await user.clear(screen.getByLabelText("Metadata URL"));
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() =>
      expect(updateSpy).toHaveBeenCalledWith({
        mdm: {
          end_user_authentication: {
            idp_name: "",
            entity_id: "",
            metadata_url: "",
            metadata: "",
          },
        },
      })
    );
  });
});
