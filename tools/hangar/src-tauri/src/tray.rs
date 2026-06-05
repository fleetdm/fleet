use serde::Deserialize;
use tauri::{
    menu::{Menu, MenuItem, PredefinedMenuItem},
    tray::{MouseButton, MouseButtonState, TrayIconBuilder, TrayIconEvent},
    AppHandle, Emitter, Manager,
};

// Mirror of the frontend-side tray state. Frontend pushes this via the
// update_tray command whenever the relevant slices of state change
// (branch info, health probes, proc list). The backend rebuilds the
// menu in place — no polling, no duplicated state.
#[derive(Debug, Clone, Deserialize, Default)]
pub struct TrayState {
    pub branch: Option<String>,
    pub serve_up: bool,
    pub docker_up: bool,
    pub ngrok_running: bool,
    pub python_running: bool,
}

const TRAY_ID: &str = "main";

fn dot(on: bool) -> &'static str {
    // Native macOS menus can't animate or color-style arbitrary text, so
    // we lean on emoji to get the "this is live" green color signal —
    // closest we can get to the pulsing dot in the main UI.
    if on { "🟢" } else { "⚪" }
}

/// Single-column label: dot + name, with an optional extra (e.g. port).
/// The dot already encodes up/down, so we don't repeat it in words —
/// which side-steps the proportional-font alignment problem entirely.
fn svc_label(on: bool, name: &str, extra: Option<&str>) -> String {
    match extra {
        Some(x) => format!("{}  {}  ·  {}", dot(on), name, x),
        None => format!("{}  {}", dot(on), name),
    }
}

fn build_menu(
    app: &AppHandle,
    state: &TrayState,
) -> tauri::Result<Menu<tauri::Wry>> {
    // --- Branch row ---
    let branch_label = match &state.branch {
        Some(b) => format!("Branch: {b}"),
        None => "No repo configured".into(),
    };
    let branch_item =
        MenuItem::with_id(app, "tray:branch", &branch_label, false, None::<&str>)?;

    let sep1 = PredefinedMenuItem::separator(app)?;

    // --- "What's running" rows (disabled / informational) ---
    let svc_serve = MenuItem::with_id(
        app,
        "tray:svc-serve",
        // Keep the port when up because it's real info the dot doesn't
        // already convey; drop it (and any status word) when down.
        &svc_label(state.serve_up, "fleet serve", state.serve_up.then_some(":8080")),
        false,
        None::<&str>,
    )?;
    let svc_docker = MenuItem::with_id(
        app,
        "tray:svc-docker",
        &svc_label(state.docker_up, "docker", None),
        false,
        None::<&str>,
    )?;
    let svc_ngrok = MenuItem::with_id(
        app,
        "tray:svc-ngrok",
        &svc_label(state.ngrok_running, "ngrok", None),
        false,
        None::<&str>,
    )?;
    let svc_python = MenuItem::with_id(
        app,
        "tray:svc-python",
        &svc_label(state.python_running, "python", None),
        false,
        None::<&str>,
    )?;

    let sep2 = PredefinedMenuItem::separator(app)?;

    // --- Start/Stop all toggle ---
    let any_running = state.serve_up
        || state.docker_up
        || state.ngrok_running
        || state.python_running;
    let start_stop = if any_running {
        MenuItem::with_id(app, "tray:stop-all", "■ Stop all", true, None::<&str>)?
    } else {
        MenuItem::with_id(
            app,
            "tray:start-all",
            "▶ Start all",
            state.branch.is_some(),
            None::<&str>,
        )?
    };

    let show = MenuItem::with_id(app, "tray:show", "Open Fleet Dev", true, None::<&str>)?;
    let quit = MenuItem::with_id(app, "tray:quit", "Quit", true, None::<&str>)?;

    Menu::with_items(
        app,
        &[
            &branch_item,
            &sep1,
            &svc_serve,
            &svc_docker,
            &svc_ngrok,
            &svc_python,
            &sep2,
            &start_stop,
            &show,
            &quit,
        ],
    )
}

/// Bring the main window back from the dead in every reasonable
/// state: hidden via close-to-tray, minimized by the user, or just not
/// focused. macOS doesn't always re-show a hidden window from a bare
/// `show()`, so we also call `unminimize` and `set_focus` defensively.
/// Activation policy stays Regular at all times — see lib.rs for why
/// we no longer flip to Accessory on hide.
pub fn show_main_window(app: &AppHandle) {
    if let Some(w) = app.get_webview_window("main") {
        let _ = w.unminimize();
        let _ = w.show();
        let _ = w.set_focus();
    }
}

pub fn build(app: &AppHandle) -> tauri::Result<()> {
    let initial_menu = build_menu(app, &TrayState::default())?;
    let icon = tauri::include_image!("icons/32x32.png");

    TrayIconBuilder::with_id(TRAY_ID)
        .icon(icon)
        // Template mode renders the icon as a white silhouette in the
        // menu bar (system foreground color, auto-inverted on light
        // bars). It uses only the alpha channel, so the colorful logo
        // works fine here — macOS reads "where is the PNG opaque" and
        // fills that with white, ignoring colors entirely. Matches the
        // rest of the macOS menu bar where every icon is monochrome.
        .icon_as_template(true)
        .menu(&initial_menu)
        // Left-click opens the main window; right-click shows the menu
        // (macOS's NSStatusItem still surfaces the attached menu on
        // right-click even when left-click is intercepted). The
        // "Open Fleet Dev" menu item stays as a backup for users who
        // always go through the menu.
        .show_menu_on_left_click(false)
        .on_menu_event(|app, event| match event.id.as_ref() {
            "tray:show" => {
                show_main_window(app);
            }
            "tray:quit" => {
                // Surface the window so the user sees the confirm modal
                // even if the app was tray-only. Frontend handles the
                // confirm flow and calls shutdown_now when ready; no
                // timer here because we want Cancel to truly cancel.
                show_main_window(app);
                let _ = app.emit("app:quit-requested", ());
            }
            // Everything else routes to the frontend, which owns the
            // orchestration (start/stop chain, git fetch/pull).
            id if id.starts_with("tray:") => {
                let _ = app.emit(id, ());
            }
            _ => {}
        })
        .on_tray_icon_event(|tray, event| {
            // We act on the Up edge of left-click so the window doesn't
            // open mid-press if the user is about to drag (which macOS
            // uses for moving the icon when Cmd is held).
            if let TrayIconEvent::Click {
                button: MouseButton::Left,
                button_state: MouseButtonState::Up,
                ..
            } = event
            {
                show_main_window(&tray.app_handle().clone());
            }
        })
        .build(app)?;

    Ok(())
}

#[tauri::command(rename_all = "snake_case")]
pub fn update_tray(app: AppHandle, state: TrayState) -> Result<(), String> {
    let menu = build_menu(&app, &state).map_err(|e| e.to_string())?;
    let Some(tray) = app.tray_by_id(TRAY_ID) else {
        return Err("tray not initialized".into());
    };
    tray.set_menu(Some(menu)).map_err(|e| e.to_string())?;
    Ok(())
}
