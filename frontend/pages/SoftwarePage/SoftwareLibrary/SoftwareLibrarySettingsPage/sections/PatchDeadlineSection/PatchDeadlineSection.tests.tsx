import { screen, waitFor, within } from "@testing-library/react";
import React, { useContext } from "react";

import createMockConfig from "__mocks__/configMock";
import createMockUser from "__mocks__/userMock";
import { notify } from "components/ToastNotification";
import { AppContext } from "context/app";
import useGitOpsMode from "hooks/useGitOpsMode";
import {
  emptyPatchDeadlineAppsHandler,
  getPatchDeadlineAppsErrorHandler,
  getPatchDeadlineAppsHandler,
  getPatchDeadlineSettingsHandler,
  getPatchDeadlineSettingsNotFoundHandler,
  getPatchDeadlineSettingsServerErrorHandler,
  updatePatchDeadlineSettingsFieldErrorHandler,
  updatePatchDeadlineSettingsHandler,
  updatePatchDeadlineSettingsPendingHandler,
  updatePatchDeadlineSettingsServerErrorHandler,
} from "test/handlers/patch-deadline-settings-handlers";
import mockServer from "test/mock-server";
import { createCustomRenderer, createMockRouter } from "test/test-utils";

import PatchDeadlineSection from "./PatchDeadlineSection";

jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    batch: jest.fn(),
    dismiss: jest.fn(),
  },
}));

interface IWrapperProps {
  fleetId?: number;
}

// Mirror what SoftwareLibrarySettingsPage passes to the section: canManage
// from the user's role, gitOpsModeEnabled from config. The section still
// receives location/router from the common ILibrarySettingsCardProps shape
// even though it doesn't read them, which keeps parity with the sibling
// categories section tests.
const SectionWrapper = ({ fleetId = 1 }: IWrapperProps) => {
  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamAdmin,
    isTeamMaintainer,
  } = useContext(AppContext);
  const { gitOpsModeEnabled } = useGitOpsMode("software");
  const canManage =
    !!isGlobalAdmin ||
    !!isGlobalMaintainer ||
    !!isTeamAdmin ||
    !!isTeamMaintainer;
  return (
    <PatchDeadlineSection
      router={createMockRouter()}
      location={{
        pathname: "/software/library/settings/patch-deadline",
        search: "?fleet_id=1",
        query: { fleet_id: "1" },
        hash: "",
      }}
      fleetId={fleetId}
      isRouteOk
      canManage={canManage}
      gitOpsModeEnabled={gitOpsModeEnabled}
    />
  );
};

const premiumAdminContext = {
  app: {
    isPremiumTier: true,
    isGlobalAdmin: true,
    currentUser: createMockUser({ global_role: "admin" }),
    setCurrentTeam: jest.fn(),
  },
};

const premiumObserverContext = {
  app: {
    isPremiumTier: true,
    isGlobalAdmin: false,
    isGlobalMaintainer: false,
    isTeamAdmin: false,
    isTeamMaintainer: false,
    currentUser: createMockUser({ global_role: "observer" }),
    setCurrentTeam: jest.fn(),
  },
};

const premiumAdminGitOpsContext = {
  app: {
    ...premiumAdminContext.app,
    config: createMockConfig({
      gitops: {
        gitops_mode_enabled: true,
        repository_url: "https://github.com/example/fleet-gitops",
        exceptions: { labels: false, software: false, secrets: true },
      },
    }),
  },
};

describe("PatchDeadlineSection", () => {
  beforeEach(() => {
    jest.clearAllMocks();
    // Keep the apps query from erroring in every test. Specific tests can
    // override by registering their own apps handler.
    mockServer.use(emptyPatchDeadlineAppsHandler);
  });

  it("populates the form from the server defaults", async () => {
    mockServer.use(getPatchDeadlineSettingsHandler());
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    render(<SectionWrapper />);

    // The checkbox mounts unchecked and flips to checked when the server
    // response seeds the form via useEffect — wait for the checked state
    // itself, not just the element's presence.
    const tuesday = await screen.findByRole("checkbox", { name: "Tuesday" });
    await waitFor(() => expect(tuesday).toBeChecked());
    expect(screen.getByRole("checkbox", { name: "Monday" })).not.toBeChecked();
    expect(screen.getByLabelText("Time")).toHaveValue("17:00");
  });

  it("shows an inline error on Save when no days are selected", async () => {
    mockServer.use(getPatchDeadlineSettingsHandler());
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    const { user } = render(<SectionWrapper />);

    const tuesday = await screen.findByRole("checkbox", { name: "Tuesday" });
    await user.click(tuesday); // uncheck the only day

    await user.click(screen.getByRole("button", { name: /save/i }));

    expect(
      await screen.findByText("Choose at least one day")
    ).toBeInTheDocument();
  });

  it("shows the Days error immediately on toggle (no Save required) and clears it when a day is re-checked", async () => {
    mockServer.use(getPatchDeadlineSettingsHandler());
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    const { user } = render(<SectionWrapper />);

    const tuesday = await screen.findByRole("checkbox", { name: "Tuesday" });
    await waitFor(() => expect(tuesday).toBeChecked());
    await user.click(tuesday); // uncheck the only day, days = []

    expect(
      await screen.findByText("Choose at least one day")
    ).toBeInTheDocument();

    await user.click(screen.getByRole("checkbox", { name: "Friday" }));

    await waitFor(() =>
      expect(
        screen.queryByText("Choose at least one day")
      ).not.toBeInTheDocument()
    );
  });

  it("shows an inline error on blur when the time is malformed", async () => {
    mockServer.use(getPatchDeadlineSettingsHandler());
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    const { user } = render(<SectionWrapper />);

    const time = await screen.findByLabelText("Time");
    await user.clear(time);
    await user.type(time, "25:99");
    await user.tab();

    expect(await screen.findByText("Enter time as HH:MM")).toBeInTheDocument();
  });

  it("falls back to the API defaults when the GET endpoint 404s", async () => {
    // Not-yet-deployed endpoint should let the page render against the
    // documented defaults (Tuesday 17:00), not surface a DataError.
    mockServer.use(getPatchDeadlineSettingsNotFoundHandler);
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    render(<SectionWrapper />);

    const tuesday = await screen.findByRole("checkbox", { name: "Tuesday" });
    await waitFor(() => expect(tuesday).toBeChecked());
    expect(screen.getByLabelText("Time")).toHaveValue("17:00");
  });

  it("surfaces non-404 GET failures instead of silently seeding defaults", async () => {
    // A real 500 or network blip must NOT look like "configured as Tuesday
    // 17:00" — the user would hit Save and overwrite the actual schedule.
    mockServer.use(getPatchDeadlineSettingsServerErrorHandler);
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    render(<SectionWrapper />);

    expect(
      await screen.findByText(/something's gone wrong/i)
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("checkbox", { name: "Tuesday" })
    ).not.toBeInTheDocument();
  });

  it("submits natively when Enter is pressed in the Time field", async () => {
    mockServer.use(
      getPatchDeadlineSettingsHandler(),
      updatePatchDeadlineSettingsHandler
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    const { user } = render(<SectionWrapper />);

    const time = await screen.findByLabelText("Time");
    await waitFor(() => expect(time).toHaveValue("17:00"));
    time.focus();
    await user.keyboard("{Enter}");

    await waitFor(() => {
      expect(notify.success).toHaveBeenCalledWith(
        "Successfully updated patch deadline."
      );
    });
  });

  it("saves on valid submit and toasts success", async () => {
    mockServer.use(
      getPatchDeadlineSettingsHandler(),
      updatePatchDeadlineSettingsHandler
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    const { user } = render(<SectionWrapper />);

    const thursday = await screen.findByRole("checkbox", { name: "Thursday" });
    await user.click(thursday);

    await user.click(screen.getByRole("button", { name: /save/i }));

    await waitFor(() => {
      expect(notify.success).toHaveBeenCalledWith(
        "Successfully updated patch deadline."
      );
    });
  });

  it("toasts an error when the server rejects the save", async () => {
    mockServer.use(
      getPatchDeadlineSettingsHandler(),
      updatePatchDeadlineSettingsServerErrorHandler
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    const { user } = render(<SectionWrapper />);

    await screen.findByRole("checkbox", { name: "Tuesday" });
    await user.click(screen.getByRole("button", { name: /save/i }));

    await waitFor(() => {
      expect(notify.error).toHaveBeenCalledWith(
        "Couldn't update patch deadline.",
        {
          response: expect.anything(),
        }
      );
    });
  });

  it("hides Save and disables fields for non-managers", async () => {
    mockServer.use(getPatchDeadlineSettingsHandler());
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumObserverContext,
    });

    render(<SectionWrapper />);

    // Fleet's Checkbox renders a div with role=checkbox + aria-disabled, so
    // jest-dom's toBeDisabled() doesn't match it. Native inputs still do.
    const tuesday = await screen.findByRole("checkbox", { name: "Tuesday" });
    expect(tuesday).toHaveAttribute("aria-disabled", "true");
    expect(screen.getByLabelText("Time")).toBeDisabled();
    expect(
      screen.queryByRole("button", { name: /save/i })
    ).not.toBeInTheDocument();
  });

  it("disables Save and the day/time fields under GitOps mode", async () => {
    mockServer.use(getPatchDeadlineSettingsHandler());
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminGitOpsContext,
    });

    render(<SectionWrapper />);

    const tuesday = await screen.findByRole("checkbox", { name: "Tuesday" });
    expect(screen.getByRole("button", { name: /save/i })).toBeDisabled();
    // Fields disable too so a GitOps admin isn't able to edit values they
    // can never persist. Fleet's custom checkbox div surfaces its disabled
    // state via aria-disabled; the native time input via disabled.
    expect(tuesday).toHaveAttribute("aria-disabled", "true");
    expect(screen.getByLabelText("Time")).toBeDisabled();
  });

  it("surfaces field-scoped server errors inline next to the field", async () => {
    mockServer.use(
      getPatchDeadlineSettingsHandler(),
      updatePatchDeadlineSettingsFieldErrorHandler([
        { name: "time", reason: "Time must be before midnight." },
      ])
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    const { user } = render(<SectionWrapper />);

    await screen.findByRole("checkbox", { name: "Tuesday" });
    await user.click(screen.getByRole("button", { name: /save/i }));

    expect(
      await screen.findByText("Time must be before midnight.")
    ).toBeInTheDocument();
  });

  it("renders a loading spinner on Save while the PATCH is in flight", async () => {
    mockServer.use(
      getPatchDeadlineSettingsHandler(),
      updatePatchDeadlineSettingsPendingHandler
    );
    const render = createCustomRenderer({
      withBackendMock: true,
      context: premiumAdminContext,
    });

    const { user } = render(<SectionWrapper />);

    await screen.findByRole("checkbox", { name: "Tuesday" });
    const save = screen.getByRole("button", { name: /save/i });
    await user.click(save);

    // Button goes disabled + mounts a spinner while the PATCH hangs.
    expect(save).toBeDisabled();
    expect(within(save).getByTestId("spinner")).toBeInTheDocument();
  });

  describe("Apps table", () => {
    it("hides the Apps section and the 'listed below' pointer when the fleet has no deadline apps", async () => {
      mockServer.use(getPatchDeadlineSettingsHandler());
      // beforeEach already registers emptyPatchDeadlineAppsHandler.
      const render = createCustomRenderer({
        withBackendMock: true,
        context: premiumAdminContext,
      });

      render(<SectionWrapper />);

      // Wait for the form to settle so the apps query has had a chance to
      // resolve, then assert the Apps subsection and its description pointer
      // are both absent.
      await screen.findByRole("checkbox", { name: "Tuesday" });
      await waitFor(() => {
        expect(
          screen.queryByRole("heading", { level: 2, name: "Apps" })
        ).not.toBeInTheDocument();
      });
      expect(
        screen.queryByText(/Apps with this option are listed below/i)
      ).not.toBeInTheDocument();
    });

    it("sorts apps alphabetically and paginates at 10 per page", async () => {
      // 12 apps in reverse-alphabetical order to prove client-side sort.
      const apps = Array.from({ length: 12 }, (_, i) => ({
        id: i + 1,
        name: String.fromCharCode(76 - i).repeat(5), // "LLLLL" down to "AAAAA"
      }));
      mockServer.use(
        getPatchDeadlineSettingsHandler(),
        getPatchDeadlineAppsHandler(apps)
      );
      const render = createCustomRenderer({
        withBackendMock: true,
        context: premiumAdminContext,
      });

      const { user } = render(<SectionWrapper />);

      // Settle the settings query first so the component is past its loading
      // spinner before we assert on apps content.
      await screen.findByRole("checkbox", { name: "Tuesday" });

      // First page: AAAAA..JJJJJ (10 items). KKKKK + LLLLL are on page 2.
      expect(await screen.findByText("AAAAA")).toBeInTheDocument();
      expect(screen.getByText("JJJJJ")).toBeInTheDocument();
      expect(screen.queryByText("KKKKK")).not.toBeInTheDocument();

      // "Listed below" pointer only appears when there are apps.
      expect(
        screen.getByText(/Apps with this option are listed below/i)
      ).toBeInTheDocument();

      await user.click(screen.getByRole("button", { name: /next/i }));

      expect(await screen.findByText("KKKKK")).toBeInTheDocument();
      expect(screen.getByText("LLLLL")).toBeInTheDocument();
      expect(screen.queryByText("AAAAA")).not.toBeInTheDocument();
    });

    it("rewinds to page 0 when the fleet switches", async () => {
      // Two multi-page fleets: user navigates to page 2 of fleet 1, then the
      // fleet dropdown switches to fleet 2. Pagination must reset so the user
      // sees fleet 2's PAGE 1 (prefix), not page 2 (suffix).
      const fleet1Apps = Array.from({ length: 12 }, (_, i) => ({
        id: i + 1,
        name: `fleet1-${String.fromCharCode(65 + i)}`, // fleet1-A..fleet1-L
      }));
      const fleet2Apps = Array.from({ length: 12 }, (_, i) => ({
        id: 100 + i,
        name: `fleet2-${String.fromCharCode(65 + i)}`, // fleet2-A..fleet2-L
      }));
      mockServer.use(
        getPatchDeadlineSettingsHandler(),
        getPatchDeadlineAppsHandler(fleet1Apps)
      );
      const render = createCustomRenderer({
        withBackendMock: true,
        context: premiumAdminContext,
      });

      const { user, rerender } = render(<SectionWrapper fleetId={1} />);

      // Wait past the settings-query spinner before asserting on apps.
      await screen.findByRole("checkbox", { name: "Tuesday" });
      await screen.findByText("fleet1-A");
      await user.click(screen.getByRole("button", { name: /next/i }));
      // Confirm page 2 rendered before the fleet switch.
      await screen.findByText("fleet1-K");

      // Fleet switch: wrapper re-renders with a new fleetId, which should
      // trigger the page-reset effect and a fresh apps query.
      mockServer.use(getPatchDeadlineAppsHandler(fleet2Apps));
      rerender(<SectionWrapper fleetId={2} />);

      // Fleet 2 page 1: A-J visible, K-L not. The settings query may spin
      // during the fleet switch, so wait past the spinner before asserting.
      await waitFor(() =>
        expect(screen.getByText("fleet2-A")).toBeInTheDocument()
      );
      expect(screen.getByText("fleet2-J")).toBeInTheDocument();
      expect(screen.queryByText("fleet2-K")).not.toBeInTheDocument();
    });

    it("renders a direct icon_url when the policy carries one", async () => {
      // Direct (non-/api/) icon_url is a VPP-style URL and SoftwareIcon
      // renders it as an <img src=…>. Prove the icon_url path survives the
      // install_software -> SoftwareIcon handoff.
      mockServer.use(
        getPatchDeadlineSettingsHandler(),
        getPatchDeadlineAppsHandler([
          {
            software_title_id: 42,
            name: "Figma",
            icon_url: "https://cdn.example.test/figma.png",
          },
        ])
      );
      const render = createCustomRenderer({
        withBackendMock: true,
        context: premiumAdminContext,
      });

      const { container } = render(<SectionWrapper />);
      await screen.findByRole("heading", { level: 2, name: "Apps" });

      // SoftwareIcon renders a direct URL as <img src>; the row name link
      // confirms the icon_url survived the install_software -> SoftwareIcon
      // handoff. Query by src attribute (role="img" is unreliable in jsdom
      // because the <img>'s onError handler fires and swaps to a fallback).
      const img = container.querySelector(
        'img[src="https://cdn.example.test/figma.png"]'
      );
      expect(img).not.toBeNull();
    });

    it("shows a DataError in the Apps slot when the policies query fails", async () => {
      // Hard-fail distinguishes from "fleet has no deadline apps" (which hides
      // the whole subsection). Section header stays so the user knows what
      // broke; the "listed below" pointer stays hidden because we don't know.
      mockServer.use(
        getPatchDeadlineSettingsHandler(),
        getPatchDeadlineAppsErrorHandler
      );
      const render = createCustomRenderer({
        withBackendMock: true,
        context: premiumAdminContext,
      });

      render(<SectionWrapper />);
      await screen.findByRole("checkbox", { name: "Tuesday" });

      expect(
        await screen.findByRole("heading", { level: 2, name: "Apps" })
      ).toBeInTheDocument();
      // Fleet's DataError card copy.
      expect(screen.getByText(/something's gone wrong/i)).toBeInTheDocument();
      // Pointer sentence doesn't appear when we don't know whether apps exist.
      expect(
        screen.queryByText(/Apps with this option are listed below/i)
      ).not.toBeInTheDocument();
    });
  });
});
