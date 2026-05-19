import React from "react";
import { screen, waitFor } from "@testing-library/react";

import { IVariable } from "interfaces/variables";
import { UserEvent } from "@testing-library/user-event";
import { createCustomRenderer } from "test/test-utils";
import { http, HttpResponse } from "msw";
import mockServer from "test/mock-server";

import Variables from "./Variables";

const baseUrl = (path: string) => {
  return `/api/latest/fleet${path}`;
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

      render(<Variables />);
      await waitFor(() => {
        expect(screen.getByText(/No custom variables/i)).toBeInTheDocument();
        expect(
          screen.getByText(
            "Add a custom variable to make it available in scripts and profiles."
          )
        ).toBeInTheDocument();
        expect(
          screen.getByRole("button", { name: "Add custom variable" })
        ).toBeInTheDocument();
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

      renderReadOnly(<Variables />);
      await waitFor(() => {
        expect(screen.getByText("No custom variables")).toBeInTheDocument();
        expect(
          screen.getByText(
            "No custom variables are available for scripts and profiles."
          )
        ).toBeInTheDocument();
        expect(
          screen.queryByRole("button", { name: "Add custom variable" })
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
      render(<Variables />);
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
        renderInGOM(<Variables />);

        let addVariableButton;
        await waitFor(() => {
          addVariableButton = screen.getByRole("button", {
            name: /Add custom variable/,
          });

          expect(addVariableButton).toBeInTheDocument();
        });
        if (!addVariableButton) {
          throw new Error("Add custom variable button not found");
        }

        expect(addVariableButton).toHaveAttribute("disabled");
        expect(addVariableButton).toHaveClass("button--disabled");

        // Tooltip behavior covered in GitOpsModeWrapper.tests.tsx; omitted here due to flakiness
      });

      it("deleting a variable is successful in GitOps mode", async () => {
        const { user } = renderInGOM(<Variables />);
        await waitFor(() => {
          expect(screen.getByText("Add custom variable")).toBeInTheDocument();
        });
        // Get the element with SECRET_UNO in it.
        let variableUno: HTMLElement | null = null;
        await waitFor(() => {
          variableUno = screen.getByText("SECRET_UNO");
          expect(variableUno).toBeInTheDocument();
        });
        if (variableUno === null) {
          throw new Error("Variable not found");
        }
        // Find the element with .paginated-list__row class that is ancestor to that element.
        const variableUnoRow = (variableUno as HTMLElement).closest(
          ".paginated-list__row"
        );
        expect(variableUnoRow).toBeInTheDocument();
        if (!variableUnoRow) {
          throw new Error("Variable row not found");
        }
        // Find the element with data-id="trash-icon"
        const trashIcon = variableUnoRow.querySelector(
          "[data-testid='trash-icon']"
        );
        expect(trashIcon).toBeInTheDocument();
        if (!trashIcon) {
          throw new Error("Trash icon not found");
        }
        // Click it.
        await user.click(trashIcon);
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
        ({ user } = render(<Variables />));
        let addVariableButton;
        await waitFor(() => {
          addVariableButton = screen.getByRole("button", {
            name: /Add custom variable/,
          });
          expect(addVariableButton).toBeInTheDocument();
        });
        if (!addVariableButton) {
          throw new Error("Add custom variable button not found");
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
      it("does not allow saving very long name", async () => {
        const { nameInput, valueInput, saveButton } = await getAddVariableUI();
        await user.type(nameInput, new Array(256).fill("A").join("")); // Invalid name
        await user.type(valueInput, "a value");
        await user.click(saveButton);
        await waitFor(() => {
          expect(
            screen.getByText("Name may not exceed 255 characters")
          ).toBeInTheDocument();
          expect(saveButton).toBeDisabled();
        });
      });
    });

    it("deleting a variable is successful", async () => {
      const { user } = render(<Variables />);
      await waitFor(() => {
        expect(screen.getByText("Add custom variable")).toBeInTheDocument();
      });
      // Get the element with SECRET_UNO in it.
      let variableUno: HTMLElement | null = null;
      await waitFor(() => {
        variableUno = screen.getByText("SECRET_UNO");
        expect(variableUno).toBeInTheDocument();
      });
      if (variableUno === null) {
        throw new Error("Variable not found");
      }
      // Find the element with .paginated-list__row class that is ancestor to that element.
      const variableUnoRow = (variableUno as HTMLElement).closest(
        ".paginated-list__row"
      );
      expect(variableUnoRow).toBeInTheDocument();
      if (!variableUnoRow) {
        throw new Error("Variable row not found");
      }
      // Find the element with data-id="trash-icon"
      const trashIcon = variableUnoRow.querySelector(
        "[data-testid='trash-icon']"
      );
      expect(trashIcon).toBeInTheDocument();
      if (!trashIcon) {
        throw new Error("Trash icon not found");
      }
      // Click it.
      await user.click(trashIcon);
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
