import React, { useContext } from "react";
import { syntaxHighlight } from "utilities/helpers";

import { AppContext } from "context/app";
import { IPolicyWebhookPreviewPayload } from "interfaces/policy";

const baseClass = "example-payload";

interface IHostPreview {
  id: number;
  display_name: string;
  url: string;
}

interface IExamplePayload {
  timestamp: string;
  policy: IPolicyWebhookPreviewPayload;
  hosts: IHostPreview[];
}

const ExamplePayload = (): JSX.Element => {
  const { isFreeTier } = useContext(AppContext);

  const json: IExamplePayload = {
    timestamp: "0000-00-00T00:00:00Z",
    policy: {
      id: 1,
      name: "Is Gatekeeper enabled?",
      query: "SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;",
      description: "Checks if gatekeeper is enabled on macOS devices.",
      author_id: 1,
      author_name: "John",
      author_email: "john@example.com",
      resolution: "Turn on Gatekeeper feature in System Preferences.",
      passing_host_count: 2000,
      failing_host_count: 300,
      critical: false,
    },
    hosts: [
      {
        id: 1,
        display_name: "macbook-1",
        url: "https://fleet.example.com/hosts/1",
      },
      {
        id: 2,
        display_name: "macbbook-2",
        url: "https://fleet.example.com/hosts/2",
      },
    ],
  };
  if (isFreeTier) {
    delete json.policy.critical;
  }

  return (
    <div className={baseClass}>
      <pre>POST https://server.com/example</pre>
      <pre dangerouslySetInnerHTML={{ __html: syntaxHighlight(json) }} />
    </div>
  );
};

export default ExamplePayload;
