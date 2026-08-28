import React from "react";

import { screen, waitFor } from "@testing-library/react";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import { AxiosError } from "axios";

import createMockUser from "__mocks__/userMock";
import hostAPI from "services/entities/hosts";
import paths from "router/paths";
import { internationalTimeFormat } from "utilities/helpers";
import { notify } from "components/ToastNotification";
import MDMStatusModal from "./MDMStatusModal";

jest.mock("services/entities/hosts");
jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    batch: jest.fn(),
    dismiss: jest.fn(),
  },
}));

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
  // Rendered as a global admin throughout. The check-in action is expected to
  // become permission-gated, and these cases should still pass once it is.
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isGlobalAdmin: true,
        currentUser: createMockUser(),
      },
    },
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
        platform="windows"
        router={mockRouter}
        user={createMockUser()}
        lastMDMCheckIn=""
        onSuccessfulCheckIn={jest.fn()}
        fleetId={null}
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
        platform={"darwin"}
        user={createMockUser()}
        lastMDMCheckIn=""
        onSuccessfulCheckIn={jest.fn()}
        fleetId={null}
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
        platform="windows"
        user={createMockUser()}
        lastMDMCheckIn=""
        onSuccessfulCheckIn={jest.fn()}
        fleetId={null}
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
        platform={"darwin"}
        user={createMockUser()}
        lastMDMCheckIn=""
        onSuccessfulCheckIn={jest.fn()}
        fleetId={null}
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
        platform="windows"
        enrollmentStatus="On (manual)"
        router={mockRouter}
        isPremiumTier
        user={createMockUser()}
        lastMDMCheckIn=""
        onSuccessfulCheckIn={jest.fn()}
        fleetId={null}
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

  it("does not render profile assignment section for a non-DEP host (host_dep_assignment is null)", async () => {
    (hostAPI.getDepAssignment as jest.Mock).mockResolvedValue({
      id: 3,
      dep_device: null,
      dep_device_error: null,
      host_dep_assignment: null,
    });

    render(
      <MDMStatusModal
        hostId={3}
        enrollmentStatus="On (manual)"
        router={mockRouter}
        isPremiumTier
        platform={"darwin"}
        user={createMockUser()}
        lastMDMCheckIn=""
        onSuccessfulCheckIn={jest.fn()}
        fleetId={null}
        onExit={jest.fn()}
      />
    );

    // Wait for the section to disappear once the query settles (it's shown
    // while loading, since a DEP host wouldn't be distinguishable from a
    // non-DEP host until the response comes back -- the section then hides
    // itself once host_dep_assignment resolves to null). Waiting on this
    // directly, rather than on the spinner's absence, since the spinner's
    // anti-flash delay means it may never render at all for a fast-resolving
    // mock, making its absence a false signal that the query has settled.
    await waitFor(() => {
      expect(screen.queryByText("Profile assignment")).not.toBeInTheDocument();
    });
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
        platform={"darwin"}
        user={createMockUser()}
        lastMDMCheckIn=""
        onSuccessfulCheckIn={jest.fn()}
        fleetId={null}
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
        platform={"darwin"}
        user={createMockUser()}
        lastMDMCheckIn=""
        onSuccessfulCheckIn={jest.fn()}
        fleetId={null}
        onExit={jest.fn()}
      />
    );

    expect(
      await screen.findByText(
        "Fleet can't retrieve data from Apple right now. Please try again later."
      )
    ).toBeInTheDocument();
  });

  it("shows the dep_device_error message from the API when Apple returns no dep_device", async () => {
    (hostAPI.getDepAssignment as jest.Mock).mockResolvedValue({
      ...mockDepAssignmentResponse,
      dep_device: null,
      dep_device_error:
        "Fleet can't connect to Apple Business. An admin needs to renew the AB token.",
    });

    render(
      <MDMStatusModal
        hostId={3}
        enrollmentStatus="On (manual)"
        router={mockRouter}
        isPremiumTier
        platform={"darwin"}
        user={createMockUser()}
        lastMDMCheckIn=""
        onSuccessfulCheckIn={jest.fn()}
        fleetId={null}
        onExit={jest.fn()}
      />
    );

    expect(
      await screen.findByText(
        "Fleet can't connect to Apple Business. An admin needs to renew the AB token."
      )
    ).toBeInTheDocument();
  });

  it("falls back to the generic message when dep_device is missing and dep_device_error is unset", async () => {
    (hostAPI.getDepAssignment as jest.Mock).mockResolvedValue({
      ...mockDepAssignmentResponse,
      dep_device: null,
      dep_device_error: null,
    });

    render(
      <MDMStatusModal
        hostId={3}
        enrollmentStatus="On (manual)"
        router={mockRouter}
        isPremiumTier
        platform={"darwin"}
        user={createMockUser()}
        lastMDMCheckIn=""
        onSuccessfulCheckIn={jest.fn()}
        fleetId={null}
        onExit={jest.fn()}
      />
    );

    expect(
      await screen.findByText(
        "Fleet can't retrieve data from Apple right now. Please try again later."
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
        platform={"darwin"}
        depProfileError
        user={createMockUser()}
        lastMDMCheckIn=""
        onSuccessfulCheckIn={jest.fn()}
        fleetId={null}
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
        platform={"darwin"}
        depProfileError
        user={createMockUser()}
        lastMDMCheckIn=""
        onSuccessfulCheckIn={jest.fn()}
        fleetId={null}
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
        platform="windows"
        enrollmentStatus="On (manual)"
        router={router}
        user={createMockUser()}
        lastMDMCheckIn=""
        onSuccessfulCheckIn={jest.fn()}
        fleetId={null}
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
        platform={"darwin"}
        user={createMockUser()}
        lastMDMCheckIn=""
        onSuccessfulCheckIn={jest.fn()}
        fleetId={null}
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
        platform={"darwin"}
        user={createMockUser()}
        lastMDMCheckIn=""
        onSuccessfulCheckIn={jest.fn()}
        fleetId={null}
        onExit={onExit}
      />
    );

    await user.click(screen.getByRole("button", { name: "Close" }));
    expect(onExit).toHaveBeenCalled();
  });
});

describe("MDMStatusModal - MDM check-in", () => {
  const LAST_CHECK_IN = "2026-02-10T18:30:00Z";

  // The check-in action is gated at observer-or-higher, and the modal reads
  // the role off its `user` prop rather than app context -- so cases that
  // exercise the gate have to set both.
  const MAINTAINER = createMockUser({
    role: "maintainer",
    global_role: "maintainer",
  });
  const OBSERVER = createMockUser({
    role: "observer",
    global_role: "observer",
  });

  const renderAsAdmin = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isGlobalAdmin: true,
        currentUser: createMockUser(),
      },
    },
  });

  const renderAsMaintainer = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isGlobalMaintainer: true,
        currentUser: MAINTAINER,
      },
    },
  });

  const renderModal = (
    renderer: typeof renderAsAdmin,
    props: Partial<React.ComponentProps<typeof MDMStatusModal>> = {}
  ) =>
    renderer(
      <MDMStatusModal
        hostId={3}
        enrollmentStatus="On (manual)"
        router={mockRouter}
        isPremiumTier
        platform={"darwin"}
        user={createMockUser()}
        lastMDMCheckIn={LAST_CHECK_IN}
        onSuccessfulCheckIn={jest.fn()}
        onExit={jest.fn()}
        fleetId={props.fleetId ?? null}
        connectedToFleet
        {...props}
      />
    );

  beforeEach(() => {
    (hostAPI.getDepAssignment as jest.Mock).mockResolvedValue(
      mockDepAssignmentResponse
    );
  });

  afterEach(() => {
    jest.resetAllMocks();
  });

  it("renders the last MDM check-in time for an Apple host", async () => {
    renderModal(renderAsAdmin);

    expect(await screen.findByText("Last MDM check-in")).toBeInTheDocument();
    expect(
      screen.getByText(internationalTimeFormat(new Date(LAST_CHECK_IN)))
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /check in now/i })).toBeEnabled();
  });

  it("renders 'Never' when the host has not checked in yet", async () => {
    renderModal(renderAsAdmin, { lastMDMCheckIn: "" });

    expect(await screen.findByText("Last MDM check-in")).toBeInTheDocument();
    expect(screen.getByText("Never")).toBeInTheDocument();
  });

  it("does not render the check-in button for a non-Apple host", async () => {
    renderModal(renderAsAdmin, { platform: "windows" });

    // "MDM status" is also the modal title, so settle on the status row's
    // value instead to know the list rendered.
    expect(await screen.findByText(/On \(manual\)/i)).toBeInTheDocument();
    expect(screen.queryByText("Last MDM check-in")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /check in now/i })
    ).not.toBeInTheDocument();
  });

  it("pings the host and refreshes host details when 'Check in now' is clicked", async () => {
    (hostAPI.apnsPing as jest.Mock).mockResolvedValue({});
    const onSuccessfulCheckIn = jest.fn();

    const { user } = renderModal(renderAsAdmin, { onSuccessfulCheckIn });

    await user.click(
      await screen.findByRole("button", { name: /check in now/i })
    );

    await waitFor(() => {
      expect(onSuccessfulCheckIn).toHaveBeenCalled();
    });
    expect(hostAPI.apnsPing).toHaveBeenCalledWith(3);
    expect(notify.error).not.toHaveBeenCalled();
  });

  it("pings the host when a global maintainer clicks 'Check in now'", async () => {
    (hostAPI.apnsPing as jest.Mock).mockResolvedValue({});
    const onSuccessfulCheckIn = jest.fn();

    const { user } = renderModal(renderAsMaintainer, {
      user: MAINTAINER,
      onSuccessfulCheckIn,
    });

    await user.click(
      await screen.findByRole("button", { name: /check in now/i })
    );

    await waitFor(() => {
      expect(onSuccessfulCheckIn).toHaveBeenCalled();
    });
    expect(hostAPI.apnsPing).toHaveBeenCalledWith(3);
  });

  it("surfaces an error and does not refresh host details when the ping fails", async () => {
    (hostAPI.apnsPing as jest.Mock).mockRejectedValue(
      new AxiosError("network error")
    );
    const onSuccessfulCheckIn = jest.fn();

    const { user } = renderModal(renderAsAdmin, { onSuccessfulCheckIn });

    await user.click(
      await screen.findByRole("button", { name: /check in now/i })
    );

    await waitFor(() => {
      expect(notify.error).toHaveBeenCalled();
    });
    expect(onSuccessfulCheckIn).not.toHaveBeenCalled();
    // The button has to come back so the admin can retry.
    await waitFor(() => {
      expect(
        screen.getByRole("button", { name: /check in now/i })
      ).toBeEnabled();
    });
  });

  it("disables 'Check in now' while the ping is in flight", async () => {
    (hostAPI.apnsPing as jest.Mock).mockReturnValue(
      new Promise(() => {
        // never resolve
      })
    );

    const { user } = renderModal(renderAsAdmin);

    const checkInButton = await screen.findByRole("button", {
      name: /check in now/i,
    });
    await user.click(checkInButton);

    await waitFor(() => {
      expect(checkInButton).toBeDisabled();
    });
  });

  it("does not navigate to the hosts list when 'Check in now' is clicked", async () => {
    (hostAPI.apnsPing as jest.Mock).mockResolvedValue({});
    const router = createMockRouter();

    const { user } = renderModal(renderAsAdmin, { router });

    await user.click(
      await screen.findByRole("button", { name: /check in now/i })
    );

    await waitFor(() => {
      expect(hostAPI.apnsPing).toHaveBeenCalled();
    });
    expect(router.push).not.toHaveBeenCalled();
  });

  it("offers 'Check in now' to observers and above", async () => {
    const CHECK_IN_BUTTON = { name: /check in now/i };

    // observers and above can all ping
    const { unmount } = renderModal(renderAsAdmin);
    expect(
      await screen.findByRole("button", CHECK_IN_BUTTON)
    ).toBeInTheDocument();
    unmount();

    const { unmount: unmountMaintainer } = renderModal(renderAsMaintainer, {
      user: MAINTAINER,
    });
    expect(
      await screen.findByRole("button", CHECK_IN_BUTTON)
    ).toBeInTheDocument();
    unmountMaintainer();

    renderModal(renderAsAdmin, { user: OBSERVER });
    expect(await screen.findByText("Last MDM check-in")).toBeInTheDocument();
    expect(
      await screen.findByRole("button", CHECK_IN_BUTTON)
    ).toBeInTheDocument();
  });

  it("still navigates to filtered hosts when the MDM status row is clicked", async () => {
    const router = createMockRouter();

    const { user } = renderModal(renderAsAdmin, { router });

    await user.click(await screen.findByText(/On \(manual\)/i));

    await waitFor(() => {
      expect(router.push).toHaveBeenCalled();
    });
    expect((router.push as jest.Mock).mock.calls[0][0]).toContain(
      "mdm_enrollment_status="
    );
    expect(hostAPI.apnsPing).not.toHaveBeenCalled();
  });
});
