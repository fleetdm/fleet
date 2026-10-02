import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";

import { createMockConfig } from "__mocks__/configMock";
import { notify } from "components/ToastNotification";
import { IConfig } from "interfaces/config";
import mdmAPI from "services/entities/mdm";
import { createCustomRenderer } from "test/test-utils";

import EulaSection, { IPlatformEula } from "./EulaSection";
import { EulaPlatform } from "./helpers";

jest.mock("services/entities/mdm", () => ({
  __esModule: true,
  default: {
    uploadEULA: jest.fn().mockResolvedValue({}),
    uploadWindowsEULA: jest.fn().mockResolvedValue({}),
    deleteEULA: jest.fn().mockResolvedValue({}),
    deleteWindowsEULA: jest.fn().mockResolvedValue({}),
  },
}));

jest.mock("components/ToastNotification", () => ({
  notify: {
    success: jest.fn(),
    error: jest.fn(),
    batch: jest.fn(),
    dismiss: jest.fn(),
  },
}));

const macOSMetadata = {
  name: "eula.pdf",
  token: "mac-token",
  created_at: "2026-10-01T00:00:00Z",
};
const windowsMetadata = {
  name: "terms.md",
  token: "win-token",
  created_at: "2026-10-01T00:00:00Z",
};

const GITOPS_CONFIG: Partial<IConfig> = {
  gitops: {
    gitops_mode_enabled: true,
    repository_url: "https://example.com/fleet-gitops",
    exceptions: { labels: false, software: false, secrets: true },
  },
};

const getFileInput = () =>
  document.querySelector('input[type="file"]') as HTMLInputElement;

const renderSection = (
  overrides: Partial<Record<EulaPlatform, IPlatformEula>> = {},
  configOverrides: Partial<IConfig> = {}
) => {
  const render = createCustomRenderer({
    context: {
      app: { isPremiumTier: true, config: createMockConfig(configOverrides) },
    },
  });
  const eulas: Record<EulaPlatform, IPlatformEula> = {
    darwin: { isAvailable: true, onChange: jest.fn() },
    windows: { isAvailable: true, onChange: jest.fn() },
    ...overrides,
  };
  return render(<EulaSection eulas={eulas} />);
};

describe("EulaSection", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("shows a tab per platform with the macOS uploader first", () => {
    renderSection();

    expect(screen.getByRole("tab", { name: "macOS" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Windows" })).toBeInTheDocument();
    expect(screen.getByText("PDF (.pdf)")).toBeInTheDocument();
  });

  it("shows the markdown uploader on the Windows tab", async () => {
    const { user } = renderSection();

    await user.click(screen.getByRole("tab", { name: "Windows" }));

    expect(
      screen.getByText(
        "Export your document as a markdown (.md) file and upload it."
      )
    ).toBeInTheDocument();
  });

  it("opens on the Windows tab and explains the macOS requirement when Apple MDM is not set up", async () => {
    const { user } = renderSection({
      darwin: { isAvailable: false, onChange: jest.fn() },
    });

    expect(
      screen.getByRole("tab", { name: "Windows", selected: true })
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Export your document as a markdown (.md) file and upload it."
      )
    ).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "macOS" }));

    expect(
      screen.getByText(
        "To require a EULA on macOS hosts, first turn on Apple (macOS, iOS, iPadOS) MDM and add an Apple Business Manager token."
      )
    ).toBeInTheDocument();
  });

  it("explains the Windows requirement when Windows MDM is off", async () => {
    const { user } = renderSection({
      windows: { isAvailable: false, onChange: jest.fn() },
    });

    await user.click(screen.getByRole("tab", { name: "Windows" }));

    expect(
      screen.getByText(
        "To require terms on Windows hosts, first turn on Windows MDM."
      )
    ).toBeInTheDocument();
  });

  it("shows the uploaded Windows agreement with an example of the enrollment page", async () => {
    const { user } = renderSection({
      windows: {
        isAvailable: true,
        metadata: windowsMetadata,
        onChange: jest.fn(),
      },
    });

    await user.click(screen.getByRole("tab", { name: "Windows" }));
    expect(screen.getByText("terms.md")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Example EULA" }));
    expect(screen.getByText("Example Windows EULA")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Close" }));
    expect(screen.queryByText("Example Windows EULA")).not.toBeInTheDocument();
  });

  it("deletes the Windows agreement through the Windows endpoint and refetches", async () => {
    const onChange = jest.fn();
    const { user } = renderSection({
      windows: {
        isAvailable: true,
        metadata: windowsMetadata,
        onChange,
      },
    });

    await user.click(screen.getByRole("tab", { name: "Windows" }));
    await user.click(screen.getByRole("button", { name: "Delete EULA" }));
    expect(
      screen.getByText("Fleet’s default terms will be shown instead.", {
        exact: false,
      })
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Delete" }));

    await waitFor(() => {
      expect(jest.mocked(mdmAPI.deleteWindowsEULA)).toHaveBeenCalledTimes(1);
    });
    expect(jest.mocked(mdmAPI.deleteWindowsEULA)).toHaveBeenCalledWith(
      "win-token"
    );
    expect(jest.mocked(mdmAPI.deleteEULA)).not.toHaveBeenCalled();
    expect(onChange).toHaveBeenCalledTimes(1);
  });

  it("deletes the macOS EULA through the macOS endpoint", async () => {
    const onChange = jest.fn();
    const { user } = renderSection({
      darwin: {
        isAvailable: true,
        metadata: macOSMetadata,
        onChange,
      },
    });

    await user.click(screen.getByRole("button", { name: "Delete EULA" }));
    await user.click(screen.getByRole("button", { name: "Delete" }));

    await waitFor(() => {
      expect(jest.mocked(mdmAPI.deleteEULA)).toHaveBeenCalledTimes(1);
    });
    expect(jest.mocked(mdmAPI.deleteEULA)).toHaveBeenCalledWith("mac-token");
    expect(jest.mocked(mdmAPI.deleteWindowsEULA)).not.toHaveBeenCalled();
    expect(onChange).toHaveBeenCalledTimes(1);
  });

  it("deletes once when Delete is pressed again while deleting", async () => {
    jest
      .mocked(mdmAPI.deleteWindowsEULA)
      .mockReturnValueOnce(new Promise(() => undefined));
    const { user } = renderSection({
      windows: {
        isAvailable: true,
        metadata: windowsMetadata,
        onChange: jest.fn(),
      },
    });

    await user.click(screen.getByRole("tab", { name: "Windows" }));
    await user.click(screen.getByRole("button", { name: "Delete EULA" }));
    await user.click(screen.getByRole("button", { name: "Delete" }));
    await user.keyboard("{Enter}");

    expect(jest.mocked(mdmAPI.deleteWindowsEULA)).toHaveBeenCalledTimes(1);
  });

  it("uploads a macOS EULA through the macOS endpoint and refetches", async () => {
    const onChange = jest.fn();
    const { user } = renderSection({
      darwin: { isAvailable: true, onChange },
    });
    const file = new File(["%PDF-1.7"], "eula.pdf", {
      type: "application/pdf",
    });

    await user.upload(getFileInput(), file);

    await waitFor(() => {
      expect(onChange).toHaveBeenCalledTimes(1);
    });
    expect(jest.mocked(mdmAPI.uploadEULA)).toHaveBeenCalledTimes(1);
    expect(jest.mocked(mdmAPI.uploadEULA)).toHaveBeenCalledWith(file);
    expect(jest.mocked(mdmAPI.uploadWindowsEULA)).not.toHaveBeenCalled();
  });

  it("previews the macOS EULA in a new tab", async () => {
    const open = jest.spyOn(window, "open").mockReturnValue(null);
    const { user } = renderSection({
      darwin: {
        isAvailable: true,
        metadata: macOSMetadata,
        onChange: jest.fn(),
      },
    });

    await user.click(screen.getByRole("button", { name: "Preview EULA" }));

    expect(open).toHaveBeenCalledTimes(1);
    expect(open).toHaveBeenCalledWith(
      "/api/latest/fleet/mdm/setup/eula/mac-token",
      "_blank"
    );
    open.mockRestore();
  });

  it("uploads a Windows agreement through the Windows endpoint and refetches", async () => {
    const onChange = jest.fn();
    const { user } = renderSection({
      windows: { isAvailable: true, onChange },
    });
    const file = new File(["# Terms"], "terms.md", { type: "text/markdown" });

    await user.click(screen.getByRole("tab", { name: "Windows" }));
    await user.upload(getFileInput(), file);

    await waitFor(() => {
      expect(onChange).toHaveBeenCalledTimes(1);
    });
    expect(jest.mocked(mdmAPI.uploadWindowsEULA)).toHaveBeenCalledTimes(1);
    expect(jest.mocked(mdmAPI.uploadWindowsEULA)).toHaveBeenCalledWith(file);
    expect(jest.mocked(mdmAPI.uploadEULA)).not.toHaveBeenCalled();
    expect(jest.mocked(notify.success)).toHaveBeenCalledWith(
      "Successfully uploaded."
    );
  });

  it("ignores another file while an upload is in flight", async () => {
    let finishUpload: (value: unknown) => void = () => undefined;
    jest.mocked(mdmAPI.uploadWindowsEULA).mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finishUpload = resolve;
        })
    );
    const { user } = renderSection();
    const file = new File(["# Terms"], "terms.md", { type: "text/markdown" });

    await user.click(screen.getByRole("tab", { name: "Windows" }));
    await user.upload(getFileInput(), file);
    await user.upload(getFileInput(), file);

    expect(jest.mocked(mdmAPI.uploadWindowsEULA)).toHaveBeenCalledTimes(1);
    finishUpload({});
    await waitFor(() => {
      expect(jest.mocked(notify.success)).toHaveBeenCalledTimes(1);
    });
  });

  it("rejects a Windows file that isn't markdown without uploading it", async () => {
    const onChange = jest.fn();
    const { user } = renderSection({
      windows: { isAvailable: true, onChange },
    });

    await user.click(screen.getByRole("tab", { name: "Windows" }));
    // Bypass the input's accept filter, like a drop or a picker set to show
    // all files.
    await userEvent.upload(getFileInput(), new File(["%PDF"], "terms.pdf"), {
      applyAccept: false,
    });

    await waitFor(() => {
      expect(jest.mocked(notify.error)).toHaveBeenCalledWith(
        "Couldn’t upload EULA. The file must be a markdown (.md) file."
      );
    });
    expect(jest.mocked(mdmAPI.uploadWindowsEULA)).not.toHaveBeenCalled();
    expect(onChange).not.toHaveBeenCalled();
  });

  it("rejects a Windows file over 512 KB without uploading it", async () => {
    const onChange = jest.fn();
    const { user } = renderSection({
      windows: { isAvailable: true, onChange },
    });

    await user.click(screen.getByRole("tab", { name: "Windows" }));
    await user.upload(
      getFileInput(),
      new File([new Uint8Array(512 * 1024 + 1)], "terms.md", {
        type: "text/markdown",
      })
    );

    await waitFor(() => {
      expect(jest.mocked(notify.error)).toHaveBeenCalledWith(
        "Couldn’t upload EULA. The file must be 512 KB or smaller."
      );
    });
    expect(jest.mocked(mdmAPI.uploadWindowsEULA)).not.toHaveBeenCalled();
    expect(onChange).not.toHaveBeenCalled();
  });

  it("shows the server's reason when a Windows upload is rejected", async () => {
    jest.mocked(mdmAPI.uploadWindowsEULA).mockRejectedValueOnce({
      data: {
        message: "Bad request",
        errors: [
          {
            name: "base",
            reason:
              "The file contains HTML. Convert it to markdown and upload again.",
          },
        ],
      },
    });
    const onChange = jest.fn();
    const { user } = renderSection({
      windows: { isAvailable: true, onChange },
    });

    await user.click(screen.getByRole("tab", { name: "Windows" }));
    await user.upload(
      getFileInput(),
      new File(["<div>Terms</div>"], "terms.md", { type: "text/markdown" })
    );

    await waitFor(() => {
      expect(jest.mocked(notify.error)).toHaveBeenCalledTimes(1);
    });
    expect(jest.mocked(notify.error)).toHaveBeenCalledWith(
      "Couldn’t upload EULA. The file contains HTML. Convert it to markdown and upload again.",
      expect.anything()
    );
    expect(onChange).not.toHaveBeenCalled();
  });

  it("reports a failed delete and still refetches", async () => {
    jest
      .mocked(mdmAPI.deleteWindowsEULA)
      .mockRejectedValueOnce({ status: 500 });
    const onChange = jest.fn();
    const { user } = renderSection({
      windows: {
        isAvailable: true,
        metadata: windowsMetadata,
        onChange,
      },
    });

    await user.click(screen.getByRole("tab", { name: "Windows" }));
    await user.click(screen.getByRole("button", { name: "Delete EULA" }));
    await user.click(screen.getByRole("button", { name: "Delete" }));

    await waitFor(() => {
      expect(onChange).toHaveBeenCalledTimes(1);
    });
    expect(jest.mocked(notify.error)).toHaveBeenCalledWith(
      "Couldn’t delete. Please try again.",
      expect.anything()
    );
    expect(jest.mocked(notify.success)).not.toHaveBeenCalled();
  });

  describe("in GitOps mode", () => {
    it("keeps the agreements viewable but disables delete", async () => {
      const { user } = renderSection(
        {
          darwin: {
            isAvailable: true,
            metadata: macOSMetadata,
            onChange: jest.fn(),
          },
          windows: {
            isAvailable: true,
            metadata: windowsMetadata,
            onChange: jest.fn(),
          },
        },
        GITOPS_CONFIG
      );

      expect(
        screen.getByRole("button", { name: "Preview EULA" })
      ).toBeEnabled();
      expect(
        screen.getByRole("button", { name: "Delete EULA" })
      ).toBeDisabled();

      await user.click(screen.getByRole("tab", { name: "Windows" }));

      expect(
        screen.getByRole("button", { name: "Example EULA" })
      ).toBeEnabled();
      expect(
        screen.getByRole("button", { name: "Delete EULA" })
      ).toBeDisabled();
    });

    it("disables uploading", async () => {
      const { user } = renderSection({}, GITOPS_CONFIG);

      expect(screen.getByRole("button", { name: "Upload" })).toBeDisabled();

      await user.click(screen.getByRole("tab", { name: "Windows" }));

      expect(screen.getByRole("button", { name: "Upload" })).toBeDisabled();
    });

    it("ignores a file that reaches the uploader anyway", async () => {
      const { user } = renderSection({}, GITOPS_CONFIG);

      await user.click(screen.getByRole("tab", { name: "Windows" }));
      await user.upload(getFileInput(), new File(["# Terms"], "terms.md"));

      expect(jest.mocked(mdmAPI.uploadWindowsEULA)).not.toHaveBeenCalled();
    });
  });
});
