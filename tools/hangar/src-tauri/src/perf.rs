use serde::Serialize;

/// The hardcoded set of OS templates the osquery-perf agent will
/// accept. Mirrors the `validTemplateNames` map in
/// `cmd/osquery-perf/agent.go` (search for that symbol upstream when
/// fleet updates). The agent only takes templates it has embedded —
/// listing anything else would just produce a "Invalid template name"
/// error on launch, so we don't bother enumerating the filesystem.
///
/// Each entry is the bare template id (no `.tmpl` suffix — the agent
/// accepts either form but bare ids read cleaner in the form preview).
#[derive(Debug, Clone, Serialize)]
pub struct PerfTemplate {
    pub id: String,
    pub label: String,
    pub version: String,
    /// True for templates the agent treats as mobile (no enroll secret
    /// — these enroll via MDM only). UI uses this to gray out the
    /// enroll-secret requirement when *only* mobile templates are
    /// selected.
    pub mobile: bool,
    /// True for Apple platforms (macOS, iOS, iPadOS). MDM enrollment on
    /// Apple needs a SCEP challenge; Windows MDM does not. UI uses this
    /// to require the SCEP field only when an Apple template is in the
    /// run.
    pub apple: bool,
}

pub fn templates() -> Vec<PerfTemplate> {
    // Order: desktop first, then mobile, alphabetical within each
    // group. Friendly labels reflect the agent's actual platform
    // mapping (e.g. iphone_* reports `platform=ios`).
    //                                                  mobile  apple
    vec![
        t("macos_13.6.2", "macOS", "13 Ventura", false, true),
        t("macos_14.1.2", "macOS", "14 Sonoma", false, true),
        t("windows_11", "Windows", "11", false, false),
        t("windows_11_22H2_2861", "Windows", "11 22H2 (build 2861)", false, false),
        t("windows_11_22H2_3007", "Windows", "11 22H2 (build 3007)", false, false),
        t("ubuntu_22.04", "Ubuntu", "22.04 LTS", false, false),
        t("rhel_8", "RHEL", "8", false, false),
        t("rhel_9", "RHEL", "9", false, false),
        t("rhel_10", "RHEL", "10", false, false),
        t("iphone_14.6", "iOS", "14.6", true, true),
        t("iphone_17", "iOS", "17", true, true),
        t("ipad_13.18", "iPadOS", "13.18", true, true),
    ]
}

fn t(id: &str, label: &str, version: &str, mobile: bool, apple: bool) -> PerfTemplate {
    PerfTemplate {
        id: id.into(),
        label: label.into(),
        version: version.into(),
        mobile,
        apple,
    }
}

#[tauri::command]
pub fn perf_list_templates() -> Vec<PerfTemplate> {
    templates()
}
