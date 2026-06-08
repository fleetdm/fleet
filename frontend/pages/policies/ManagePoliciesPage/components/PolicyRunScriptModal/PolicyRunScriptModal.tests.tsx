import { IScript } from "interfaces/script";
import { IFormPolicy } from "../PoliciesPaginatedList/PoliciesPaginatedList";
import { getTrulyDirtyItems } from "./PolicyRunScriptModal";

describe("getTrulyDirtyItems", () => {
  const createMockPolicyForScriptAutomation = (
    overrides: Partial<IFormPolicy> = {}
  ): IFormPolicy =>
    ({
      id: 1,
      name: "Policy",
      run_script: undefined,
      runScriptEnabled: false,
      scriptIdToRun: undefined,
      ...overrides,
    } as IFormPolicy);

  it("returns only policies that changed enablement or script", () => {
    const originalScriptId = 10;

    const dirtyItems: IFormPolicy[] = [
      // 1. Unchanged: originally enabled, still enabled, same script -> should be filtered out
      createMockPolicyForScriptAutomation({
        id: 1,
        run_script: { id: originalScriptId } as Pick<IScript, "id" | "name">,
        runScriptEnabled: true,
        scriptIdToRun: originalScriptId,
      }),

      // 2. Turned on: originally disabled (no run_script), now enabled with script -> included
      createMockPolicyForScriptAutomation({
        id: 2,
        run_script: undefined,
        runScriptEnabled: true,
        scriptIdToRun: 20,
      }),

      // 3. Turned off: originally enabled, now disabled -> included
      createMockPolicyForScriptAutomation({
        id: 3,
        run_script: { id: originalScriptId } as Pick<IScript, "id" | "name">,
        runScriptEnabled: false,
        scriptIdToRun: undefined,
      }),

      // 4. Script changed: originally enabled with script A, now enabled with script B -> included
      createMockPolicyForScriptAutomation({
        id: 4,
        run_script: { id: originalScriptId } as Pick<IScript, "id" | "name">,
        runScriptEnabled: true,
        scriptIdToRun: 30,
      }),
    ];

    const result = getTrulyDirtyItems(dirtyItems);

    const ids = result.map((p) => p.id).sort();
    expect(ids).toEqual([2, 3, 4]);
    expect(ids).not.toContain(1);
  });
});
