import { screen, waitFor } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import createMockConfig from "__mocks__/configMock";
import createMockHost from "__mocks__/hostMock";
import { createMockHostScript } from "__mocks__/scriptMock";
import createMockUser from "__mocks__/userMock";
import scriptsAPI from "services/entities/scripts";
import { createCustomRenderer } from "test/test-utils";

import ScriptModalGroup from "./ScriptModalGroup";

const setup = (onClose: () => void = noop) => {
  const getHostScripts = jest
    .spyOn(scriptsAPI, "getHostScripts")
    .mockResolvedValue({
      scripts: [createMockHostScript({ name: "script.sh" })],
      meta: { has_next_results: false, has_previous_results: false },
    });
  jest.spyOn(scriptsAPI, "downloadScript").mockResolvedValue("echo hi");
  const currentUser = createMockUser({ global_role: "admin" });
  const render = createCustomRenderer({
    withBackendMock: true,
    context: {
      app: { config: createMockConfig(), currentUser, isPremiumTier: true },
    },
  });
  return {
    getHostScripts,
    ...render(
      <ScriptModalGroup
        currentUser={currentUser}
        host={createMockHost()}
        onCloseScriptModalGroup={onClose}
      />
    ),
  };
};

describe("ScriptModalGroup", () => {
  it("keeps the Name header focused across keyboard sorts", async () => {
    const onClose = jest.fn();
    const { getHostScripts, user } = setup(onClose);

    const header = await screen.findByRole("button", { name: /^Name/ });
    header.focus();
    await user.keyboard("{Enter}");
    await waitFor(() =>
      expect(getHostScripts).toHaveBeenLastCalledWith(
        expect.objectContaining({ order_direction: "desc" })
      )
    );
    await user.keyboard("{Enter}");

    await waitFor(() =>
      expect(getHostScripts).toHaveBeenLastCalledWith(
        expect.objectContaining({ order_direction: "asc" })
      )
    );
    expect(onClose).not.toHaveBeenCalled();
  });

  it("returns to Run after Cancel on confirm then Close on script details", async () => {
    const { user } = setup();

    await user.click(await screen.findByRole("button", { name: "script.sh" }));
    await user.click(await screen.findByText("More actions"));
    await user.click((await screen.findAllByText("Run")).slice(-1)[0]);
    await user.click(await screen.findByRole("button", { name: "Cancel" }));
    await user.click(await screen.findByRole("button", { name: "Close" }));

    expect(
      document.querySelector(".run-script-modal")?.closest(".modal__hidden")
    ).toBeNull();
  });

  it("shows Delete script again after closing it with Escape", async () => {
    const { user } = setup();

    const clickTrash = async () =>
      user.click(await screen.findByRole("button", { name: "Delete script" }));

    await user.click(await screen.findByRole("button", { name: "script.sh" }));
    await clickTrash();
    await user.keyboard("{Escape}");
    await waitFor(() =>
      expect(document.querySelector(".modal__closing")).toBeNull()
    );
    await clickTrash();

    expect(document.querySelectorAll(".modal__closing")).toHaveLength(0);
  });
});
