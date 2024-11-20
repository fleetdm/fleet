import {
  generateActions,
  DEFAULT_ACTION_OPTIONS,
  generateActionsProps,
} from "./HostSoftwareTableConfig";

describe("generateActions", () => {
  const defaultProps: generateActionsProps = {
    userHasSWWritePermission: true,
    hostScriptsEnabled: true,
    hostCanWriteSoftware: true,
    softwareIdActionPending: null,
    softwareId: 1,
    status: null,
    software_package: null,
    app_store_app: null,
    hostMDMEnrolled: false,
  };

  it("returns default actions when user has write permission and scripts are enabled", () => {
    const actions = generateActions(defaultProps);
    expect(actions).toEqual(DEFAULT_ACTION_OPTIONS);
  });

  it("removes install and uninstall actions when user has no write permission", () => {
    const props = { ...defaultProps, userHasSWWritePermission: false };
    const actions = generateActions(props);
    expect(actions.find((a) => a.value === "install")).toBeUndefined();
    expect(actions.find((a) => a.value === "uninstall")).toBeUndefined();
  });

  it("disables install and uninstall actions when host scripts are disabled", () => {
    const props = { ...defaultProps, hostScriptsEnabled: false };
    const actions = generateActions(props);
    expect(actions.find((a) => a.value === "install")?.disabled).toBe(true);
    expect(actions.find((a) => a.value === "uninstall")?.disabled).toBe(true);
  });

  it("disables install and uninstall actions when locally pending (waiting for API response)", () => {
    const props = {
      ...defaultProps,
      softwareIdActionPending: 1,
      softwareId: 1,
    };
    const actions = generateActions(props);
    expect(actions.find((a) => a.value === "install")?.disabled).toBe(true);
    expect(actions.find((a) => a.value === "uninstall")?.disabled).toBe(true);
  });

  it("disables install and uninstall actions when pending install status", () => {
    const props: generateActionsProps = {
      ...defaultProps,
      status: "pending_install",
    };
    const actions = generateActions(props);
    expect(actions.find((a) => a.value === "install")?.disabled).toBe(true);
    expect(actions.find((a) => a.value === "uninstall")?.disabled).toBe(true);
  });

  it("disables install and uninstall actions when pending uninstall status", () => {
    const props: generateActionsProps = {
      ...defaultProps,
      status: "pending_uninstall",
    };
    const actions = generateActions(props);
    expect(actions.find((a) => a.value === "install")?.disabled).toBe(true);
    expect(actions.find((a) => a.value === "uninstall")?.disabled).toBe(true);
  });

  it("removes uninstall action for VPP apps", () => {
    const props: generateActionsProps = {
      ...defaultProps,
      app_store_app: {
        app_store_id: "1",
        self_service: false,
        icon_url: "",
        version: "",
        last_install: { command_uuid: "", installed_at: "" },
      },
    };
    const actions = generateActions(props);
    expect(actions.find((a) => a.value === "uninstall")).toBeUndefined();
  });

  it("allows to install VPP apps even if scripts are disabled", () => {
    const props: generateActionsProps = {
      ...defaultProps,
      hostMDMEnrolled: true,
      hostScriptsEnabled: false,
      app_store_app: {
        app_store_id: "1",
        self_service: false,
        icon_url: "",
        version: "",
        last_install: { command_uuid: "", installed_at: "" },
      },
    };
    const actions = generateActions(props);
    expect(actions.find((a) => a.value === "install")?.disabled).toBe(false);
    expect(actions.find((a) => a.value === "uninstall")).toBeUndefined();
  });

  it("disables installing VPP app if host is not MDM enrolled", () => {
    const props: generateActionsProps = {
      ...defaultProps,
      hostScriptsEnabled: false,
      app_store_app: {
        app_store_id: "1",
        self_service: false,
        icon_url: "",
        version: "",
        last_install: { command_uuid: "", installed_at: "" },
      },
    };
    const actions = generateActions(props);
    expect(actions.find((a) => a.value === "install")?.disabled).toBe(true);
    expect(actions.find((a) => a.value === "uninstall")).toBeUndefined();
  });
});
