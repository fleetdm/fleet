## Initial overview + notes

Data need to know:
  !!idpConfigured  - from `/GET .../scim/details response.last_requested.requested_at`
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
    Future may look like:
      name: Engineering department or IT admins who are named Ricky
  description: Hosts with end users in engineering or who are IT admins named Ricky
  label_membership_type: host_vitals

  criteria:
    or:  
      - and:
        - vital: end_user_idp_groups
          value: IT admins
        - vital: end_user_first_name
          value: Ricky
      - vital: end_user_idp_department
        value: Engineering -->

UI suggestions:

- ["IdP is not configured in integration
  settings."](https://www.figma.com/design/cGSVuzQvRaF4uHejqpM74K/-23899-Add-labels-based-on-end-user-s-IdP-information?node-id=5415-16469&t=ZxZOyk7eOIChor1A-1):
  this language is a bit confusing, sounds like "integration settings is not the place where you
  configure IdP" suggestion: "Configure IdP in a<href="<path>">integration settings</a>.



  Choices made:
  - This will need to be used for the Edit Label flow as well, so should be easily extensible in
    that direction. Question is do I make a router-level page for each one (like current), which
    renders the same `UniversalLabelForm` component, or have one router-level page rendered
    conditionally in edit or new mode. **Decision: 2 router-level pages**
  - Abstract form to a unviersal form now or implement on NewLabelPage? **Decision: imlpement directly on page** - intricacies of query side panel rendering outside of main content and
    dependence on form state make this the right choice for now. Also since I am still only
    *assuming* we'll apply this to the edit query page as well, I'll leave the abstraction work for
    when that is specced and prioritized.
  - organize NewLabelPage states by function instead of all together


  Misc TODO:
  - finalize URL routes
  - update API call
  - clean up styling
  - tests