import React, { useContext, useMemo, useState } from "react";

import SettingsSection from "pages/admin/components/SettingsSection";
import PageDescription from "components/PageDescription";
import Button from "components/buttons/Button";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import InputField from "components/forms/fields/InputField";
import Icon from "components/Icon/Icon";

import { NotificationContext } from "context/notification";
import configAPI from "services/entities/config";
import {
  ISlackNotificationRoute,
} from "interfaces/integration";

import { IAppConfigFormProps } from "pages/admin/OrgSettingsPage/cards/constants";

const baseClass = "slack-notifications";

// Mirrors server/fleet/notification.go AllNotificationCategories, plus the
// route-only wildcard "all" at the top for admins who want one webhook that
// receives every Fleet notification.
const CATEGORY_OPTIONS = [
  { label: "All categories", value: "all" },
  { label: "MDM", value: "mdm" },
  { label: "License", value: "license" },
  { label: "Vulnerabilities", value: "vulnerabilities" },
  { label: "Policies", value: "policies" },
  { label: "Software", value: "software" },
  { label: "Hosts", value: "hosts" },
  { label: "Integrations", value: "integrations" },
  { label: "System", value: "system" },
];

const WEBHOOK_PREFIX = "https://hooks.slack.com/";

const emptyRoute = (): ISlackNotificationRoute => ({
  category: "vulnerabilities",
  webhook_url: "",
});

const SlackNotifications = ({
  appConfig,
}: IAppConfigFormProps): JSX.Element => {
  const { renderFlash } = useContext(NotificationContext);

  const initialRoutes = useMemo<ISlackNotificationRoute[]>(
    () =>
      appConfig.integrations.slack_notifications?.routes?.map((r) => ({
        ...r,
      })) ?? [],
    [appConfig]
  );

  const [routes, setRoutes] = useState<ISlackNotificationRoute[]>(
    initialRoutes
  );
  const [rowErrors, setRowErrors] = useState<Record<number, string>>({});
  const [isSaving, setIsSaving] = useState(false);

  const addRoute = () => setRoutes((prev) => [...prev, emptyRoute()]);

  const removeRoute = (idx: number) =>
    setRoutes((prev) => prev.filter((_, i) => i !== idx));

  const updateRoute = (
    idx: number,
    patch: Partial<ISlackNotificationRoute>
  ) => {
    setRoutes((prev) =>
      prev.map((r, i) => (i === idx ? { ...r, ...patch } : r))
    );
  };

  const validate = (): boolean => {
    const errs: Record<number, string> = {};
    const seen = new Set<string>();
    routes.forEach((r, i) => {
      const url = r.webhook_url.trim();
      if (!url.startsWith(WEBHOOK_PREFIX)) {
        errs[i] = `Webhook URL must start with ${WEBHOOK_PREFIX}`;
        return;
      }
      const key = `${r.category}|${url}`;
      if (seen.has(key)) {
        errs[i] = "Duplicate (category, webhook URL) row";
        return;
      }
      seen.add(key);
    });
    setRowErrors(errs);
    return Object.keys(errs).length === 0;
  };

  const onSave = async (evt: React.MouseEvent) => {
    evt.preventDefault();
    if (!validate()) return;
    const trimmedRoutes = routes.map((r) => ({
      category: r.category,
      webhook_url: r.webhook_url.trim(),
    }));

    // Bypass the shared deepDifference-driven save used by the rest of the
    // Integrations page: for arrays, deepDifference uses lodash's
    // differenceWith which only returns elements present in the new value
    // but not the old, collapsing an added/edited row into a partial array
    // and replacing the stored routes with just the delta. Sending the
    // full routes array directly is the simplest fix and keeps the Slack
    // card's behavior independent of future refactors to that utility.
    setIsSaving(true);
    try {
      await configAPI.update({
        integrations: {
          slack_notifications: { routes: trimmedRoutes },
        },
      });
      renderFlash("success", "Slack notification routes saved.");
    } catch (err: unknown) {
      renderFlash("error", "Could not save Slack notification routes.");
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <SettingsSection title="Slack notifications" className={baseClass}>
      <PageDescription content="Route in-app notifications to Slack by category. Each row posts Fleet's system notifications for the selected category to its incoming-webhook URL." />

      <div className={`${baseClass}__routes`}>
        {routes.length === 0 && (
          <p className={`${baseClass}__empty`}>
            No routes configured. Add one below to start delivering to Slack.
          </p>
        )}
        {routes.map((r, idx) => (
          // eslint-disable-next-line react/no-array-index-key
          <div key={idx} className={`${baseClass}__row`}>
            <Dropdown
              className={`${baseClass}__category`}
              label={idx === 0 ? "Category" : undefined}
              options={CATEGORY_OPTIONS}
              value={r.category}
              onChange={(value: string) =>
                updateRoute(idx, { category: value })
              }
              searchable={false}
            />
            <InputField
              label={idx === 0 ? "Slack incoming-webhook URL" : undefined}
              placeholder={WEBHOOK_PREFIX}
              value={r.webhook_url}
              onChange={(value: string) =>
                updateRoute(idx, { webhook_url: value })
              }
              error={rowErrors[idx]}
              name={`slack-webhook-${idx}`}
            />
            <Button
              variant="text-icon"
              onClick={() => removeRoute(idx)}
              className={`${baseClass}__remove`}
            >
              <Icon name="close" />
              Remove
            </Button>
          </div>
        ))}
      </div>

      <div className={`${baseClass}__actions`}>
        <Button variant="text-icon" onClick={addRoute}>
          <Icon name="plus" />
          Add route
        </Button>
        <Button
          variant="default"
          onClick={onSave}
          disabled={isSaving}
        >
          Save
        </Button>
      </div>
    </SettingsSection>
  );
};

export default SlackNotifications;
