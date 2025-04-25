import React from "react";

import { render, screen } from "@testing-library/react";

import { http, HttpResponse } from "msw";

import mockServer from "test/mock-server";
import { baseUrl } from "test/test-utils";

import RunScriptBatchPaginatedList from "./RunScriptBatchPaginatedList";

// const teamScripts = [

// ]:

// const teamScriptsHandler = http.get(baseUrl("/scripts"), () => {
//   return HttpResponse.json({
//     scripts: teamScripts,
//   });
// });

// describe("RunScriptBatchPaginatedList component", () => {
//   it("Lists a page of the team's scripts", async () => {
//   mockServer.use(teamPoliciesHandler);
//   const { container } = render(
//     <PoliciesPaginatedList
//       isSelected={jest.fn()}
//       onToggleItem={jest.fn()}
//       onCancel={jest.fn()}
//       onSubmit={jest.fn()}
//       teamId={2}
//       footer={null}
//       isUpdating={false}
//     />
//   );
//   await waitForLoadingToFinish(container);

//   const checkboxes = screen.getAllByRole("checkbox");
//   expect(checkboxes).toHaveLength(2);
//   teamPolicies.forEach((item, index) => {
//     expect(checkboxes[index]).toHaveTextContent(item.name);
//     expect(checkboxes[index]).not.toBeChecked();
//   });
// });
// });
