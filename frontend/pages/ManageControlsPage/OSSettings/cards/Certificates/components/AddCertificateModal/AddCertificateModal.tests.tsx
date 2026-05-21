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

const NAME_PLACEHOLDER = "VPN certificate";
const SUBJECT_NAME_PLACEHOLDER =
  "CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME, O=Your Organization";
const SAN_PLACEHOLDER =
  "UPN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME, EMAIL=$FLEET_VAR_HOST_END_USER_IDP_USERNAME";

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

// Captures every POST /certificates body so multi-call tests can inspect the full sequence.
const addCertCalls: Array<Record<string, unknown>> = [];
const addCertHandler = http.post(
  baseUrl("/certificates"),
  async ({ request }) => {
    addCertCalls.push((await request.json()) as Record<string, unknown>);
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

// Renders the modal and waits for the form to be interactive (the Name input present).
// Returns userEvent + the rendered scope.
const renderModal = async ({ existingCerts = [] as ICertificate[] } = {}) => {
  const render = createCustomRenderer({ withBackendMock: true });
  const result = render(
    <AddCertModal
      existingCerts={existingCerts}
      onExit={mockOnExit}
      onSuccess={mockOnSuccess}
    />
  );
  await screen.findByPlaceholderText(NAME_PLACEHOLDER);
  return result;
};

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
    addCertCalls.length = 0;
    mockOnExit.mockClear();
    mockOnSuccess.mockClear();
    mockServer.use(getCAsHandler);
    mockServer.use(addCertHandler);
  });
  afterEach(() => {
    mockServer.resetHandlers();
  });

  it("renders the SAN field alongside the existing fields", async () => {
    await renderModal();
    expect(
      screen.getByText("Subject alternative name (SAN)")
    ).toBeInTheDocument();
    expect(screen.getByPlaceholderText(SAN_PLACEHOLDER)).toBeInTheDocument();
  });

  it("clicking Add with all required fields empty shows three inline errors and does not call the API", async () => {
    const { user } = await renderModal();
    await user.click(screen.getByRole("button", { name: /Add/i }));

    await waitFor(() => {
      expect(screen.getByText(NAME_REQUIRED_MSG)).toBeInTheDocument();
    });
    expect(screen.getByText(CA_REQUIRED_MSG)).toBeInTheDocument();
    expect(screen.getByText(SUBJECT_NAME_REQUIRED_MSG)).toBeInTheDocument();
    expect(addCertCalls).toHaveLength(0);
    expect(mockOnSuccess).not.toHaveBeenCalled();
  });

  it("clicking Add with only Subject name empty shows exactly that one inline error", async () => {
    const { user } = await renderModal();

    await user.type(
      screen.getByPlaceholderText(NAME_PLACEHOLDER),
      "Valid Name"
    );
    await selectScepCa(user);

    await user.click(screen.getByRole("button", { name: /Add/i }));

    await waitFor(() => {
      expect(screen.getByText(SUBJECT_NAME_REQUIRED_MSG)).toBeInTheDocument();
    });
    expect(screen.queryByText(NAME_REQUIRED_MSG)).not.toBeInTheDocument();
    expect(screen.queryByText(CA_REQUIRED_MSG)).not.toBeInTheDocument();
    expect(addCertCalls).toHaveLength(0);
  });

  it("shows inline error for Name with invalid characters as user types (no submit needed)", async () => {
    const { user } = await renderModal();

    await user.type(
      screen.getByPlaceholderText(NAME_PLACEHOLDER),
      "Invalid@Name#"
    );

    await waitFor(() => {
      expect(screen.getByText(INVALID_NAME_MSG)).toBeInTheDocument();
    });
  });

  it("shows inline error for duplicate Name as user types", async () => {
    const { user } = await renderModal({ existingCerts: mockExistingCerts });

    await user.type(
      screen.getByPlaceholderText(NAME_PLACEHOLDER),
      "Existing Certificate"
    );

    await waitFor(() => {
      expect(screen.getByText(USED_NAME_MSG)).toBeInTheDocument();
    });
  });

  it("shows inline error for Name longer than 255 characters as user types", async () => {
    const { user } = await renderModal();

    // Paste rather than type to keep the test fast (256 simulated keypresses is slow).
    await user.click(screen.getByPlaceholderText(NAME_PLACEHOLDER));
    await user.paste("a".repeat(256));

    await waitFor(() => {
      expect(screen.getByText(NAME_TOO_LONG_MSG)).toBeInTheDocument();
    });
  });

  it("submits successfully without SAN (field omitted from request body)", async () => {
    const { user } = await renderModal();

    await user.type(
      screen.getByPlaceholderText(NAME_PLACEHOLDER),
      "Valid Name"
    );
    await user.type(
      screen.getByPlaceholderText(SUBJECT_NAME_PLACEHOLDER),
      "/CN=test/O=Org"
    );
    await selectScepCa(user);
    await user.click(screen.getByRole("button", { name: /Add/i }));

    await waitFor(() => {
      expect(mockOnSuccess).toHaveBeenCalledTimes(1);
    });
    expect(addCertCalls).toHaveLength(1);
    expect(addCertCalls[0]).not.toHaveProperty("subject_alternative_name");
  });

  it("submits successfully with SAN (field included in request body)", async () => {
    const { user } = await renderModal();

    await user.type(
      screen.getByPlaceholderText(NAME_PLACEHOLDER),
      "Valid Name"
    );
    await user.type(
      screen.getByPlaceholderText(SUBJECT_NAME_PLACEHOLDER),
      "/CN=test/O=Org"
    );
    await user.type(
      screen.getByPlaceholderText(SAN_PLACEHOLDER),
      "DNS=host.example.com, EMAIL=user@example.com"
    );
    await selectScepCa(user);
    await user.click(screen.getByRole("button", { name: /Add/i }));

    await waitFor(() => {
      expect(mockOnSuccess).toHaveBeenCalledTimes(1);
    });
    expect(addCertCalls[0]).toMatchObject({
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
              { name: "subject_alternative_name", reason: SERVER_SAN_ERR },
            ],
          },
          { status: 422 }
        );
      })
    );

    const { user } = await renderModal();

    await user.type(
      screen.getByPlaceholderText(NAME_PLACEHOLDER),
      "Valid Name"
    );
    await user.type(
      screen.getByPlaceholderText(SUBJECT_NAME_PLACEHOLDER),
      "/CN=test/O=Org"
    );
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

    const { user } = await renderModal();

    await user.type(
      screen.getByPlaceholderText(NAME_PLACEHOLDER),
      "Valid Name"
    );
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
    const { user } = await renderModal();
    await user.click(screen.getByText("Cancel"));
    expect(mockOnExit).toHaveBeenCalledTimes(1);
  });
});
