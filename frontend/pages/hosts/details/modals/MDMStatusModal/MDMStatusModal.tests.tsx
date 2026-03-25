// MDMStatusModal.test.tsx
import React from "react";

import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import { http, HttpResponse } from "msw";
import mockServer from "test/mock-server";

import hostAPI from "services/entities/hosts";
import paths from "router/paths";
import MDMStatusModal from "./MDMStatusModal";

jest.mock("services/entities/hosts");

const mockRouter = createMockRouter();

const baseUrl = (path: string) => `/api/latest/fleet${path}`;

// You only need these if the component actually calls the API,
// but you mentioned we want to keep the fakeDepAssignmentData.
// If/when you hook this up to the backend, you can switch to msw handlers.
const mockDepAssignmentResponse = {
  id: 32,
  dep_device: {
    asset_tag: "",
    color: "MIDNIGHT",
    description: "IPHONE 13 MIDNIGHT 128GB-USA",
    device_assigned_by: "fleetie@example.com",
    device_assigned_date: "2026-01-29T21:17:25Z",
    device_family: "iPhone",
    os: "iOS",
    profile_status: "assigned",
    profile_assign_time: "2026-01-29T21:17:25Z",
    profile_push_time: "2026-01-03T00:00:00Z",
    profile_uuid: "762C4D36550103CCC53AA212A8D31CDD",
    mdm_migration_deadline: null,
    serial_number: "ABC1FND0ZX",
  },
  host_dep_assignment: {
    assign_profile_response: "SUCCESS",
    profile_uuid: "762C4D36550103CCC53AA212A8D31CDD",
    response_updated_at: "2025-12-04 01:35:27",
    added_at: "2025-12-04 01:35:27",
    deleted_at: null,
    abm_token_id: 1,
    mdm_migration_deadline: "2025-12-05 00:00:00.000000",
    mdm_migration_completed: "2025-12-05 00:00:00.000000",
  },
};

describe("MDMStatusModal - component", () => {
  const render = createCustomRenderer({
    withBackendMock: true,
    // No special app/query context needed as of now.
    context: {},
  });

  beforeEach(() => {
    jest.resetAllMocks();
  });

  it("renders MDM status row with enrollment status text", () => {
    (hostAPI.getDepAssignment as jest.Mock).mockResolvedValue(
      mockDepAssignmentResponse
    );

    render(
      <MDMStatusModal
        hostId={3}
        enrollmentStatus="On (manual)"
        router={mockRouter}
        onExit={jest.fn()}
      />
    );

    expect(screen.getByText("MDM status")).toBeInTheDocument();
    // Adjust expected text if your UI map uses a different label
    expect(screen.getByText("On (manual)")).toBeInTheDocument();
  });

  it("does not render profile assignment section when not premium or not macOS", () => {
    (hostAPI.getDepAssignment as jest.Mock).mockResolvedValue(
      mockDepAssignmentResponse
    );

    // not premium
    render(
      <MDMStatusModal
        hostId={3}
        enrollmentStatus="On (manual)"
        router={mockRouter}
        isPremiumTier={false}
        isMacOSHost
        onExit={jest.fn()}
      />
    );
    expect(screen.queryByText("Profile assignment")).not.toBeInTheDocument();

    // not macOS
    render(
      <MDMStatusModal
        hostId={3}
        enrollmentStatus="On (manual)"
        router={mockRouter}
        isPremiumTier
        isMacOSHost={false}
        onExit={jest.fn()}
      />
    );
    expect(screen.queryByText("Profile assignment")).not.toBeInTheDocument();
  });

  it("renders profile assignment section when premium macOS host", () => {
    (hostAPI.getDepAssignment as jest.Mock).mockResolvedValue(
      mockDepAssignmentResponse
    );

    render(
      <MDMStatusModal
        hostId={3}
        enrollmentStatus="On (manual)"
        router={mockRouter}
        isPremiumTier
        isMacOSHost
        onExit={jest.fn()}
      />
    );

    expect(screen.getByText("Profile assignment")).toBeInTheDocument();
    expect(
      screen.getByText(/Details about automatic enrollment profile from Apple/i)
    ).toBeInTheDocument();
    expect(screen.getByText("Profile assigned")).toBeInTheDocument();
    expect(screen.getByText("Profile pushed")).toBeInTheDocument();
    expect(screen.getByText("Profile status")).toBeInTheDocument();
    expect(screen.getByText("Assigned")).toBeInTheDocument();
  });

  it("shows spinner while DEP assignment is loading", async () => {
    (hostAPI.getDepAssignment as jest.Mock).mockReturnValue(
      new Promise(() => {
        // never resolve: keeps loading
      })
    );

    render(
      <MDMStatusModal
        hostId={3}
        enrollmentStatus="On (manual)"
        router={mockRouter}
        isPremiumTier
        isMacOSHost
        onExit={jest.fn()}
      />
    );

    // Fleet <Spinner /> typically renders role="status"
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("shows DataError if DEP assignment fails", async () => {
    (hostAPI.getDepAssignment as jest.Mock).mockRejectedValue(
      new Error("network error")
    );

    render(
      <MDMStatusModal
        hostId={3}
        enrollmentStatus="On (manual)"
        router={mockRouter}
        isPremiumTier
        isMacOSHost
        onExit={jest.fn()}
      />
    );

    expect(
      await screen.findByText(
        "We can't retrieve data from Apple right now. Please try again later."
      )
    ).toBeInTheDocument();
  });

  it("adds profile assignment error row when depProfileError is true and API returns throttled", async () => {
    (hostAPI.getDepAssignment as jest.Mock).mockResolvedValue({
      ...mockDepAssignmentResponse,
      host_dep_assignment: {
        ...mockDepAssignmentResponse.host_dep_assignment,
        assign_profile_response: "throttled",
      },
    });

    render(
      <MDMStatusModal
        hostId={3}
        enrollmentStatus="On (manual)"
        router={mockRouter}
        isPremiumTier
        isMacOSHost
        depProfileError
        onExit={jest.fn()}
      />
    );

    expect(
      await screen.findByText("Profile assignment error")
    ).toBeInTheDocument();
    expect(screen.getByText("Throttled")).toBeInTheDocument();
  });

  it("navigates to filtered hosts when MDM status row is clicked", async () => {
    (hostAPI.getDepAssignment as jest.Mock).mockResolvedValue(
      mockDepAssignmentResponse
    );
    const router = createMockRouter();

    const { user } = render(
      <MDMStatusModal
        hostId={3}
        enrollmentStatus="On (manual)"
        router={router}
        onExit={jest.fn()}
      />
    );

    const mdmStatus = screen.getByText("MDM status");
    await user.click(mdmStatus);

    await waitFor(() => {
      expect(router.push).toHaveBeenCalled();
      const firstCall = (router.push as jest.Mock).mock.calls[0][0];
      expect(firstCall).toContain(paths.MANAGE_HOSTS);
      expect(firstCall).toContain("mdm_enrollment_status");
    });
  });

  it("navigates to hosts with dep_profile_error when profile row is clicked", async () => {
    (hostAPI.getDepAssignment as jest.Mock).mockResolvedValue({
      ...mockDepAssignmentResponse,
      host_dep_assignment: {
        ...mockDepAssignmentResponse.host_dep_assignment,
        assign_profile_response: "failed",
      },
    });
    const router = createMockRouter();

    const { user } = render(
      <MDMStatusModal
        hostId={3}
        enrollmentStatus="On (manual)"
        router={router}
        isPremiumTier
        isMacOSHost
        depProfileError
        onExit={jest.fn()}
      />
    );

    const profileErrorRow = await screen.findByText("Profile assignment error");
    await user.click(profileErrorRow);

    await waitFor(() => {
      expect(router.push).toHaveBeenCalled();
      const firstCall = (router.push as jest.Mock).mock.calls[0][0];
      expect(firstCall).toContain(paths.MANAGE_HOSTS);
      expect(firstCall).toContain("dep_profile_error=true");
    });
  });

  it("calls onExit when Done is clicked", async () => {
    (hostAPI.getDepAssignment as jest.Mock).mockResolvedValue(
      mockDepAssignmentResponse
    );
    const onExit = jest.fn();

    const { user } = render(
      <MDMStatusModal
        hostId={3}
        enrollmentStatus="On (manual)"
        router={mockRouter}
        isPremiumTier
        isMacOSHost
        onExit={onExit}
      />
    );

    await user.click(screen.getByRole("button", { name: "Done" }));
    expect(onExit).toHaveBeenCalled();
  });
});
