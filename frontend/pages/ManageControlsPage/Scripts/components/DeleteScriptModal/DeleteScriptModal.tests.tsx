import { screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import scriptAPI from "services/entities/scripts";
import { renderWithSetup } from "test/test-utils";

import DeleteScriptModal from "./DeleteScriptModal";

describe("DeleteScriptModal", () => {
  it("sends one delete request when Enter is pressed twice", async () => {
    const deleteScript = jest
      .spyOn(scriptAPI, "deleteScript")
      .mockReturnValue(new Promise(noop));

    const { user } = renderWithSetup(
      <DeleteScriptModal
        scriptName="script.sh"
        scriptId={7}
        onCancel={noop}
        afterDelete={noop}
      />
    );

    expect(screen.getByText("Delete script")).toBeVisible();
    await user.keyboard("{Enter}");
    await user.keyboard("{Enter}");

    expect(deleteScript).toHaveBeenCalledTimes(1);
    expect(deleteScript).toHaveBeenCalledWith(7);
  });
});
