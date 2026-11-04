import { screen } from "@testing-library/react";
import { noop } from "lodash";
import React from "react";

import { createCustomRenderer } from "test/test-utils";

import BootstrapPackageUploader from "./BootstrapPackageUploader";

describe("BootstrapPackageUploader", () => {
  it("disables the upload button in GitOps mode", () => {
    const render = createCustomRenderer({
      context: {
        app: {
          config: {
            gitops: {
              gitops_mode_enabled: true,
              repository_url: "https://example.com/repo",
            },
          },
        },
      },
    });

    render(<BootstrapPackageUploader currentTeamId={0} onUpload={noop} />);

    expect(screen.getByRole("button", { name: "Upload" })).toBeDisabled();
  });
});
