import React from "react";

import { screen, waitFor } from "@testing-library/react";
import { ICertificate } from "services/entities/certificates";
import { renderWithSetup } from "test/test-utils";

import AddCertModal from "./AddCertificateModal";

jest.mock("services/entities/certificates");

const mockOnExit = jest.fn();
const mockOnSuccess = jest.fn();

const mockCertAuthorities = [
  { id: 1, name: "Test CA 1", type: "custom_scep_proxy" as const },
  { id: 2, name: "Test CA 2", type: "custom_scep_proxy" as const },
];

const mockExistingCTs: ICertificate[] = [
  {
    id: 1,
    name: "Existing Certificate",
    certificate_authority_id: 1,
    certificate_authority_name: "Test CA 1",
    created_at: "2024-01-01T00:00:00Z",
  },
];

// Mock the useQuery hook
jest.mock("react-query", () => ({
  ...jest.requireActual("react-query"),
  useQuery: jest.fn(() => ({
    data: { certificate_authorities: mockCertAuthorities },
    isLoading: false,
    isError: false,
  })),
}));

// Mock the contexts
jest.mock("context/notification", () => ({
  NotificationContext: {
    Consumer: ({ children }: any) => children({ renderFlash: jest.fn() }),
  },
}));

jest.mock("context/app", () => ({
  AppContext: {
    Consumer: ({ children }: any) => children({ currentTeam: { id: 1 } }),
  },
}));

describe("AddCertModal", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("renders the modal with all form fields", async () => {
    renderWithSetup(
      <AddCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    expect(screen.getByText("Add certificate")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("VPN certificate")).toBeInTheDocument();
    expect(screen.getByText("Certificate authority (CA)")).toBeInTheDocument();
    expect(
      screen.getByPlaceholderText(
        "/CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME/O=Your Organization"
      )
    ).toBeInTheDocument();
    expect(screen.getByText("Create")).toBeInTheDocument();
    expect(screen.getByText("Cancel")).toBeInTheDocument();
  });

  it("disables Create button when Name field is empty", async () => {
    const { user } = renderWithSetup(
      <AddCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    const createButton = screen.getByText("Create");
    expect(createButton).toBeDisabled();

    await user.hover(createButton);
    await waitFor(() => {
      expect(
        screen.getByText("Complete all required fields to save")
      ).toBeInTheDocument();
    });
  });

  it("shows error for Name with invalid characters and disables Create button", async () => {
    const { user } = renderWithSetup(
      <AddCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    const nameInput = screen.getByPlaceholderText("VPN certificate");
    await user.type(nameInput, "Invalid@Name#");

    await waitFor(() => {
      expect(
        screen.getByText(
          "Invalid characters. Only letters, numbers, spaces, dashes, and underscores allowed."
        )
      ).toBeInTheDocument();
    });

    const createButton = screen.getByText("Create");
    expect(createButton).toBeDisabled();
  });

  it("shows error for Name that already exists and disables Create button", async () => {
    const { user } = renderWithSetup(
      <AddCertModal
        existingCerts={mockExistingCTs}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    const nameInput = screen.getByPlaceholderText("VPN certificate");
    await user.type(nameInput, "Existing Certificate");

    await waitFor(() => {
      expect(
        screen.getByText("Name is already used by another certificate.")
      ).toBeInTheDocument();
    });

    const createButton = screen.getByText("Create");
    expect(createButton).toBeDisabled();
  });

  it("shows error for Name with more than 255 characters and disables Create button", async () => {
    const { user } = renderWithSetup(
      <AddCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    const nameInput = screen.getByPlaceholderText("VPN certificate");
    const longName = "a".repeat(256);
    await user.type(nameInput, longName);

    await waitFor(() => {
      expect(
        screen.getByText("Name is too long. Maximum is 255 characters.")
      ).toBeInTheDocument();
    });

    const createButton = screen.getByText("Create");
    expect(createButton).toBeDisabled();
  });

  it("disables Create button when Certificate authority is not selected", async () => {
    const { user } = renderWithSetup(
      <AddCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    // Fill in name and subject name, but leave CA unselected
    const nameInput = screen.getByPlaceholderText("VPN certificate");
    await user.type(nameInput, "Valid Name");

    const subjectNameInput = screen.getByPlaceholderText(
      "/CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME/O=Your Organization"
    );
    await user.type(subjectNameInput, "/CN=test/O=Org");

    const createButton = screen.getByText("Create");
    expect(createButton).toBeDisabled();
  });

  it("disables Create button when Subject name is empty", async () => {
    const { user } = renderWithSetup(
      <AddCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    // Fill in name only
    const nameInput = screen.getByPlaceholderText("VPN certificate");
    await user.type(nameInput, "Valid Name");

    const createButton = screen.getByText("Create");
    expect(createButton).toBeDisabled();
  });

  it("enables Create button when all fields are valid", async () => {
    const { user } = renderWithSetup(
      <AddCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    // Fill in all fields with valid data
    const nameInput = screen.getByPlaceholderText("VPN certificate");
    await user.type(nameInput, "Valid Name");

    const subjectNameInput = screen.getByPlaceholderText(
      "/CN=$FLEET_VAR_HOST_END_USER_IDP_USERNAME/O=Your Organization"
    );
    await user.type(subjectNameInput, "/CN=test/O=Org");

    // Select a certificate authority from dropdown
    const caDropdown = screen.getByText("Select certificate authority");
    await user.click(caDropdown);

    await waitFor(() => {
      expect(screen.getByText("Test CA 1")).toBeInTheDocument();
    });

    await user.click(screen.getByText("Test CA 1"));

    // Wait for the Create button to be enabled
    await waitFor(() => {
      const createButton = screen.getByText("Create");
      expect(createButton).not.toBeDisabled();
    });
  });

  it("calls onExit when Cancel button is clicked", async () => {
    const { user } = renderWithSetup(
      <AddCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    const cancelButton = screen.getByText("Cancel");
    await user.click(cancelButton);

    expect(mockOnExit).toHaveBeenCalledTimes(1);
  });

  it("accepts valid name with letters, numbers, spaces, dashes, and underscores", async () => {
    const { user } = renderWithSetup(
      <AddCertModal
        existingCerts={[]}
        onExit={mockOnExit}
        onSuccess={mockOnSuccess}
      />
    );

    const nameInput = screen.getByPlaceholderText("VPN certificate");
    await user.type(nameInput, "Valid Name-123_test");

    await waitFor(() => {
      // Should not show error message
      expect(
        screen.queryByText(
          "Invalid characters. Only letters, numbers, spaces, dashes, and underscores allowed."
        )
      ).not.toBeInTheDocument();
    });
  });
});
