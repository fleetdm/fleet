import React from "react";
import { screen, waitFor } from "@testing-library/react";

import { IVariable } from "interfaces/variables";
import { UserEvent } from "@testing-library/user-event";
import { createCustomRenderer, createMockRouter } from "test/test-utils";
import { http, HttpResponse } from "msw";
import mockServer from "test/mock-server";

import GlobalVariables from "./GlobalVariables";

const baseUrl = (path: string) => {
  return `/api/latest/fleet${path}`;
};

const baseProps = {
  router: createMockRouter(),
  location: { pathname: "/controls/variables", query: {} },
};

describe("Custom variables", () => {
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        isGlobalAdmin: true,
      },
    },
  });

  const gomURL = "https://www.a.bc";
  const renderInGOM = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: {
        config: {
          gitops: {
            gitops_mode_enabled: true,
            repository_url: gomURL,
          },
        },
        isGlobalAdmin: true,
      },
    },
  });
  describe("empty state", () => {
    const emptyVariablesHandler = http.get(baseUrl("/custom_variables"), () => {
      return HttpResponse.json({
        custom_variables: [],
        count: 0,
        meta: {
          has_previous_results: false,
          has_next_results: false,
        },
      });
    });

    afterAll(() => {
      mockServer.resetHandlers();
    });
    it("renders with Add CTA and edit info when user can edit", async () => {
      mockServer.use(emptyVariablesHandler);

      render(<GlobalVariables {...baseProps} />);
      await waitFor(() => {
        expect(screen.getByText(/No custom variables/i)).toBeInTheDocument();
        expect(
          screen.getByText(
            "Add a custom variable to make it available in scripts and profiles."
          )
        ).toBeInTheDocument();
        // The header Add button stays visible on the empty state, alongside
        // the EmptyState CTA, so both render.
        const addButtons = screen.getAllByRole("button", {
          name: /Add variable/,
        });
        expect(addButtons).toHaveLength(2);
      });
    });

    it("renders without Add CTA and with read-only info when user cannot edit", async () => {
      mockServer.use(emptyVariablesHandler);

      const renderReadOnly = createCustomRenderer({
        withBackendMock: true,
        context: {
          app: {
            isGlobalAdmin: false,
            isGlobalMaintainer: false,
          },
        },
      });

      renderReadOnly(<GlobalVariables {...baseProps} />);
      await waitFor(() => {
        expect(screen.getByText("No custom variables")).toBeInTheDocument();
        expect(
          screen.getByText(
            "No custom variables are available for scripts and profiles."
          )
        ).toBeInTheDocument();
        expect(
          screen.queryByRole("button", { name: "Add variable" })
        ).not.toBeInTheDocument();
      });
    });
  });

  describe("non-empty state", () => {
    const mockVariables: IVariable[] = [
      {
        name: "SECRET_UNO",
        id: 1,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
      {
        name: "SECRET_DOS",
        id: 2,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      },
    ];
    const variablesResponse: { variables: IVariable[] } = { variables: [] };
    // Mock the variables endpoint to return our two test variables.
    const variablesHandler = http.get(baseUrl("/custom_variables"), () => {
      return HttpResponse.json({
        custom_variables: variablesResponse.variables,
        count: variablesResponse.variables.length,
        meta: {
          has_previous_results: false,
          has_next_results: false,
        },
      });
    });
    const addVariableHandler = http.post(
      baseUrl("/custom_variables"),
      async ({ request }) => {
        const { name, value } = (await request.json()) as {
          name: string;
          value: string;
        };
        // const name = formData.get("name");
        // const value = formData.get("value");
        const newVariable = {
          id: mockVariables.length + 1,
          name,
          value,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        } as IVariable;
        variablesResponse.variables.push(newVariable);
        return HttpResponse.json(newVariable);
      }
    );
    const deleteVariableHandler = http.delete(
      baseUrl("/custom_variables/:id"),
      async ({ request }) => {
        const id = request.url.split("/").pop();
        if (!id) {
          throw new Error("Variable ID not found in request URL");
        }
        variablesResponse.variables = variablesResponse.variables.filter(
          (variable) => variable.id !== parseInt(id, 10)
        );
        return HttpResponse.json({ success: true });
      }
    );
    beforeEach(async () => {
      // Wait for the query stale timer to expire.
      await new Promise((resolve) => setTimeout(resolve, 250));
      mockServer.use(variablesHandler);
      mockServer.use(addVariableHandler);
      mockServer.use(deleteVariableHandler);
      variablesResponse.variables = [...mockVariables];
    });

    it("renders when variables are saved", async () => {
      render(<GlobalVariables {...baseProps} />);
      await waitFor(
        () => {
          expect(screen.getByText("SECRET_UNO")).toBeInTheDocument();
          expect(screen.getByText("SECRET_DOS")).toBeInTheDocument();
        },
        {
          timeout: 3000,
        }
      );
    });

    describe("gitops mode", () => {
      it("renders the add button disabled in GitOps mode", async () => {
        renderInGOM(<GlobalVariables {...baseProps} />);

        let addVariableButton;
        await waitFor(() => {
          addVariableButton = screen.getByRole("button", {
            name: /Add variable/,
          });

          expect(addVariableButton).toBeInTheDocument();
        });
        if (!addVariableButton) {
          throw new Error("Add variable button not found");
        }

        expect(addVariableButton).toHaveAttribute("disabled");
        expect(addVariableButton).toHaveClass("button--disabled");

        // Tooltip behavior covered in GitOpsModeWrapper.tests.tsx; omitted here due to flakiness
      });

      it("deleting a variable is successful in GitOps mode", async () => {
        const { user } = renderInGOM(<GlobalVariables {...baseProps} />);
        await waitFor(() => {
          expect(screen.getByText("SECRET_UNO")).toBeInTheDocument();
        });
        const deleteButton = screen.getByRole("button", {
          name: "Delete SECRET_UNO",
        });
        expect(deleteButton).toBeInTheDocument();
        await user.click(deleteButton);
        await waitFor(() => {
          expect(
            screen.getByText(/Delete custom variable\?/)
          ).toBeInTheDocument();
          expect(screen.getByText(/This will delete the/)).toBeInTheDocument();
        });
        await new Promise((resolve) => setTimeout(resolve, 250));
        await user.click(screen.getByRole("button", { name: "Delete" }));
        await waitFor(() => {
          expect(
            screen.queryByText(/Delete custom variable\?/)
          ).not.toBeInTheDocument();
          expect(screen.queryByText("SECRET_UNO")).not.toBeInTheDocument();
          expect(screen.queryByText("SECRET_DOS")).toBeInTheDocument();
        });
      });
    });

    describe("adding a new variable", () => {
      const getAddVariableUI = async () => {
        let nameInput;
        let valueInput;
        let saveButton;
        await waitFor(() => {
          nameInput = screen.getByLabelText("Name");
          expect(nameInput).toBeInTheDocument();
          valueInput = screen.getByLabelText("Value");
          expect(valueInput).toBeInTheDocument();
          saveButton = screen.getByRole("button", { name: "Save" });
          expect(saveButton).toBeInTheDocument();
        });
        if (!nameInput || !valueInput || !saveButton) {
          throw new Error("UI not found");
        }
        return { nameInput, valueInput, saveButton };
      };

      let user: UserEvent;
      beforeEach(async () => {
        ({ user } = render(<GlobalVariables {...baseProps} />));
        let addVariableButton;
        await waitFor(() => {
          addVariableButton = screen.getByRole("button", {
            name: /Add variable/,
          });
          expect(addVariableButton).toBeInTheDocument();
        });
        if (!addVariableButton) {
          throw new Error("Add variable button not found");
        }
        await user.click(addVariableButton);
      });
      it("is successful with valid name and value", async () => {
        const { nameInput, valueInput, saveButton } = await getAddVariableUI();
        await user.type(nameInput, "New_Secret");
        await user.type(valueInput, "Secret Value");
        await user.click(saveButton);
        await waitFor(() => {
          expect(screen.getByText("SECRET_UNO")).toBeInTheDocument();
          expect(screen.getByText("SECRET_DOS")).toBeInTheDocument();
          expect(screen.getByText("NEW_SECRET")).toBeInTheDocument();
        });
      });
      it("does not allow saving without name", async () => {
        const { valueInput, saveButton } = await getAddVariableUI();
        await user.type(valueInput, "Secret Value");
        await user.click(saveButton);
        await waitFor(() => {
          expect(screen.getByText("Name is required")).toBeInTheDocument();
          expect(saveButton).toBeDisabled();
        });
      });
      it("does not allow saving without value", async () => {
        const { nameInput, saveButton } = await getAddVariableUI();
        await user.type(nameInput, "Secret Name");
        await user.click(saveButton);
        await waitFor(() => {
          expect(screen.getByText("Value is required")).toBeInTheDocument();
          expect(saveButton).toBeDisabled();
        });
      });
      it("does not allow saving with invalid name", async () => {
        const { nameInput, valueInput, saveButton } = await getAddVariableUI();
        await user.type(nameInput, "COOL!"); // Invalid name
        await user.type(valueInput, "Secret Value");
        await user.click(saveButton);
        await waitFor(() => {
          expect(
            screen.getByText(
              "Name may only include uppercase letters, numbers, and underscores"
            )
          ).toBeInTheDocument();
          expect(saveButton).toBeDisabled();
        });
      });
      it("caps the name input at 255 characters (matches DB varchar(255))", async () => {
        const { nameInput } = await getAddVariableUI();
        expect((nameInput as HTMLInputElement).maxLength).toBe(255);
      });
    });

    it("deleting a variable is successful", async () => {
      const { user } = render(<GlobalVariables {...baseProps} />);
      await waitFor(() => {
        expect(screen.getByText("Add variable")).toBeInTheDocument();
      });
      await waitFor(() => {
        expect(screen.getByText("SECRET_UNO")).toBeInTheDocument();
      });
      // The row action is a trash-icon button labeled "Delete <name>".
      const deleteButton = screen.getByRole("button", {
        name: "Delete SECRET_UNO",
      });
      expect(deleteButton).toBeInTheDocument();
      // Click it.
      await user.click(deleteButton);
      // Confirm the deletion.
      await waitFor(() => {
        expect(
          screen.getByText(/Delete custom variable\?/)
        ).toBeInTheDocument();
        expect(screen.getByText(/This will delete the/)).toBeInTheDocument();
      });
      await new Promise((resolve) => setTimeout(resolve, 250));
      await user.click(screen.getByRole("button", { name: "Delete" }));
      await waitFor(() => {
        expect(
          screen.queryByText(/Delete custom variable\?/)
        ).not.toBeInTheDocument();
        expect(screen.queryByText("SECRET_UNO")).not.toBeInTheDocument();
        expect(screen.queryByText("SECRET_DOS")).toBeInTheDocument();
      });
    });
  });
});
