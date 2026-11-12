import { screen, waitFor } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import createMockTeam from "__mocks__/teamMock";
import createMockUser from "__mocks__/userMock";
import { notify } from "components/ToastNotification";
import mockServer from "test/mock-server";
import {
  createCustomRenderer,
  baseUrl,
  createMockRouter,
} from "test/test-utils";

import ManageFleetsPage from "./ManageFleetsPage";

jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    batch: jest.fn(),
    dismiss: jest.fn(),
  },
}));

const CONFLICT_ERROR = "Choose a fleet name distinct from other fleets.";
const NO_LETTER_OR_NUMBER_ERROR =
  "Fleet name must include at least one letter or number.";

const conflictResponse = () =>
  HttpResponse.json(
    {
      message: "Conflict",
      errors: [
        {
          name: "base",
          reason:
            'A fleet named "Workstations" already exists. Choose a fleet name distinct from other fleets.',
        },
      ],
    },
    { status: 409 }
  );

const invalidNameResponse = (reason: string) =>
  HttpResponse.json(
    { message: "Validation Failed", errors: [{ name: "name", reason }] },
    { status: 422 }
  );

const serverErrorResponse = () =>
  HttpResponse.json(
    { message: "error", errors: [{ name: "base", reason: "boom" }] },
    { status: 500 }
  );

const renderPage = () => {
  mockServer.use(
    http.get(baseUrl("/me"), () =>
      HttpResponse.json({ user: createMockUser({ global_role: "admin" }) })
    ),
    http.get(baseUrl("/fleets"), () =>
      HttpResponse.json({
        fleets: [createMockTeam({ id: 1, name: "Workstations" })],
      })
    )
  );
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isGlobalAdmin: true,
        isPremiumTier: true,
        currentUser: createMockUser({ global_role: "admin" }),
        config: createMockConfig(),
        setAvailableTeams: jest.fn(),
        setCurrentTeam: jest.fn(),
        setCurrentUser: jest.fn(),
        setUserSettings: jest.fn(),
      },
    },
  });
  return render(
    <ManageFleetsPage
      router={createMockRouter()}
      location={{ pathname: "/settings/fleets", query: {} }}
    />
  );
};

describe("ManageFleetsPage", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe("creating a fleet", () => {
    const submitCreate = async (name: string) => {
      const { user } = renderPage();
      await user.click(
        await screen.findByRole("button", { name: "Add fleet" })
      );
      await user.type(screen.getByLabelText("Fleet name"), name);
      await user.click(screen.getByRole("button", { name: "Create" }));
    };

    it("shows the conflict error under the name field and keeps the modal open on a 409", async () => {
      mockServer.use(http.post(baseUrl("/fleets"), conflictResponse));

      await submitCreate("work-stations");

      expect(await screen.findByText(CONFLICT_ERROR)).toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: "Create" })
      ).toBeInTheDocument();
      expect(notify.error).not.toHaveBeenCalled();
    });

    it("shows the letter-or-number error under the name field", async () => {
      mockServer.use(
        http.post(baseUrl("/fleets"), () =>
          invalidNameResponse("must include at least one letter or number")
        )
      );

      await submitCreate("🚀");

      expect(
        await screen.findByText(NO_LETTER_OR_NUMBER_ERROR)
      ).toBeInTheDocument();
      expect(
        screen.getByRole("button", { name: "Create" })
      ).toBeInTheDocument();
      expect(notify.error).not.toHaveBeenCalled();
    });

    it("shows the reserved-name error under the name field", async () => {
      mockServer.use(
        http.post(baseUrl("/fleets"), () =>
          invalidNameResponse('"Unassigned" is a reserved fleet name')
        )
      );

      await submitCreate("Unassigned");

      expect(
        await screen.findByText(
          '"Unassigned" is a reserved fleet name. Please try another name.'
        )
      ).toBeInTheDocument();
      expect(notify.error).not.toHaveBeenCalled();
    });

    it("shows a toast and closes the modal for other errors", async () => {
      mockServer.use(http.post(baseUrl("/fleets"), serverErrorResponse));

      await submitCreate("Servers");

      await waitFor(() => {
        expect(notify.error).toHaveBeenCalledWith(
          "Could not create fleet. Please try again.",
          expect.anything()
        );
      });
      expect(
        screen.queryByRole("button", { name: "Create" })
      ).not.toBeInTheDocument();
    });
  });

  describe("renaming a fleet", () => {
    const submitRename = async (name: string) => {
      const { user } = renderPage();
      await user.click(await screen.findByText("Actions"));
      await user.click(await screen.findByText("Rename"));
      const nameInput = screen.getByLabelText("Fleet name");
      await user.clear(nameInput);
      await user.type(nameInput, name);
      await user.click(screen.getByRole("button", { name: "Save" }));
    };

    it("shows the conflict error under the name field on a 409, with no toast", async () => {
      mockServer.use(http.patch(baseUrl("/fleets/1"), conflictResponse));

      await submitRename("Laptops!");

      expect(await screen.findByText(CONFLICT_ERROR)).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();
      expect(notify.error).not.toHaveBeenCalled();
    });

    it("shows the letter-or-number error under the name field", async () => {
      mockServer.use(
        http.patch(baseUrl("/fleets/1"), () =>
          invalidNameResponse("must include at least one letter or number")
        )
      );

      await submitRename("---");

      expect(
        await screen.findByText(NO_LETTER_OR_NUMBER_ERROR)
      ).toBeInTheDocument();
      expect(notify.error).not.toHaveBeenCalled();
    });

    it("shows the reserved-name error under the name field", async () => {
      mockServer.use(
        http.patch(baseUrl("/fleets/1"), () =>
          invalidNameResponse('"All fleets" is a reserved fleet name')
        )
      );

      await submitRename("All fleets");

      expect(
        await screen.findByText(
          '"All fleets" is a reserved fleet name. Please try another name.'
        )
      ).toBeInTheDocument();
      expect(notify.error).not.toHaveBeenCalled();
    });

    it("sends a rename to a variant of the fleet's own name", async () => {
      const patchSpy = jest.fn();
      mockServer.use(
        http.patch(baseUrl("/fleets/1"), async ({ request }) => {
          patchSpy(await request.json());
          return HttpResponse.json({
            fleet: createMockTeam({ id: 1, name: "workstations" }),
          });
        })
      );

      await submitRename("workstations");

      await waitFor(() => {
        expect(notify.success).toHaveBeenCalledWith(
          "Successfully updated fleet name to workstations."
        );
      });
      expect(patchSpy).toHaveBeenCalledWith({ name: "workstations" });
      expect(
        screen.queryByRole("button", { name: "Save" })
      ).not.toBeInTheDocument();
    });

    it("shows a toast for other errors", async () => {
      mockServer.use(http.patch(baseUrl("/fleets/1"), serverErrorResponse));

      await submitRename("Servers");

      await waitFor(() => {
        expect(notify.error).toHaveBeenCalledWith(
          "Could not rename Workstations. Please try again.",
          expect.anything()
        );
      });
    });
  });
});
