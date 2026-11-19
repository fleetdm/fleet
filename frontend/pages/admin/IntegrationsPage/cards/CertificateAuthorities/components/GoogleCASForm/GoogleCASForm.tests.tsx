import { screen } from "@testing-library/react";
import { UserEvent } from "@testing-library/user-event";
import { noop } from "lodash";
import React from "react";

import { renderWithSetup } from "test/test-utils";
import { UNCHANGED_PASSWORD_API_RESPONSE } from "utilities/constants";

import GoogleCASForm, { IGoogleCASFormData } from "./GoogleCASForm";

const createTestFormData = (overrides?: Partial<IGoogleCASFormData>) => ({
  name: "GOOGLE_WIFI",
  projectId: "my-project",
  caPool: "my-ca-pool",
  location: "us-central1",
  apiKeyJson: JSON.stringify({
    type: "service_account",
    client_email: "fleet@my-project.iam.gserviceaccount.com",
    private_key: "test-private-key",
  }),
  ...overrides,
});

type IFormProps = React.ComponentProps<typeof GoogleCASForm>;

const renderForm = (props?: Partial<IFormProps>) =>
  renderWithSetup(
    <GoogleCASForm
      isSubmitting={false}
      submitBtnText="Add CA"
      onSubmit={noop}
      onCancel={noop}
      {...props}
    />
  );

// Pasted rather than typed, because user.type reads "{" in the key's JSON as a
// key name.
const pasteInto = async (user: UserEvent, label: string, value: string) => {
  await user.click(screen.getByLabelText(label));
  await user.paste(value);
};

const fillForm = async (user: UserEvent, formData: IGoogleCASFormData) => {
  await pasteInto(user, "Name", formData.name);
  await pasteInto(user, "Project ID", formData.projectId);
  await pasteInto(user, "CA pool", formData.caPool);
  await pasteInto(user, "Location", formData.location);
  await pasteInto(user, "Service account key", formData.apiKeyJson);
};

describe("GoogleCASForm", () => {
  it("renders the fields in the designed order with links to the Google Cloud console", () => {
    renderForm();

    const fieldNames = Array.from(
      document.querySelectorAll<HTMLInputElement>("form input")
    ).map((input) => input.name);
    expect(fieldNames).toEqual([
      "name",
      "projectId",
      "caPool",
      "location",
      "apiKeyJson",
    ]);
    expect(
      screen.getByRole("link", { name: /CA pool manager/ })
    ).toHaveAttribute(
      "href",
      "https://console.cloud.google.com/security/cas/caPools"
    );
    expect(
      screen.getByRole("link", { name: /Google Cloud's Service accounts/ })
    ).toHaveAttribute(
      "href",
      "https://console.cloud.google.com/iam-admin/serviceaccounts"
    );
  });

  it("hides the service account key", () => {
    renderForm();
    expect(screen.getByLabelText("Service account key")).toHaveAttribute(
      "type",
      "password"
    );
  });

  it("keeps submit enabled and shows every missing field on submit", async () => {
    const onSubmit = jest.fn();
    const { user } = renderForm({ onSubmit });

    await user.click(screen.getByRole("button", { name: "Add CA" }));

    expect(screen.getByText("Enter a name")).toBeInTheDocument();
    expect(screen.getByText("Enter a project ID")).toBeInTheDocument();
    expect(screen.getByText("Enter a CA pool")).toBeInTheDocument();
    expect(screen.getByText("Enter a location")).toBeInTheDocument();
    expect(screen.getByText("Enter a service account key")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add CA" })).toBeEnabled();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it.each([
    [
      "Project ID",
      "my/project",
      "Enter a project ID with only lowercase letters, numbers, hyphens, periods, and colons",
    ],
    [
      "CA pool",
      "pool/certificates/other",
      "Enter a CA pool with only letters, numbers, hyphens, and underscores",
    ],
    [
      "Location",
      "US-CENTRAL1",
      "Enter a location with only lowercase letters, numbers, and hyphens",
    ],
    [
      "Service account key",
      "not json",
      "Enter the service account key file's JSON",
    ],
  ])(
    "shows an error when %s isn't valid and the field loses focus",
    async (label, value, message) => {
      const { user } = renderForm();

      await user.type(screen.getByLabelText(label), value);
      await user.tab();

      expect(screen.getByText(message)).toBeInTheDocument();
    }
  );

  const certAuthorities = [
    { id: 1, name: "TAKEN", type: "google_cas" as const },
    { id: 2, name: "HYDRANT_NAME", type: "hydrant" as const },
  ];

  it("rejects a name another Google CAS CA already uses", async () => {
    const onSubmit = jest.fn();
    const { user } = renderForm({ certAuthorities, onSubmit });

    await fillForm(user, createTestFormData({ name: "TAKEN" }));
    await user.click(screen.getByRole("button", { name: "Add CA" }));

    expect(
      screen.getByText("Enter a name that isn't used by another Google CAS CA")
    ).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("accepts a name used by a CA of another type", async () => {
    const onSubmit = jest.fn();
    const { user } = renderForm({ certAuthorities, onSubmit });

    await fillForm(user, createTestFormData({ name: "HYDRANT_NAME" }));
    await user.click(screen.getByRole("button", { name: "Add CA" }));

    expect(onSubmit).toHaveBeenCalledTimes(1);
  });

  it("rejects the masked key when adding", async () => {
    const onSubmit = jest.fn();
    const { user } = renderForm({ onSubmit });

    await fillForm(
      user,
      createTestFormData({ apiKeyJson: UNCHANGED_PASSWORD_API_RESPONSE })
    );
    await user.click(screen.getByRole("button", { name: "Add CA" }));

    expect(
      screen.getByText("Enter the service account key file's JSON")
    ).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("submits the trimmed values", async () => {
    const onSubmit = jest.fn();
    const { user } = renderForm({ onSubmit });

    await fillForm(
      user,
      createTestFormData({ name: " GOOGLE_WIFI ", location: "us-central1 " })
    );
    await user.click(screen.getByRole("button", { name: "Add CA" }));

    expect(onSubmit).toHaveBeenCalledTimes(1);
    expect(onSubmit).toHaveBeenCalledWith(createTestFormData());
  });

  it("submits a saved CA unchanged, keeping its masked key, when editing", async () => {
    const onSubmit = jest.fn();
    const saved = createTestFormData({
      apiKeyJson: UNCHANGED_PASSWORD_API_RESPONSE,
    });
    const { user } = renderForm({
      defaultData: saved,
      submitBtnText: "Save",
      certAuthorities: [{ id: 1, name: "GOOGLE_WIFI", type: "google_cas" }],
      onSubmit,
    });

    expect(screen.getByLabelText("Name")).toHaveValue("GOOGLE_WIFI");
    await user.click(screen.getByRole("button", { name: "Save" }));

    expect(onSubmit).toHaveBeenCalledTimes(1);
    expect(onSubmit).toHaveBeenCalledWith(saved);
  });

  it("disables the fields and submit while submitting", () => {
    renderForm({ isSubmitting: true });

    expect(screen.getByRole("button", { name: "Add CA" })).toBeDisabled();
    expect(screen.getByLabelText("Name")).toBeDisabled();
    expect(screen.getByLabelText("Service account key")).toBeDisabled();
  });
});
