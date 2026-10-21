import { screen } from "@testing-library/react";
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

// Holds the form data the way the add and edit modals do, so typed values stick.
const StatefulGoogleCASForm = ({
  initialFormData,
  ...props
}: Omit<IFormProps, "formData" | "onChange"> & {
  initialFormData: IGoogleCASFormData;
}) => {
  const [formData, setFormData] = React.useState(initialFormData);
  return (
    <GoogleCASForm
      {...props}
      formData={formData}
      onChange={({ name, value }) =>
        setFormData((prev) => ({ ...prev, [name]: value }))
      }
    />
  );
};

const renderForm = (
  formData: IGoogleCASFormData,
  props?: Partial<Omit<IFormProps, "formData" | "onChange">>
) =>
  renderWithSetup(
    <StatefulGoogleCASForm
      initialFormData={formData}
      isSubmitting={false}
      submitBtnText="Add CA"
      onSubmit={noop}
      onCancel={noop}
      {...props}
    />
  );

describe("GoogleCASForm", () => {
  it("renders the fields in the designed order with links to the Google Cloud console", () => {
    renderForm(createTestFormData());

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
    renderForm(createTestFormData());
    expect(screen.getByLabelText("Service account key")).toHaveAttribute(
      "type",
      "password"
    );
  });

  it("enables submit for valid data, including the masked key of a saved CA", () => {
    renderForm(
      createTestFormData({ apiKeyJson: UNCHANGED_PASSWORD_API_RESPONSE })
    );
    expect(screen.getByRole("button", { name: "Add CA" })).toBeEnabled();
  });

  it.each([
    [
      "Project ID",
      "my/project",
      "Must contain only lowercase letters, numbers, hyphens, periods and colons.",
    ],
    [
      "CA pool",
      "pool/certificates/other",
      "Must contain only letters, numbers, hyphens and underscores.",
    ],
    [
      "Location",
      "US-CENTRAL1",
      "Must contain only lowercase letters, numbers and hyphens.",
    ],
  ])(
    "shows an error and disables submit when %s has characters Google doesn't allow",
    async (label, value, message) => {
      const { user } = renderForm(createTestFormData());

      await user.clear(screen.getByLabelText(label));
      await user.type(screen.getByLabelText(label), value);

      expect(screen.getByText(message)).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Add CA" })).toBeDisabled();
    }
  );

  it("shows an error when the service account key isn't a JSON object", async () => {
    const { user } = renderForm(createTestFormData());

    await user.clear(screen.getByLabelText("Service account key"));
    await user.type(screen.getByLabelText("Service account key"), "not json");

    expect(
      screen.getByText("Must be the service account key file's JSON.")
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add CA" })).toBeDisabled();
  });

  const certAuthorities = [
    { id: 1, name: "TAKEN", type: "google_cas" as const },
    { id: 2, name: "HYDRANT_NAME", type: "hydrant" as const },
  ];

  it("rejects a name another Google CAS CA already uses", async () => {
    const { user } = renderForm(createTestFormData(), { certAuthorities });

    await user.clear(screen.getByLabelText("Name"));
    await user.type(screen.getByLabelText("Name"), "TAKEN");

    expect(
      screen.getByText("Name is already used by another Google CAS CA.")
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add CA" })).toBeDisabled();
  });

  it("re-checks the name when the list of CAs changes", () => {
    const props = {
      isSubmitting: false,
      submitBtnText: "Add CA",
      onSubmit: noop,
      onCancel: noop,
      initialFormData: createTestFormData({ name: "TAKEN" }),
    };
    const { rerender } = renderWithSetup(
      <StatefulGoogleCASForm {...props} certAuthorities={[]} />
    );
    expect(screen.getByRole("button", { name: "Add CA" })).toBeEnabled();

    rerender(
      <StatefulGoogleCASForm {...props} certAuthorities={certAuthorities} />
    );

    expect(
      screen.getByText("Name is already used by another Google CAS CA.")
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add CA" })).toBeDisabled();
  });

  it("accepts a name used by a CA of another type", async () => {
    const { user } = renderForm(createTestFormData(), { certAuthorities });

    await user.clear(screen.getByLabelText("Name"));
    await user.type(screen.getByLabelText("Name"), "HYDRANT_NAME");

    expect(screen.getByRole("button", { name: "Add CA" })).toBeEnabled();
  });

  it("disables submit while submitting", () => {
    renderForm(createTestFormData(), { isSubmitting: true });
    expect(screen.getByRole("button", { name: "Add CA" })).toBeDisabled();
  });

  it("disables submit when editing without changes", () => {
    renderForm(createTestFormData(), { isEditing: true, isDirty: false });
    expect(screen.getByRole("button", { name: "Add CA" })).toBeDisabled();
  });

  it("submits through onSubmit", async () => {
    const onSubmit = jest.fn();
    const { user } = renderForm(createTestFormData(), { onSubmit });

    await user.click(screen.getByRole("button", { name: "Add CA" }));

    expect(onSubmit).toHaveBeenCalledTimes(1);
  });
});
