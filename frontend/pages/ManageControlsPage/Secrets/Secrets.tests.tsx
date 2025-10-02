import React from "react";
import { getByText, screen, waitFor } from "@testing-library/react";

import { ISecret } from "interfaces/secrets";
import { UserEvent } from "@testing-library/user-event";
import { createCustomRenderer } from "test/test-utils";
import { http, HttpResponse } from "msw";
import mockServer from "test/mock-server";

import Secrets from "./Secrets";

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
    afterAll(() => {
      mockServer.resetHandlers();
    });
    it("renders when no secrets are saved", async () => {
      const secretsHandler = http.get(baseUrl("/custom_variables"), () => {
        return HttpResponse.json({
          custom_variables: [],
          count: 0,
          has_prev_results: false,
          has_next_results: false,
        });
      });
      mockServer.use(secretsHandler);

      render(<Secrets />);
      await waitFor(() => {
        expect(
          screen.getByText("No custom variables created yet")
        ).toBeInTheDocument();
        expect(
          screen.getByRole("button", { name: "Add custom variable" })
        ).toBeInTheDocument();
      });
    });
  });

  describe("non-empty state", () => {
    const mockSecrets: ISecret[] = [
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
    const secretsResponse: { secrets: ISecret[] } = { secrets: [] };
    // Mock the scripts endpoint to return our two test scripts.
    const secretsHandler = http.get(baseUrl("/custom_variables"), () => {
      return HttpResponse.json({
        custom_variables: secretsResponse.secrets,
        count: mockSecrets.length,
        has_prev_results: false,
        has_next_results: false,
      });
    });
    const addSecretHandler = http.post(
      baseUrl("/custom_variables"),
      async ({ request }) => {
        const { name, value } = (await request.json()) as {
          name: string;
          value: string;
        };
        // const name = formData.get("name");
        // const value = formData.get("value");
        const newSecret = {
          id: mockSecrets.length + 1,
          name,
          value,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        } as ISecret;
        secretsResponse.secrets.push(newSecret);
        return HttpResponse.json(newSecret);
      }
    );
    const deleteSecretHandler = http.delete(
      baseUrl("/custom_variables/:id"),
      async ({ request }) => {
        const id = request.url.split("/").pop();
        if (!id) {
          throw new Error("Secret ID not found in request URL");
        }
        secretsResponse.secrets = secretsResponse.secrets.filter(
          (secret) => secret.id !== parseInt(id, 10)
        );
        return HttpResponse.json({ success: true });
      }
    );
    beforeEach(async () => {
      // Wait for the query stale timer to expire.
      await new Promise((resolve) => setTimeout(resolve, 250));
      mockServer.use(secretsHandler);
      mockServer.use(addSecretHandler);
      mockServer.use(deleteSecretHandler);
      secretsResponse.secrets = [...mockSecrets];
    });

    it("renders when secrets are saved", async () => {
      render(<Secrets />);
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
        const { user } = renderInGOM(<Secrets />);

        let addSecretButton;
        await waitFor(() => {
          addSecretButton = screen.getByRole("button", {
            name: /Add custom variable/,
          });

          expect(addSecretButton).toBeInTheDocument();
        });
        if (!addSecretButton) {
          throw new Error("Add custom variable button not found");
        }

        expect(addSecretButton).toHaveAttribute("disabled");
        expect(addSecretButton).toHaveClass("button--disabled");

        // Tooltip behavior covered in GitOpsModeWrapper.tests.tsx; omitted here due to flakiness
      });

      it("deleting a secret is successful in GitOps mode", async () => {
        const { user } = renderInGOM(<Secrets />);
        await waitFor(() => {
          expect(screen.getByText("Add custom variable")).toBeInTheDocument();
        });
        // Get the element with SECRET_UNO in it.
        let secretUno: HTMLElement | null = null;
        await waitFor(() => {
          secretUno = screen.getByText("SECRET_UNO");
          expect(secretUno).toBeInTheDocument();
        });
        if (secretUno === null) {
          throw new Error("Secret not found");
        }
        // Find the element with .paginated-list__row class that is ancestor to that element.
        const secretUnoRow = (secretUno as HTMLElement).closest(
          ".paginated-list__row"
        );
        expect(secretUnoRow).toBeInTheDocument();
        if (!secretUnoRow) {
          throw new Error("Secret row not found");
        }
        // Find the element with data-id="trash-icon"
        const trashIcon = secretUnoRow.querySelector(
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

    describe("adding a new secret", () => {
      const getAddSecretUI = async () => {
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
        ({ user } = render(<Secrets />));
        let addSecretButton;
        await waitFor(() => {
          addSecretButton = screen.getByRole("button", {
            name: /Add custom variable/,
          });
          expect(addSecretButton).toBeInTheDocument();
        });
        if (!addSecretButton) {
          throw new Error("Add custom variable button not found");
        }
        await user.click(addSecretButton);
      });
      it("is successful with valid name and value", async () => {
        const { nameInput, valueInput, saveButton } = await getAddSecretUI();
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
        const { valueInput, saveButton } = await getAddSecretUI();
        await user.type(valueInput, "Secret Value");
        await user.click(saveButton);
        await waitFor(() => {
          expect(screen.getByText("Name is required")).toBeInTheDocument();
          expect(saveButton).toBeDisabled();
        });
      });
      it("does not allow saving without value", async () => {
        const { nameInput, saveButton } = await getAddSecretUI();
        await user.type(nameInput, "Secret Name");
        await user.click(saveButton);
        await waitFor(() => {
          expect(screen.getByText("Value is required")).toBeInTheDocument();
          expect(saveButton).toBeDisabled();
        });
      });
      it("does not allow saving with invalid name", async () => {
        const { nameInput, valueInput, saveButton } = await getAddSecretUI();
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
        const { nameInput, valueInput, saveButton } = await getAddSecretUI();
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

    it("deleting a secret is successful", async () => {
      const { user } = render(<Secrets />);
      await waitFor(() => {
        expect(screen.getByText("Add custom variable")).toBeInTheDocument();
      });
      // Get the element with SECRET_UNO in it.
      let secretUno: HTMLElement | null = null;
      await waitFor(() => {
        secretUno = screen.getByText("SECRET_UNO");
        expect(secretUno).toBeInTheDocument();
      });
      if (secretUno === null) {
        throw new Error("Secret not found");
      }
      // Find the element with .paginated-list__row class that is ancestor to that element.
      const secretUnoRow = (secretUno as HTMLElement).closest(
        ".paginated-list__row"
      );
      expect(secretUnoRow).toBeInTheDocument();
      if (!secretUnoRow) {
        throw new Error("Secret row not found");
      }
      // Find the element with data-id="trash-icon"
      const trashIcon = secretUnoRow.querySelector(
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
