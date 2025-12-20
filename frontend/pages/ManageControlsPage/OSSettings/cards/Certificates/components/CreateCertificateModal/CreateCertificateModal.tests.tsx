import React from "react";

import { http, HttpResponse } from "msw";
import { screen, waitFor } from "@testing-library/react";
import { ICertificate } from "services/entities/certificates";
import mockServer from "test/mock-server";
import { baseUrl, createCustomRenderer } from "test/test-utils";

import CreateCertModal from "./CreateCertificateModal";
import { INVALID_NAME_MSG, NAME_TOO_LONG_MSG, USED_NAME_MSG } from "./helpers";

const mockOnExit = jest.fn();
const mockOnSuccess = jest.fn();

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

const createCertHandler = http.post(baseUrl("/certificates"), () => {
  return HttpResponse.json({
    id: 123,
    name: "New Certificate",
    certificate_authority_id: 1,
    subject_name: "Test subject name",
    created_at: new Date().toISOString(),
  });
});

const mockExistingCerts: ICertificate[] = [
  {
    id: 1,
    name: "Existing Certificate",
    certificate_authority_id: 1,
    certificate_authority_name: "Test CA 1",
    created_at: "2024-01-01T00:00:00Z",
  },
];

describe("CreateCertModal", () => {
  beforeEach(() => {
    mockServer.use(getCAsHandler);
    mockServer.use(createCertHandler);
  });
  afterEach(() => {
    mockServer.resetHandlers();
  });

  it("renders the modal with all form fields", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });
    render(
      <CreateCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    await waitFor(() => {
      expect(screen.queryByTestId("spinner")).not.toBeInTheDocument();
    });

    expect(screen.getByText("Create certificate")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("VPN certificate")).toBeInTheDocument();
    expect(screen.getByText("Certificate authority (CA)")).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText(
        "CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME, O=Your Organization"
      )
    ).toBeInTheDocument();
    expect(screen.getByText("Create")).toBeInTheDocument();
    expect(screen.getByText("Cancel")).toBeInTheDocument();
  });

  it("disables Create button when Name field is empty", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });
    const { user } = render(
      <CreateCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    await waitFor(() => {
      expect(screen.queryByTestId("spinner")).not.toBeInTheDocument();
    });

    const createButton = screen.getByRole("button", { name: /Create/i });
    expect(createButton).toBeDisabled();

    await user.hover(createButton);
    await waitFor(() => {
      expect(
        screen.getByText("Complete all fields to save.")
      ).toBeInTheDocument();
    });
  });

  it("shows error for Name with invalid characters and disables Create button", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });
    const { user } = render(
      <CreateCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    await waitFor(() => {
      expect(screen.queryByTestId("spinner")).not.toBeInTheDocument();
    });

    const nameInput = screen.getByPlaceholderText("VPN certificate");
    await user.type(nameInput, "Invalid@Name#");

    await waitFor(() => {
      expect(screen.getByText(INVALID_NAME_MSG)).toBeInTheDocument();
    });

    const createButton = screen.getByRole("button", { name: /Create/i });
    expect(createButton).toBeDisabled();
  });

  it("shows error for Name that already exists and disables Create button", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });
    const { user } = render(
      <CreateCertModal
        existingCerts={mockExistingCerts}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    await waitFor(() => {
      expect(screen.queryByTestId("spinner")).not.toBeInTheDocument();
    });

    const nameInput = screen.getByPlaceholderText("VPN certificate");
    await user.type(nameInput, "Existing Certificate");

    await waitFor(() => {
      expect(screen.getByText(USED_NAME_MSG)).toBeInTheDocument();
    });

    const createButton = screen.getByRole("button", { name: /Create/i });
    expect(createButton).toBeDisabled();
  });

  it("shows error for Name with more than 255 characters and disables Create button", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });
    const { user } = render(
      <CreateCertModal
        existingCerts={mockExistingCerts}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    await waitFor(() => {
      expect(screen.queryByTestId("spinner")).not.toBeInTheDocument();
    });

    const nameInput = screen.getByPlaceholderText("VPN certificate");

    const longName = "a".repeat(256);
    await user.type(nameInput, longName);
    await waitFor(() => {
      expect(screen.getByText(NAME_TOO_LONG_MSG)).toBeInTheDocument();
    });

    const createButton = screen.getByRole("button", { name: /Create/i });
    expect(createButton).toBeDisabled();
  });

  it("disables Create button when Certificate authority is not selected", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });
    const { user } = render(
      <CreateCertModal
        existingCerts={mockExistingCerts}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    await waitFor(() => {
      expect(screen.queryByTestId("spinner")).not.toBeInTheDocument();
    });

    const nameInput = screen.getByPlaceholderText("VPN certificate");
    await user.type(nameInput, "Valid Name");

    const subjectNameInput = screen.getByPlaceholderText(
      "CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME, O=Your Organization"
    );
    await user.type(subjectNameInput, "/CN=test/O=Org");

    const createButton = screen.getByRole("button", { name: /Create/i });
    expect(createButton).toBeDisabled();
  });

  it("disables Create button when Subject name is empty", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });
    const { user } = render(
      <CreateCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    await waitFor(() => {
      expect(screen.queryByTestId("spinner")).not.toBeInTheDocument();
    });

    const nameInput = screen.getByPlaceholderText("VPN certificate");
    await user.type(nameInput, "Valid Name");

    const caDropdown = screen.getByText("Select certificate authority");
    await user.click(caDropdown);

    await waitFor(() => {
      expect(screen.getByText("TEST_SCEP_CA")).toBeInTheDocument();
    });

    await user.click(screen.getByText("TEST_SCEP_CA"));

    expect(screen.queryByText("Select certificate authority")).toBeNull();

    const createButton = screen.getByRole("button", { name: /Create/i });
    expect(createButton).toBeDisabled();
  });

  it("full flow is okay when all fields are valid", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });
    const { user } = render(
      <CreateCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    await waitFor(() => {
      expect(screen.queryByTestId("spinner")).not.toBeInTheDocument();
    });

    // Fill in all fields with valid data
    const nameInput = screen.getByPlaceholderText("VPN certificate");
    await user.type(nameInput, "Valid Name");

    const subjectNameInput = screen.getByPlaceholderText(
      "CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME, O=Your Organization"
    );
    await user.type(subjectNameInput, "/CN=test/O=Org");

    const caDropdown = screen.getByText("Select certificate authority");
    await user.click(caDropdown);

    await waitFor(() => {
      expect(screen.getByText("TEST_SCEP_CA")).toBeInTheDocument();
    });
    await user.click(screen.getByText("TEST_SCEP_CA"));
    expect(screen.queryByText("Select certificate authority")).toBeNull();

    const createButton = screen.getByRole("button", { name: /Create/i });
    expect(createButton).not.toBeDisabled();

    await user.click(createButton);

    expect(mockOnSuccess).toHaveBeenCalledTimes(1);
  });

  it("calls onExit when Cancel button is clicked", async () => {
    const render = createCustomRenderer({
      withBackendMock: true,
    });
    const { user } = render(
      <CreateCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    await waitFor(() => {
      expect(screen.queryByTestId("spinner")).not.toBeInTheDocument();
    });

    const cancelButton = screen.getByText("Cancel");
    await user.click(cancelButton);

    expect(mockOnExit).toHaveBeenCalledTimes(1);
  });
});
