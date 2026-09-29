import { AxiosResponse } from "axios";
import React, { useEffect, useState } from "react";

import Button from "components/buttons/Button";
import Checkbox from "components/forms/fields/Checkbox";
import InputField from "components/forms/fields/InputField";
import GitOpsModeTooltipWrapper from "components/GitOpsModeTooltipWrapper";
import { notify } from "components/ToastNotification";
import { IConfig, IIdentityProvider } from "interfaces/config";
import { expandErrorReasonRequired } from "interfaces/errors";
import SettingsSection from "pages/admin/components/SettingsSection";
import { sendRequest } from "services";
import configAPI from "services/entities/config";

const baseClass = "identity-provider-connections";

const emptyProvider = (): IIdentityProvider => ({
  name: "",
  entity_id: "",
  idp_name: "",
  metadata: "",
  metadata_url: "",
  default: false,
});

interface IIdentityProviderConnectionsProps {
  appConfig: IConfig;
}

const IdentityProviderConnections = ({
  appConfig,
}: IIdentityProviderConnectionsProps) => {
  const saved = appConfig.mdm.identity_providers ?? [];
  const [providers, setProviders] = useState<IIdentityProvider[]>(saved);
  useEffect(() => {
    setProviders(appConfig.mdm.identity_providers ?? []);
  }, [appConfig.mdm.identity_providers]);
  const [draft, setDraft] = useState<IIdentityProvider>(emptyProvider());
  const [isUpdating, setIsUpdating] = useState(false);
  const [issuedToken, setIssuedToken] = useState<{
    name: string;
    token: string;
  } | null>(null);

  const generateToken = async (name: string) => {
    setIsUpdating(true);
    try {
      const response = (await sendRequest(
        "POST",
        `/latest/fleet/identity_providers/${encodeURIComponent(
          name
        )}/scim_token`
      )) as { token: string };
      setIssuedToken({ name, token: response.token });
      notify.success(
        `SCIM token created for ${name}. Copy it now; it is not shown again.`
      );
    } catch (err) {
      notify.error("Couldn't create a SCIM token. Please try again.", {
        response: err,
      });
    } finally {
      setIsUpdating(false);
    }
  };

  const save = async (next: IIdentityProvider[]) => {
    setIsUpdating(true);
    try {
      await configAPI.update({
        mdm: { identity_providers: next },
      });
      setProviders(next);
      notify.success("Successfully updated identity providers.");
      return true;
    } catch (err) {
      const ae = (typeof err === "object" ? err : {}) as AxiosResponse;
      if (ae.status === 422) {
        notify.error(`Couldn't update: ${expandErrorReasonRequired(err)}.`, {
          response: err,
        });
      } else {
        notify.error("Couldn't update. Please try again.", { response: err });
      }
      return false;
    } finally {
      setIsUpdating(false);
    }
  };

  const addProvider = async () => {
    const name = draft.name.trim();
    if (
      !name ||
      !draft.entity_id.trim() ||
      (!draft.metadata_url.trim() && !draft.metadata.trim())
    ) {
      notify.error(
        "Name, entity ID, and metadata URL or metadata are required."
      );
      return;
    }
    const provider: IIdentityProvider = {
      ...draft,
      name,
      entity_id: draft.entity_id.trim(),
      metadata: draft.metadata.trim(),
      metadata_url: draft.metadata_url.trim(),
      idp_name: draft.idp_name.trim() || name,
      default: draft.default,
    };
    if (providers.some((existing) => existing.name === name)) {
      notify.error(`An identity provider named "${name}" already exists.`);
      return;
    }
    const next = providers.map((existing) =>
      provider.default ? { ...existing, default: false } : existing
    );
    if (await save([...next, provider])) {
      setDraft(emptyProvider());
    }
  };

  return (
    <SettingsSection title="Identity providers">
      <p>
        Connect one identity provider for the organization, or several and
        assign each fleet its own. Fleets that do not pick one use the
        connection marked as the organization default. Fleet console sign-in is
        unchanged.
      </p>
      {providers.length === 0 ? (
        <p>No additional identity providers.</p>
      ) : (
        <ul className={`${baseClass}__list`}>
          {providers.map((provider) => (
            <li key={provider.name}>
              <span>
                {provider.name}
                {provider.default ? " (organization default)" : ""}
              </span>
              <GitOpsModeTooltipWrapper
                renderChildren={(disableChildren) => (
                  <>
                    <Button
                      variant="link"
                      disabled={disableChildren || isUpdating}
                      onClick={() => generateToken(provider.name)}
                    >
                      Generate SCIM token
                    </Button>
                    <Button
                      variant="link"
                      disabled={disableChildren || isUpdating}
                      onClick={() =>
                        save(
                          providers.filter(
                            (item) => item.name !== provider.name
                          )
                        )
                      }
                    >
                      Remove
                    </Button>
                  </>
                )}
              />
            </li>
          ))}
        </ul>
      )}
      {issuedToken && (
        <p>
          SCIM bearer token for {issuedToken.name}:{" "}
          <code>{issuedToken.token}</code>. Send it to{" "}
          <code>/api/latest/fleet/scim</code>. It is shown only once. A Fleet
          API token still writes only the organization default directory.
        </p>
      )}
      <GitOpsModeTooltipWrapper
        renderChildren={(disableChildren) => (
          <form
            onSubmit={(event) => {
              event.preventDefault();
              addProvider();
            }}
          >
            <InputField
              label="Name"
              value={draft.name}
              disabled={disableChildren}
              onChange={(value) =>
                setDraft((prev) => ({ ...prev, name: value }))
              }
              helpText="Fleets reference this name."
            />
            <InputField
              label="Entity ID"
              value={draft.entity_id}
              disabled={disableChildren}
              onChange={(value) =>
                setDraft((prev) => ({ ...prev, entity_id: value }))
              }
            />
            <InputField
              label="Metadata URL"
              value={draft.metadata_url}
              disabled={disableChildren}
              onChange={(value) =>
                setDraft((prev) => ({ ...prev, metadata_url: value }))
              }
              helpText="Metadata URL or metadata is required."
            />
            <InputField
              label="Metadata"
              type="textarea"
              value={draft.metadata}
              disabled={disableChildren}
              onChange={(value) =>
                setDraft((prev) => ({ ...prev, metadata: value }))
              }
            />
            <Checkbox
              value={draft.default}
              disabled={disableChildren}
              onChange={(value: boolean) =>
                setDraft((prev) => ({ ...prev, default: value }))
              }
            >
              Organization default
            </Checkbox>
            <Button
              type="submit"
              disabled={disableChildren || isUpdating}
              isLoading={isUpdating}
            >
              Add identity provider
            </Button>
          </form>
        )}
      />
    </SettingsSection>
  );
};

export default IdentityProviderConnections;
