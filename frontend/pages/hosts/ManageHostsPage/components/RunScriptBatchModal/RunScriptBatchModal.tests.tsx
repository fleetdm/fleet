import React from "react";
import { screen, waitFor } from "@testing-library/react";

import { IScript } from "interfaces/script";
import { createCustomRenderer } from "test/test-utils";
import { http, HttpResponse } from "msw";
import mockServer from "test/mock-server";
import RunScriptBatchModal from "./RunScriptBatchModal";
import { UserEvent } from "@testing-library/user-event";

const baseUrl = (path: string) => {
  return `/api/latest/fleet${path}`;
};

const windowsScript: IScript = {
  id: 123,
  team_id: 1,
  name: "winscript.ps1",
  created_at: "2023-01-01T00:00:00Z",
  updated_at: "2023-01-01T00:00:00Z",
};

const linuxScript: IScript = {
  id: 456,
  team_id: 1,
  name: "linuxscript.sh",
  created_at: "2023-01-01T00:00:00Z",
  updated_at: "2023-01-01T00:00:00Z",
};

jest.mock("../RunScriptBatchPaginatedList", () => {
  return {
    __esModule: true,
    default: ({ onRunScript }: { onRunScript: (script: IScript) => void }) => {
      return (
        <div>
          <div onClick={() => onRunScript(windowsScript)}>windows script</div>
          <div onClick={() => onRunScript(linuxScript)}>linux script</div>
        </div>
      );
    },
  };
});

const scriptsHandler = http.get(baseUrl("/scripts"), () => {
  return HttpResponse.json({
    data: {
      scripts: [windowsScript, linuxScript],
    },
  });
});

const selectScript = (user: UserEvent, platform: string) => {
  waitFor(async () => {
    const el = screen.getByText(`${platform} script`);
    expect(el).toBeInTheDocument();
    await user.click(el);
  });
};

describe("RunScriptBatchModal", () => {
  beforeEach(() => {
    mockServer.use(scriptsHandler);
  });

  const render = createCustomRenderer({
    withBackendMock: true,
  });

  const defaultProps = {
    runByFilters: false,
    // since teamId has multiple uses in this component, it's passed in as its own prop and added to
    // `filters` as needed
    filters: { team_id: 1, status: "" },
    teamId: 1,
    // If we are on the free tier, we don't want to apply any kind of team filters (since the feature is Premium only).
    isFreeTier: false,
    totalFilteredHostsCount: 2,
    selectedHostIds: [1, 2],
    onCancel: () => null,
  };

  it("lists the scripts available for batch running", () => {
    render(<RunScriptBatchModal {...defaultProps} />);
    waitFor(() => {
      const windowsScriptElement = screen.getByText("windows script");
      const linuxScriptElement = screen.getByText("linux script");
      expect(windowsScriptElement).toBeInTheDocument();
      expect(linuxScriptElement).toBeInTheDocument();
    });
  });

  describe("after clicking run script", () => {
    it("shows the correct heading for linux/macos scripts", async () => {
      const { user } = render(<RunScriptBatchModal {...defaultProps} />);

      await selectScript(user, "linux");
      expect(
        screen.getByText(
          /linuxscript\.sh will run on compatible hosts (macOS and Linux)/i
        )
      ).toBeInTheDocument();
    });

    it("shows the correct heading for windows", async () => {
      const { user } = render(<RunScriptBatchModal {...defaultProps} />);

      await selectScript(user, "windows");
      expect(
        screen.getByText(
          /windowscript\.ps1 will run on compatible hosts (Windows)/i
        )
      ).toBeInTheDocument();
    });

    it("does not show the scheduling UI if 'run now' is selected", () => {
      render(<RunScriptBatchModal {...defaultProps} />);
    });

    it("shows the scheduling UI if 'schedule for later' is selected", () => {
      render(<RunScriptBatchModal {...defaultProps} />);
    });

    describe("run now", () => {
      it("should call the API with no not_before param", () => {
        render(<RunScriptBatchModal {...defaultProps} />);
      });
    });

    describe("schedule for later", () => {
      it("requires a valid date", () => {
        render(<RunScriptBatchModal {...defaultProps} />);
      });

      it("requires a valid time", () => {
        render(<RunScriptBatchModal {...defaultProps} />);
      });

      it("should call the API with a not_before param", () => {
        render(<RunScriptBatchModal {...defaultProps} />);
      });
    });
  });
});
