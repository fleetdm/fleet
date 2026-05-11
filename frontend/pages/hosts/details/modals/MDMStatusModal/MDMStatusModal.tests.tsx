import React from "react";

import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import { AxiosError } from "axios";

import hostAPI from "services/entities/hosts";
import paths from "router/paths";
import MDMStatusModal from "./MDMStatusModal";

jest.mock("services/entities/hosts");

const mockRouter = createMockRouter();

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
    context: {},
  });

  afterEach(() => {
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

    expect(screen.getByText(/On \(manual\)/i)).toBeInTheDocument();
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
        isAppleDevice
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
        isAppleDevice={false}
        onExit={jest.fn()}
      />
    );
    expect(screen.queryByText("Profile assignment")).not.toBeInTheDocument();
  });

  it("renders profile assignment section when premium apple device host", async () => {
    (hostAPI.getDepAssignment as jest.Mock).mockResolvedValue(
      mockDepAssignmentResponse
    );

    render(
      <MDMStatusModal
        hostId={3}
        enrollmentStatus="On (manual)"
        router={mockRouter}
        isPremiumTier
        isAppleDevice
        onExit={jest.fn()}
      />
    );

    expect(await screen.findByText("Profile assignment")).toBeInTheDocument();
    expect(
      screen.getByText(/Details about automatic enrollment profile from Apple/i)
    ).toBeInTheDocument();
    expect(screen.getByText("Profile assigned")).toBeInTheDocument();
    expect(screen.getByText("Profile pushed")).toBeInTheDocument();
    expect(screen.getByText("Profile status")).toBeInTheDocument();
    // profile_status "assigned" renders "Assigned"
    expect(screen.getByText("Assigned")).toBeInTheDocument();
  });

  it("does not render profile assignment section when non-Apple host", async () => {
    (hostAPI.getDepAssignment as jest.Mock).mockResolvedValue(
      mockDepAssignmentResponse
    );

    render(
      <MDMStatusModal
        hostId={3}
        enrollmentStatus="On (manual)"
        router={mockRouter}
        isPremiumTier
        onExit={jest.fn()}
      />
    );

    expect(
      await screen.queryByText("Profile assignment")
    ).not.toBeInTheDocument();
    expect(
      screen.queryByText(
        /Details about automatic enrollment profile from Apple/i
      )
    ).not.toBeInTheDocument();
    expect(screen.queryByText("Profile assigned")).not.toBeInTheDocument();
    expect(screen.queryByText("Profile pushed")).not.toBeInTheDocument();
    expect(screen.queryByText("Profile status")).not.toBeInTheDocument();
    expect(screen.queryByText("Assigned")).not.toBeInTheDocument();
  });

  it("shows spinner while DEP assignment is loading", async () => {
    (hostAPI.getDepAssignment as jest.Mock).mockReturnValue(
      new Promise(() => {
        // never resolve
      })
    );

    render(
      <MDMStatusModal
        hostId={3}
        enrollmentStatus="On (manual)"
        router={mockRouter}
        isPremiumTier
        isAppleDevice
        onExit={jest.fn()}
      />
    );

    // Spinner has a built-in anti-flash delay, so wait for it to appear.
    expect(await screen.findByTestId("spinner")).toBeVisible();
  });

  it("shows DataError if DEP assignment fails", async () => {
    (hostAPI.getDepAssignment as jest.Mock).mockRejectedValue(
      new AxiosError("network error")
    );

    render(
      <MDMStatusModal
        hostId={3}
        enrollmentStatus="On (manual)"
        router={mockRouter}
        isPremiumTier
        isAppleDevice
        onExit={jest.fn()}
      />
    );

    expect(
      await screen.findByText(
        "We can't retrieve data from Apple right now. Please try again later."
      )
    ).toBeInTheDocument();
  });

  it("adds profile assignment error row when depProfileError is true and API returns THROTTLED", async () => {
    (hostAPI.getDepAssignment as jest.Mock).mockResolvedValue({
      ...mockDepAssignmentResponse,
      host_dep_assignment: {
        ...mockDepAssignmentResponse.host_dep_assignment,
        assign_profile_response: "THROTTLED",
      },
    });

    render(
      <MDMStatusModal
        hostId={3}
        enrollmentStatus="On (manual)"
        router={mockRouter}
        isPremiumTier
        isAppleDevice
        depProfileError
        onExit={jest.fn()}
      />
    );

    // Ensure the async list has loaded
    await screen.findByText("Profile assigned");

    expect(
      await screen.findByText("Profile assignment error")
    ).toBeInTheDocument();
    expect(screen.getByText("Throttled")).toBeInTheDocument();
  });

  it("navigates to hosts with dep_assign_profile_response filter when profile error row is clicked", async () => {
    (hostAPI.getDepAssignment as jest.Mock).mockResolvedValue({
      ...mockDepAssignmentResponse,
      host_dep_assignment: {
        ...mockDepAssignmentResponse.host_dep_assignment,
        assign_profile_response: "FAILED",
      },
    });
    const router = createMockRouter();

    const { user } = render(
      <MDMStatusModal
        hostId={3}
        enrollmentStatus="On (manual)"
        router={router}
        isPremiumTier
        isAppleDevice
        depProfileError
        onExit={jest.fn()}
      />
    );

    // Wait for list to be hydrated
    await screen.findByText("Profile assigned");

    const profileErrorRow = await screen.findByText("Profile assignment error");
    await user.click(profileErrorRow);

    await waitFor(() => {
      expect(router.push).toHaveBeenCalled();
      const firstCall = (router.push as jest.Mock).mock.calls[0][0];
      expect(firstCall).toContain(paths.MANAGE_HOSTS);
      // Router navigation still uses uppercase responseParam
      expect(firstCall).toContain("dep_assign_profile_response=FAILED");
    });
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

    const mdmStatus = screen.getByText(/On \(manual\)/i);
    await user.click(mdmStatus);

    await waitFor(() => {
      expect(router.push).toHaveBeenCalled();
      const firstCall = (router.push as jest.Mock).mock.calls[0][0];
      expect(firstCall).toContain(paths.MANAGE_HOSTS);
      expect(firstCall).toContain("mdm_enrollment_status=");
    });
  });

  it("renders 'Never' for zero-value profile timestamps", async () => {
    (hostAPI.getDepAssignment as jest.Mock).mockResolvedValue({
      ...mockDepAssignmentResponse,
      dep_device: {
        ...mockDepAssignmentResponse.dep_device,
        profile_assign_time: "0001-01-01T00:00:00Z",
        profile_push_time: "0001-01-01T00:00:00Z",
      },
    });

    render(
      <MDMStatusModal
        hostId={3}
        enrollmentStatus="On (manual)"
        router={mockRouter}
        isPremiumTier
        isAppleDevice
        onExit={jest.fn()}
      />
    );

    await screen.findByText("Profile assigned");

    const neverTexts = screen.getAllByText("Never");
    // Both profile_assign_time and profile_push_time should show "Never"
    expect(neverTexts.length).toBeGreaterThanOrEqual(2);
  });

  it("calls onExit when Close is clicked", async () => {
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
        isAppleDevice
        onExit={onExit}
      />
    );

    await user.click(screen.getByRole("button", { name: "Close" }));
    expect(onExit).toHaveBeenCalled();
  });
});
