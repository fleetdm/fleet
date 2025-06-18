## Initial overview + notes

Data need to know:
  !!idpConfigured
  ```
  const { data: scimIdPDetails, isLoading, isError } = useQuery(
    ["scim_details"],
    () => idpAPI.getSCIMDetails(),
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
    }
  );
  ```
  `scimIdPDetails.last_request.requested_at`


Data need to send:

  ...
  "If `query`, `criteria`, and `hosts` aren't specified, a manual label with no hosts will be created.
  ...

  POST /labels
    {
      ...
      criteria: {
        vital: "end_user_idp_group" | "end_user_idp_department";
        value: string      
      }
    }

UI suggestions:

- ["IdP is not configured in integration
  settings."](https://www.figma.com/design/cGSVuzQvRaF4uHejqpM74K/-23899-Add-labels-based-on-end-user-s-IdP-information?node-id=5415-16469&t=ZxZOyk7eOIChor1A-1):
  this language is a bit confusing, sounds like "integration settings is not the place where you
  configure IdP" suggestion: "Configure IdP in a<href="<path>">integration settings</a>.

Questions
- What is relationship between IdP and Mdm? Is MdM required for this feature? Specifically, what
  tells the UI if "IdP is configured":
    - `config.mdm.end_user_authentication.idp_name`? 
    - `config.sso_settings.idp_name`?
    - `/GET .../scim/details response.last_requested.requested_at` seems to be what's used by
      `IdentityProviders.tsx` card - assume this until informed otherwise