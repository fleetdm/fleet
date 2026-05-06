import React from "react";

import { http, HttpResponse } from "msw";
import { screen, waitFor } from "@testing-library/react";
import { ICertificate } from "services/entities/certificates";
import mockServer from "test/mock-server";
import { baseUrl, createCustomRenderer } from "test/test-utils";

import AddCertModal from "./AddCertificateModal";
import {
  CA_REQUIRED_MSG,
  INVALID_NAME_MSG,
  NAME_REQUIRED_MSG,
  NAME_TOO_LONG_MSG,
  SUBJECT_NAME_REQUIRED_MSG,
  USED_NAME_MSG,
} from "./helpers";

const mockOnExit = jest.fn();
const mockOnSuccess = jest.fn();

const SAN_PLACEHOLDER =
  "UPN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME, EMAIL=$FLEET_VAR_HOST_END_USER_IDP_USERNAME";
const SUBJECT_NAME_PLACEHOLDER =
  "CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME, O=Your Organization";

const getCAsHandler = http.get(baseUrl("/certificate_authorities"), () => {
  return HttpResponse.json({
    certificate_authorities: [
      {
        id: 1,
        name: "TEST_SCEP_CA",
        type: "custom_scep_proxy",
      },
    ],
  });
});

// Default handler that records the request body for assertions and returns 200.
let lastAddCertBody: Record<string, unknown> | null = null;
const addCertHandler = http.post(
  baseUrl("/certificates"),
  async ({ request }) => {
    lastAddCertBody = (await request.json()) as Record<string, unknown>;
    return HttpResponse.json({
      id: 123,
      name: "New Certificate",
      certificate_authority_id: 1,
      subject_name: "Test subject name",
      created_at: new Date().toISOString(),
    });
  }
);

const mockExistingCerts: ICertificate[] = [
  {
    id: 1,
    name: "Existing Certificate",
    certificate_authority_id: 1,
    certificate_authority_name: "Test CA 1",
    created_at: "2024-01-01T00:00:00Z",
  },
];

// Pick the SCEP CA option from the dropdown.
const selectScepCa = async (user: {
  click: (el: Element) => Promise<void>;
}) => {
  const caDropdown = screen.getByText("Select certificate authority");
  await user.click(caDropdown);
  await waitFor(() => {
    expect(screen.getByText("TEST_SCEP_CA")).toBeInTheDocument();
  });
  await user.click(screen.getByText("TEST_SCEP_CA"));
};

describe("AddCertModal", () => {
  beforeEach(() => {
    lastAddCertBody = null;
    mockOnExit.mockClear();
    mockOnSuccess.mockClear();
    mockServer.use(getCAsHandler);
    mockServer.use(addCertHandler);
  });
  afterEach(() => {
    mockServer.resetHandlers();
  });

  it("renders the modal with all form fields, including SAN", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });
    render(
      <AddCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    await waitFor(() => {
      expect(screen.queryByTestId("spinner")).not.toBeInTheDocument();
    });

    expect(screen.getByText("Add certificate")).toBeInTheDocument();
    expect(
      await screen.findByPlaceholderText("VPN certificate")
    ).toBeInTheDocument();
    expect(screen.getByText("Certificate authority (CA)")).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText(SUBJECT_NAME_PLACEHOLDER)
    ).toBeInTheDocument();
    expect(
      screen.getByText("Subject alternative name (SAN)")
    ).toBeInTheDocument();
    expect(screen.getByPlaceholderText(SAN_PLACEHOLDER)).toBeInTheDocument();
    expect(screen.getByText("Add")).toBeInTheDocument();
    expect(screen.getByText("Cancel")).toBeInTheDocument();
  });

  it("Add button is enabled when fields are empty (no disabled gate)", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });
    render(
      <AddCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    const addButton = await screen.findByRole("button", { name: /Add/i });
    expect(addButton).not.toBeDisabled();
  });

  it("clicking Add with all required fields empty shows three inline errors and does not call the API", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });
    const { user } = render(
      <AddCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    const addButton = await screen.findByRole("button", { name: /Add/i });
    await user.click(addButton);

    await waitFor(() => {
      expect(screen.getByText(NAME_REQUIRED_MSG)).toBeInTheDocument();
    });
    expect(screen.getByText(CA_REQUIRED_MSG)).toBeInTheDocument();
    expect(screen.getByText(SUBJECT_NAME_REQUIRED_MSG)).toBeInTheDocument();
    expect(lastAddCertBody).toBeNull();
    expect(mockOnSuccess).not.toHaveBeenCalled();
  });

  it("clicking Add with only Subject name empty shows exactly that one inline error", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });
    const { user } = render(
      <AddCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    await waitFor(() => {
      expect(screen.queryByTestId("spinner")).not.toBeInTheDocument();
    });

    const nameInput = await screen.findByPlaceholderText("VPN certificate");
    await user.type(nameInput, "Valid Name");
    await selectScepCa(user);

    const addButton = screen.getByRole("button", { name: /Add/i });
    await user.click(addButton);

    await waitFor(() => {
      expect(screen.getByText(SUBJECT_NAME_REQUIRED_MSG)).toBeInTheDocument();
    });
    expect(screen.queryByText(NAME_REQUIRED_MSG)).not.toBeInTheDocument();
    expect(screen.queryByText(CA_REQUIRED_MSG)).not.toBeInTheDocument();
    expect(lastAddCertBody).toBeNull();
  });

  it("shows inline error for Name with invalid characters as user types (no submit needed)", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });
    const { user } = render(
      <AddCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    await waitFor(() => {
      expect(screen.queryByTestId("spinner")).not.toBeInTheDocument();
    });

    const nameInput = await screen.findByPlaceholderText("VPN certificate");
    await user.type(nameInput, "Invalid@Name#");

    await waitFor(() => {
      expect(screen.getByText(INVALID_NAME_MSG)).toBeInTheDocument();
    });
  });

  it("shows inline error for duplicate Name as user types", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });
    const { user } = render(
      <AddCertModal
        existingCerts={mockExistingCerts}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    await waitFor(() => {
      expect(screen.queryByTestId("spinner")).not.toBeInTheDocument();
    });

    const nameInput = await screen.findByPlaceholderText("VPN certificate");
    await user.type(nameInput, "Existing Certificate");

    await waitFor(() => {
      expect(screen.getByText(USED_NAME_MSG)).toBeInTheDocument();
    });
  });

  it("shows inline error for Name longer than 255 characters as user types", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });
    const { user } = render(
      <AddCertModal
        existingCerts={mockExistingCerts}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    await waitFor(() => {
      expect(screen.queryByTestId("spinner")).not.toBeInTheDocument();
    });

    const nameInput = await screen.findByPlaceholderText("VPN certificate");
    await user.type(nameInput, "a".repeat(256));

    await waitFor(() => {
      expect(screen.getByText(NAME_TOO_LONG_MSG)).toBeInTheDocument();
    });
  });

  it("submits successfully without SAN (field omitted from request body)", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });
    const { user } = render(
      <AddCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    await waitFor(() => {
      expect(screen.queryByTestId("spinner")).not.toBeInTheDocument();
    });

    const nameInput = await screen.findByPlaceholderText("VPN certificate");
    await user.type(nameInput, "Valid Name");

    const subjectNameInput = screen.getByPlaceholderText(
      SUBJECT_NAME_PLACEHOLDER
    );
    await user.type(subjectNameInput, "/CN=test/O=Org");

    await selectScepCa(user);

    await user.click(screen.getByRole("button", { name: /Add/i }));

    await waitFor(() => {
      expect(mockOnSuccess).toHaveBeenCalledTimes(1);
    });
    expect(lastAddCertBody).not.toBeNull();
    expect(lastAddCertBody).not.toHaveProperty("subject_alternative_name");
  });

  it("submits successfully with SAN (field included in request body)", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });
    const { user } = render(
      <AddCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    await waitFor(() => {
      expect(screen.queryByTestId("spinner")).not.toBeInTheDocument();
    });

    const nameInput = await screen.findByPlaceholderText("VPN certificate");
    await user.type(nameInput, "Valid Name");

    const subjectNameInput = screen.getByPlaceholderText(
      SUBJECT_NAME_PLACEHOLDER
    );
    await user.type(subjectNameInput, "/CN=test/O=Org");

    const sanInput = screen.getByPlaceholderText(SAN_PLACEHOLDER);
    await user.type(sanInput, "DNS=host.example.com, EMAIL=user@example.com");

    await selectScepCa(user);

    await user.click(screen.getByRole("button", { name: /Add/i }));

    await waitFor(() => {
      expect(mockOnSuccess).toHaveBeenCalledTimes(1);
    });
    expect(lastAddCertBody).toMatchObject({
      subject_alternative_name: "DNS=host.example.com, EMAIL=user@example.com",
    });
  });

  it("surfaces a 422 server error against the SAN input inline", async () => {
    const SERVER_SAN_ERR =
      'subject_alternative_name has unsupported key "FOO". Allowed keys are DNS, EMAIL, UPN, IP, URI';
    mockServer.use(
      http.post(baseUrl("/certificates"), () => {
        return HttpResponse.json(
          {
            message: "Validation Failed",
            errors: [
              {
                name: "subject_alternative_name",
                reason: SERVER_SAN_ERR,
              },
            ],
          },
          { status: 422 }
        );
      })
    );

    const render = createCustomRenderer({
      withBackendMock: true,
    });
    const { user } = render(
      <AddCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    await waitFor(() => {
      expect(screen.queryByTestId("spinner")).not.toBeInTheDocument();
    });

    const nameInput = await screen.findByPlaceholderText("VPN certificate");
    await user.type(nameInput, "Valid Name");

    const subjectNameInput = screen.getByPlaceholderText(
      SUBJECT_NAME_PLACEHOLDER
    );
    await user.type(subjectNameInput, "/CN=test/O=Org");

    const sanInput = screen.getByPlaceholderText(SAN_PLACEHOLDER);
    await user.type(sanInput, "FOO=bar");

    await selectScepCa(user);

    await user.click(screen.getByRole("button", { name: /Add/i }));

    await waitFor(() => {
      expect(screen.getByText(SERVER_SAN_ERR)).toBeInTheDocument();
    });
    expect(mockOnSuccess).not.toHaveBeenCalled();

    // Editing the SAN clears the server error.
    await user.type(sanInput, ", DNS=host.example.com");
    await waitFor(() => {
      expect(screen.queryByText(SERVER_SAN_ERR)).not.toBeInTheDocument();
    });
  });

  it("Add button is disabled while the POST is in flight, then re-enabled on response", async () => {
    // Replace the default handler with one that won't resolve until we say so.
    let resolveServer!: () => void;
    const serverGate = new Promise<void>((resolve) => {
      resolveServer = resolve;
    });
    mockServer.use(
      http.post(baseUrl("/certificates"), async () => {
        await serverGate;
        return HttpResponse.json({
          id: 123,
          name: "New Certificate",
          certificate_authority_id: 1,
          subject_name: "Test subject name",
          created_at: new Date().toISOString(),
        });
      })
    );

    const render = createCustomRenderer({
      withBackendMock: true,
    });
    const { user } = render(
      <AddCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    const nameInput = await screen.findByPlaceholderText("VPN certificate");
    await user.type(nameInput, "Valid Name");
    await user.type(
      screen.getByPlaceholderText(SUBJECT_NAME_PLACEHOLDER),
      "/CN=test/O=Org"
    );
    await selectScepCa(user);

    const addButton = screen.getByRole("button", { name: /Add/i });
    // user.click awaits internal pointer events but the click handler kicks
    // off the POST without awaiting it, so the disabled flip happens
    // synchronously before resolveServer() is called.
    await user.click(addButton);

    expect(addButton).toBeDisabled();

    resolveServer();
    await waitFor(() => {
      expect(mockOnSuccess).toHaveBeenCalledTimes(1);
    });
  });

  it("calls onExit when Cancel button is clicked", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });
    const { user } = render(
      <AddCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    await waitFor(() => {
      expect(screen.queryByTestId("spinner")).not.toBeInTheDocument();
    });

    const cancelButton = await screen.findByText("Cancel");
    await user.click(cancelButton);

    expect(mockOnExit).toHaveBeenCalledTimes(1);
  });
});
